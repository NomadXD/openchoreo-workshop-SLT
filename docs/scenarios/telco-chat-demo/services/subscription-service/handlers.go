package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type server struct {
	db *pgxpool.Pool
}

// ---- plans ----

func (s *server) listPlans(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, data_gb, price_cents FROM plans WHERE active = true ORDER BY price_cents ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list plans")
		return
	}
	defer rows.Close()

	plans := []Plan{}
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DataGb, &p.PriceCents); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read plan")
			return
		}
		plans = append(plans, p)
	}
	writeJSON(w, http.StatusOK, plans)
}

func (s *server) createPlan(w http.ResponseWriter, r *http.Request) {
	var in Plan
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.ID == "" || in.Name == "" {
		writeError(w, http.StatusBadRequest, "id and name are required")
		return
	}

	_, err := s.db.Exec(r.Context(),
		`INSERT INTO plans (id, name, data_gb, price_cents, active) VALUES ($1, $2, $3, $4, true)`,
		in.ID, in.Name, in.DataGb, in.PriceCents)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create plan")
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

func (s *server) updatePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in Plan
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := s.db.Exec(r.Context(),
		`UPDATE plans SET name = $1, data_gb = $2, price_cents = $3 WHERE id = $4`,
		in.Name, in.DataGb, in.PriceCents, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update plan")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "plan not found")
		return
	}

	in.ID = id
	writeJSON(w, http.StatusOK, in)
}

func (s *server) deletePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	tag, err := s.db.Exec(r.Context(), `UPDATE plans SET active = false WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete plan")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "plan not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- customers ----

func (s *server) listCustomers(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	var rows pgx.Rows
	var err error
	if search == "" {
		rows, err = s.db.Query(r.Context(),
			`SELECT id, name, msisdn, email FROM customers ORDER BY id ASC`)
	} else {
		pattern := "%" + search + "%"
		rows, err = s.db.Query(r.Context(),
			`SELECT id, name, msisdn, email FROM customers
			 WHERE id ILIKE $1 OR name ILIKE $1 OR msisdn ILIKE $1
			 ORDER BY id ASC`, pattern)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list customers")
		return
	}
	defer rows.Close()

	customers := []Customer{}
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Msisdn, &c.Email); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read customer")
			return
		}
		customers = append(customers, c)
	}
	writeJSON(w, http.StatusOK, customers)
}

func (s *server) getCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var c Customer
	err := s.db.QueryRow(r.Context(),
		`SELECT id, name, msisdn, email FROM customers WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.Msisdn, &c.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "customer not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get customer")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ---- subscriptions ----

func (s *server) getSubscription(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !s.customerExists(r, id, w) {
		return
	}

	sub, status, errMsg := s.fetchSubscription(r, id)
	if errMsg != "" {
		writeError(w, status, errMsg)
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (s *server) setSubscription(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var in struct {
		PlanID string `json:"planId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !s.customerExists(r, id, w) {
		return
	}

	var active bool
	err := s.db.QueryRow(r.Context(), `SELECT active FROM plans WHERE id = $1`, in.PlanID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "plan not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up plan")
		return
	}
	if !active {
		writeError(w, http.StatusBadRequest, "plan is inactive")
		return
	}

	_, err = s.db.Exec(r.Context(),
		`INSERT INTO subscriptions (customer_id, plan_id, updated_at) VALUES ($1, $2, now())
		 ON CONFLICT (customer_id) DO UPDATE SET plan_id = $2, updated_at = now()`,
		id, in.PlanID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update subscription")
		return
	}

	sub, status, errMsg := s.fetchSubscription(r, id)
	if errMsg != "" {
		writeError(w, status, errMsg)
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (s *server) customerExists(r *http.Request, id string, w http.ResponseWriter) bool {
	var exists bool
	err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up customer")
		return false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "customer not found")
		return false
	}
	return true
}

func (s *server) fetchSubscription(r *http.Request, customerID string) (Subscription, int, string) {
	var sub Subscription
	sub.CustomerID = customerID

	err := s.db.QueryRow(r.Context(),
		`SELECT p.id, p.name, p.data_gb, p.price_cents
		 FROM subscriptions s JOIN plans p ON p.id = s.plan_id
		 WHERE s.customer_id = $1`, customerID).
		Scan(&sub.Plan.ID, &sub.Plan.Name, &sub.Plan.DataGb, &sub.Plan.PriceCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, http.StatusNotFound, "subscription not found"
	}
	if err != nil {
		return Subscription{}, http.StatusInternalServerError, "failed to get subscription"
	}
	return sub, 0, ""
}
