package main

import (
	"net/http"
)

// handleGetConversationMessages returns every message of a conversation,
// oldest first.
//
// NOTE: this endpoint intentionally performs no auth check (see README) -
// it's a demo-scoped simplification, not an oversight.
func (s *Server) handleGetConversationMessages(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	if conversationID == "" {
		writeJSONError(w, http.StatusBadRequest, "conversation id is required")
		return
	}

	messages, err := s.store.listMessages(r.Context(), conversationID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}
