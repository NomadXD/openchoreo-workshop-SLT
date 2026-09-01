package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// historyLimit caps how many prior messages of a conversation are sent to
// chat-agent as context.
const historyLimit = 20

var upgrader = websocket.Upgrader{
	// Demo scope: accept connections from any origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// clientMessage is one inbound text frame from the browser.
type clientMessage struct {
	Type             string `json:"type"`
	ConversationID   string `json:"conversationId,omitempty"`
	Content          string `json:"content"`
	TargetCustomerID string `json:"targetCustomerId,omitempty"`
}

// wsError is an outbound {"type":"error", ...} frame.
type wsError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// agentEvent is the shape of one NDJSON line emitted by chat-agent. Only
// the fields chat-gateway needs to act on are parsed; the raw line is
// still forwarded to the browser verbatim regardless of what else it
// contains.
type agentEvent struct {
	Type             string          `json:"type"`
	Content          string          `json:"content"`
	Name             string          `json:"name"`
	Args             json.RawMessage `json:"args"`
	Audit            bool            `json:"audit"`
	TargetCustomerID string          `json:"targetCustomerId"`
}

// handleWebSocket validates the JWT from the ?token= query param, upgrades
// the connection, and then serves chat turns on it until it closes.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	claims, err := parseToken(token, s.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	session := &wsSession{
		server: s,
		conn:   conn,
		role:   claims.Role,
		sub:    claims.Subject,
	}
	session.run()
}

// wsSession holds the per-connection state for one authenticated websocket.
type wsSession struct {
	server *Server
	conn   *websocket.Conn
	role   string // "customer" or "employee"
	sub    string // JWT subject: customerId or agentId

	// conversationID is sticky for the lifetime of the connection once
	// established, either from the first message that carries one or from
	// one this connection creates.
	conversationID string
}

func (sess *wsSession) run() {
	for {
		_, raw, err := sess.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket read error for actor %s: %v", sess.sub, err)
			}
			return
		}

		var msg clientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			sess.sendError("invalid message format")
			continue
		}

		if msg.Type != "message" {
			sess.sendError("unsupported message type")
			continue
		}

		sess.handleChatMessage(msg)
	}
}

func (sess *wsSession) handleChatMessage(msg clientMessage) {
	ctx := context.Background()

	// 1. Resolve targetCustomerId per role rules. Never trust a
	// client-supplied id for a customer's own scope.
	targetCustomerID := msg.TargetCustomerID
	if sess.role == "customer" {
		targetCustomerID = sess.sub
	} else if sess.role == "employee" {
		if strings.TrimSpace(msg.TargetCustomerID) == "" {
			sess.sendError("targetCustomerId is required for employee turns")
			return
		}
		targetCustomerID = msg.TargetCustomerID
	}

	// 2. Resolve the conversation id, creating a new conversation if none
	// was supplied and none is established yet on this connection.
	conversationID := msg.ConversationID
	if conversationID == "" {
		if sess.conversationID != "" {
			conversationID = sess.conversationID
		} else {
			conversationID = newConversationID()
		}
	}
	sess.conversationID = conversationID

	if err := sess.server.store.ensureConversation(ctx, conversationID, "customer", targetCustomerID); err != nil {
		log.Printf("failed to ensure conversation %s: %v", conversationID, err)
		sess.sendError("internal error")
		return
	}

	// 3. Rate limit by actor (the JWT subject), before persisting/forwarding.
	allowed, err := sess.server.rateLimiter.Allow(ctx, sess.sub)
	if err != nil {
		log.Printf("rate limiter error for actor %s: %v", sess.sub, err)
		sess.sendError("internal error")
		return
	}
	if !allowed {
		sess.sendError("rate limit exceeded, try again shortly")
		return
	}

	// 4. Persist the user's turn.
	if err := sess.server.store.insertMessage(ctx, conversationID, "user", msg.Content); err != nil {
		log.Printf("failed to persist user message for conversation %s: %v", conversationID, err)
		sess.sendError("internal error")
		return
	}

	// 5. Build history (oldest-first, capped) and forward to chat-agent.
	history, err := sess.server.store.recentHistory(ctx, conversationID, historyLimit)
	if err != nil {
		log.Printf("failed to load history for conversation %s: %v", conversationID, err)
		sess.sendError("internal error")
		return
	}

	agentReq := AgentRequest{
		Role:             sess.role,
		ActorID:          sess.sub,
		TargetCustomerID: targetCustomerID,
		ConversationID:   conversationID,
		Message:          msg.Content,
		History:          history,
	}

	body, err := sess.server.agentClient.StreamTurn(ctx, agentReq)
	if err != nil {
		log.Printf("chat-agent call failed for conversation %s: %v", conversationID, err)
		sess.sendError("agent unavailable")
		return
	}
	defer body.Close()

	sess.streamAgentResponse(ctx, body, conversationID, targetCustomerID)
}

// streamAgentResponse reads chat-agent's NDJSON response line by line,
// forwarding each event verbatim to the browser and acting on it as
// needed (redis publish, audit log, message persistence).
func (sess *wsSession) streamAgentResponse(ctx context.Context, body io.Reader, conversationID, targetCustomerID string) {
	reader := bufio.NewReader(body)
	var assembledReply strings.Builder
	sawDone := false

	for {
		line, err := reader.ReadString('\n')
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			sess.handleAgentLine(ctx, []byte(trimmed), conversationID, targetCustomerID, &assembledReply, &sawDone)
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("error reading chat-agent stream for conversation %s: %v", conversationID, err)
			}
			break
		}
	}

	if !sawDone && assembledReply.Len() > 0 {
		log.Printf("chat-agent stream for conversation %s ended without a done event; not persisting partial reply", conversationID)
	}
}

func (sess *wsSession) handleAgentLine(ctx context.Context, rawLine []byte, conversationID, targetCustomerID string, assembledReply *strings.Builder, sawDone *bool) {
	var event agentEvent
	if err := json.Unmarshal(rawLine, &event); err != nil {
		log.Printf("malformed line from chat-agent for conversation %s: %v", conversationID, err)
		return
	}

	// Forward verbatim to the browser regardless of type.
	if err := sess.conn.WriteMessage(websocket.TextMessage, rawLine); err != nil {
		log.Printf("failed to write to websocket for conversation %s: %v", conversationID, err)
		return
	}

	switch event.Type {
	case "token":
		assembledReply.WriteString(event.Content)
		if err := publishToConversation(ctx, sess.server.rdb, conversationID, rawLine); err != nil {
			log.Printf("failed to publish token to redis for conversation %s: %v", conversationID, err)
		}

	case "tool_call":
		if event.Audit {
			details := event.Args
			if len(details) == 0 {
				details = json.RawMessage("{}")
			}
			auditTarget := event.TargetCustomerID
			if auditTarget == "" {
				auditTarget = targetCustomerID
			}
			if err := sess.server.store.insertAudit(ctx, sess.role, sess.sub, auditTarget, event.Name, details); err != nil {
				log.Printf("failed to write audit log for conversation %s: %v", conversationID, err)
			}
		}

	case "tool_result":
		// No persistence needed.

	case "done":
		*sawDone = true
		reply := assembledReply.String()
		if reply != "" {
			if err := sess.server.store.insertMessage(ctx, conversationID, "assistant", reply); err != nil {
				log.Printf("failed to persist assistant message for conversation %s: %v", conversationID, err)
			}
		}
		assembledReply.Reset()

	case "error":
		// No persistence needed.
	}
}

func (sess *wsSession) sendError(message string) {
	payload, err := json.Marshal(wsError{Type: "error", Message: message})
	if err != nil {
		return
	}
	if err := sess.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		log.Printf("failed to send error to websocket for actor %s: %v", sess.sub, err)
	}
}
