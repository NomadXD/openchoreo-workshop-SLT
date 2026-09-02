// Command chat-gateway is the browser-facing BFF for the telco chat demo.
// It owns mock login, the websocket chat connection, conversation
// persistence, and forwarding turns to the internal chat-agent service.
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

// Server holds every dependency shared across HTTP/WS handlers.
type Server struct {
	cfg         Config
	store       *Store
	rdb         *redis.Client
	rateLimiter *RateLimiter
	agentClient *AgentClient
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()

	store, err := newStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize postgres store: %v", err)
	}
	defer store.Close()
	log.Println("connected to postgres and ran migrations")

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("invalid REDIS_URL: %v", err)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis at %s: %v", redisOpts.Addr, err)
	}
	defer rdb.Close()
	log.Println("connected to redis")

	srv := &Server{
		cfg:         cfg,
		store:       store,
		rdb:         rdb,
		rateLimiter: newRateLimiter(rdb),
		agentClient: newAgentClient(cfg.ChatAgentURL),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/auth/customer/login", srv.handleCustomerLogin)
	mux.HandleFunc("POST /api/auth/employee/login", srv.handleEmployeeLogin)
	mux.HandleFunc("GET /api/conversations/{id}/messages", srv.handleGetConversationMessages)
	mux.HandleFunc("GET /ws/chat", srv.handleWebSocket)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("chat-gateway listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("error during shutdown: %v", err)
	}
}
