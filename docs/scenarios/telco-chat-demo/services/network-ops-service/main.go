package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to create postgres pool: %v", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	if err := migrate(ctx, pool); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}
	if err := seed(ctx, pool); err != nil {
		log.Fatalf("failed to seed data: %v", err)
	}

	s := &server{db: pool}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /customers/{id}/usage", s.getUsage)
	mux.HandleFunc("GET /customers/{id}/usage/history", s.getUsageHistory)
	mux.HandleFunc("POST /reports", s.createReport)
	mux.HandleFunc("GET /reports", s.listReports)
	mux.HandleFunc("GET /reports/{id}", s.getReport)
	mux.HandleFunc("PATCH /reports/{id}", s.patchReport)

	handler := withLogging(mux)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Printf("network-ops-service listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
