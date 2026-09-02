package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// requireEmployee validates the `Authorization: Bearer <jwt>` header and
// requires role=="employee". Every dashboard route below reads (and one
// writes) real customer data across a project boundary, so — unlike the
// demo-scoped /api/conversations/{id}/messages endpoint — these are never
// left open to an unauthenticated or customer-role caller.
func (s *Server) requireEmployee(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || token == "" {
		return nil, errors.New("missing or malformed Authorization header")
	}
	claims, err := parseToken(token, s.cfg.JWTSecret)
	if err != nil {
		return nil, errors.New("invalid or expired token")
	}
	if claims.Role != "employee" {
		return nil, errors.New("employee role required")
	}
	return claims, nil
}

// writeBackendError propagates an upstream service's real status code
// (e.g. a genuine 404) rather than flattening every failure to 500.
func writeBackendError(w http.ResponseWriter, err error) {
	var be *backendError
	if errors.As(err, &be) {
		writeJSONError(w, be.statusCode, "upstream error")
		return
	}
	writeJSONError(w, http.StatusBadGateway, "upstream service unavailable")
}

// writeRaw writes an already-JSON-encoded byte slice straight through —
// for pure pass-through proxy responses where re-decoding and re-encoding
// would just be wasted work.
func writeRaw(w http.ResponseWriter, status int, raw json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

// handleListCustomers proxies GET /customers?search= to subscription-service.
func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireEmployee(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	q := url.Values{}
	if search := r.URL.Query().Get("search"); search != "" {
		q.Set("search", search)
	}
	raw, err := s.backend.subscriptionGet(r.Context(), "/customers", q)
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeRaw(w, http.StatusOK, raw)
}

// customerAccount is the composed view handleGetCustomerAccount returns —
// everything the employee dashboard needs for one customer in a single
// call. Mirrors chat-agent's get_customer_account tool composition, just
// for the dashboard instead of the LLM.
type customerAccount struct {
	Profile      json.RawMessage `json:"profile"`
	Subscription json.RawMessage `json:"subscription"`
	UsageHistory json.RawMessage `json:"usageHistory"`
	Reports      json.RawMessage `json:"reports"`
}

func (s *Server) handleGetCustomerAccount(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireEmployee(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	customerID := r.PathValue("id")
	if customerID == "" {
		writeJSONError(w, http.StatusBadRequest, "customer id is required")
		return
	}

	profile, err := s.backend.subscriptionGet(r.Context(), "/customers/"+customerID, nil)
	if err != nil {
		writeBackendError(w, err)
		return
	}
	subscription, err := s.backend.subscriptionGet(r.Context(), "/customers/"+customerID+"/subscription", nil)
	if err != nil {
		writeBackendError(w, err)
		return
	}
	usageHistory, err := s.backend.networkOpsGet(r.Context(), "/customers/"+customerID+"/usage/history", url.Values{"days": {"7"}})
	if err != nil {
		writeBackendError(w, err)
		return
	}
	reports, err := s.backend.networkOpsGet(r.Context(), "/reports", url.Values{"customerId": {customerID}})
	if err != nil {
		writeBackendError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, customerAccount{
		Profile:      profile,
		Subscription: subscription,
		UsageHistory: usageHistory,
		Reports:      reports,
	})
}

// handleListReports proxies GET /reports (status/category/customerId
// filters, all optional) to network-ops-service — the dashboard's
// incidents list view.
func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireEmployee(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	q := url.Values{}
	for _, key := range []string{"status", "category", "customerId"} {
		if v := r.URL.Query().Get(key); v != "" {
			q.Set(key, v)
		}
	}
	raw, err := s.backend.networkOpsGet(r.Context(), "/reports", q)
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeRaw(w, http.StatusOK, raw)
}

// reportSummary is just enough of a report's shape to read back the
// customerId/category needed to fetch related incidents and to log audit
// entries against the right customer.
type reportSummary struct {
	CustomerID string `json:"customerId"`
	Category   string `json:"category"`
}

// reportDetail bundles one report with related incidents — this
// customer's other reports, and other customers' reports in the same
// category — so the employee console's "related incidents" panel is a
// single call, not three.
type reportDetail struct {
	Report            json.RawMessage `json:"report"`
	RelatedByCustomer json.RawMessage `json:"relatedByCustomer"`
	RelatedByCategory json.RawMessage `json:"relatedByCategory"`
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireEmployee(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := r.PathValue("id")
	raw, err := s.backend.networkOpsGet(r.Context(), "/reports/"+id, nil)
	if err != nil {
		writeBackendError(w, err)
		return
	}

	var summary reportSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to parse report")
		return
	}

	relatedByCustomer, err := s.backend.networkOpsGet(r.Context(), "/reports",
		url.Values{"customerId": {summary.CustomerID}, "excludeId": {id}})
	if err != nil {
		writeBackendError(w, err)
		return
	}
	relatedByCategory, err := s.backend.networkOpsGet(r.Context(), "/reports",
		url.Values{"category": {summary.Category}, "excludeId": {id}})
	if err != nil {
		writeBackendError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reportDetail{
		Report:            raw,
		RelatedByCustomer: relatedByCustomer,
		RelatedByCategory: relatedByCategory,
	})
}

// handleUpdateReport proxies PATCH /reports/{id} to network-ops-service and
// writes an audit_log row for the change — every employee-initiated
// mutation in this system is audited, and resolving a ticket is no
// exception.
func (s *Server) handleUpdateReport(w http.ResponseWriter, r *http.Request) {
	claims, err := s.requireEmployee(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := r.PathValue("id")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	raw, err := s.backend.networkOpsPatch(r.Context(), "/reports/"+id, body)
	if err != nil {
		writeBackendError(w, err)
		return
	}

	var summary reportSummary
	targetCustomerID := ""
	if err := json.Unmarshal(raw, &summary); err == nil {
		targetCustomerID = summary.CustomerID
	}
	if err := s.store.insertAudit(r.Context(), claims.Role, claims.Subject, targetCustomerID, "update_report", body); err != nil {
		log.Printf("WARNING: failed to write audit log for report update on %s: %v", id, err)
	}

	writeRaw(w, http.StatusOK, raw)
}
