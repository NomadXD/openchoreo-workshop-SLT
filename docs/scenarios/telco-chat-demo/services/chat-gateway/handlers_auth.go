package main

import (
	"encoding/json"
	"net/http"
)

type customerLoginRequest struct {
	CustomerID string `json:"customerId"`
}

type employeeLoginRequest struct {
	AgentID string `json:"agentId"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// handleCustomerLogin issues a mock JWT for a customer. No password check:
// this is a demo, any non-empty customerId is accepted.
func (s *Server) handleCustomerLogin(w http.ResponseWriter, r *http.Request) {
	var req customerLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CustomerID == "" {
		writeJSONError(w, http.StatusBadRequest, "customerId is required")
		return
	}

	token, err := generateToken(req.CustomerID, "customer", s.cfg.JWTSecret)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

// handleEmployeeLogin issues a mock JWT for an employee. No password check:
// this is a demo, any non-empty agentId is accepted.
func (s *Server) handleEmployeeLogin(w http.ResponseWriter, r *http.Request) {
	var req employeeLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AgentID == "" {
		writeJSONError(w, http.StatusBadRequest, "agentId is required")
		return
	}

	token, err := generateToken(req.AgentID, "employee", s.cfg.JWTSecret)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
