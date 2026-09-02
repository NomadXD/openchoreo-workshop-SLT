package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type server struct {
	db *pgxpool.Pool
}

// ---- usage ----

func (s *server) getUsage(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		writeError(w, http.StatusBadRequest, "date query parameter is required (YYYY-MM-DD)")
		return
	}

	var u UsageRecord
	var d time.Time
	err := s.db.QueryRow(r.Context(),
		`SELECT customer_id, date, browsing_mb, streaming_mb, social_mb, other_mb
		 FROM usage_records WHERE customer_id = $1 AND date = $2`, customerID, date).
		Scan(&u.CustomerID, &d, &u.BrowsingMb, &u.StreamingMb, &u.SocialMb, &u.OtherMb)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "no usage record for that date")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get usage")
		return
	}
	u.Date = d.Format("2006-01-02")
	u.TotalMb = u.BrowsingMb + u.StreamingMb + u.SocialMb + u.OtherMb
	writeJSON(w, http.StatusOK, u)
}

// getUsageHistory returns the most recent `days` usage records for a
// customer (default 7, capped at 30), oldest first — enough for a simple
// dashboard sparkline/table without the caller looping single-date lookups.
func (s *server) getUsageHistory(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	days := 7
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			days = n
		}
	}
	if days > 30 {
		days = 30
	}

	rows, err := s.db.Query(r.Context(),
		`SELECT customer_id, date, browsing_mb, streaming_mb, social_mb, other_mb
		 FROM usage_records WHERE customer_id = $1
		 ORDER BY date DESC LIMIT $2`, customerID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list usage history")
		return
	}
	defer rows.Close()

	records := []UsageRecord{}
	for rows.Next() {
		var u UsageRecord
		var d time.Time
		if err := rows.Scan(&u.CustomerID, &d, &u.BrowsingMb, &u.StreamingMb, &u.SocialMb, &u.OtherMb); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read usage record")
			return
		}
		u.Date = d.Format("2006-01-02")
		u.TotalMb = u.BrowsingMb + u.StreamingMb + u.SocialMb + u.OtherMb
		records = append(records, u)
	}
	// oldest first, easier for a caller to render left-to-right
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	writeJSON(w, http.StatusOK, records)
}

// ---- reports ----

func (s *server) createReport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CustomerID  string `json:"customerId"`
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.CustomerID == "" || in.Category == "" || in.Description == "" {
		writeError(w, http.StatusBadRequest, "customerId, category, and description are required")
		return
	}

	id, err := newReportID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate report id")
		return
	}

	var rep ServiceReport
	err = s.db.QueryRow(r.Context(),
		`INSERT INTO service_reports (id, customer_id, category, description, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'open', now(), now())
		 RETURNING id, customer_id, category, description, status, resolution_notes, created_at, updated_at`,
		id, in.CustomerID, in.Category, in.Description).
		Scan(&rep.ID, &rep.CustomerID, &rep.Category, &rep.Description, &rep.Status, &rep.ResolutionNotes, &rep.CreatedAt, &rep.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create report")
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

func (s *server) listReports(w http.ResponseWriter, r *http.Request) {
	customerID := strings.TrimSpace(r.URL.Query().Get("customerId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	excludeID := strings.TrimSpace(r.URL.Query().Get("excludeId"))

	query := `SELECT id, customer_id, category, description, status, resolution_notes, created_at, updated_at
			   FROM service_reports WHERE 1=1`
	args := []interface{}{}
	if customerID != "" {
		args = append(args, customerID)
		query += fmt.Sprintf(" AND customer_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if category != "" {
		args = append(args, category)
		query += fmt.Sprintf(" AND category = $%d", len(args))
	}
	if excludeID != "" {
		args = append(args, excludeID)
		query += fmt.Sprintf(" AND id != $%d", len(args))
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list reports")
		return
	}
	defer rows.Close()

	reports := []ServiceReport{}
	for rows.Next() {
		var rep ServiceReport
		if err := rows.Scan(&rep.ID, &rep.CustomerID, &rep.Category, &rep.Description, &rep.Status, &rep.ResolutionNotes, &rep.CreatedAt, &rep.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read report")
			return
		}
		reports = append(reports, rep)
	}
	writeJSON(w, http.StatusOK, reports)
}

func (s *server) getReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var rep ServiceReport
	err := s.db.QueryRow(r.Context(),
		`SELECT id, customer_id, category, description, status, resolution_notes, created_at, updated_at
		 FROM service_reports WHERE id = $1`, id).
		Scan(&rep.ID, &rep.CustomerID, &rep.Category, &rep.Description, &rep.Status, &rep.ResolutionNotes, &rep.CreatedAt, &rep.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get report")
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *server) patchReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var in struct {
		Status          *string `json:"status"`
		ResolutionNotes *string `json:"resolutionNotes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if in.Status == nil && in.ResolutionNotes == nil {
		writeError(w, http.StatusBadRequest, "at least one of status or resolutionNotes is required")
		return
	}

	tag, err := s.db.Exec(r.Context(),
		`UPDATE service_reports SET
			status = COALESCE($1, status),
			resolution_notes = COALESCE($2, resolution_notes),
			updated_at = now()
		 WHERE id = $3`,
		in.Status, in.ResolutionNotes, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update report")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	var rep ServiceReport
	err = s.db.QueryRow(r.Context(),
		`SELECT id, customer_id, category, description, status, resolution_notes, created_at, updated_at
		 FROM service_reports WHERE id = $1`, id).
		Scan(&rep.ID, &rep.CustomerID, &rep.Category, &rep.Description, &rep.Status, &rep.ResolutionNotes, &rep.CreatedAt, &rep.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload report")
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
