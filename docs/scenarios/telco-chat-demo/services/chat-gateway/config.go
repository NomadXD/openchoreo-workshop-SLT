package main

import (
	"log"
	"os"
)

// Config holds all environment-derived settings for chat-gateway.
type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	ChatAgentURL string
	JWTSecret    string
}

const defaultJWTSecret = "dev-secret-change-me"

func loadConfig() Config {
	cfg := Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		RedisURL:     os.Getenv("REDIS_URL"),
		ChatAgentURL: os.Getenv("CHAT_AGENT_URL"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
	}

	if cfg.JWTSecret == "" {
		log.Printf("WARNING: JWT_SECRET not set, using insecure default %q (do not use in production)", defaultJWTSecret)
		cfg.JWTSecret = defaultJWTSecret
	}

	if cfg.DatabaseURL == "" {
		log.Printf("WARNING: DATABASE_URL is not set")
	}
	if cfg.RedisURL == "" {
		log.Printf("WARNING: REDIS_URL is not set")
	}
	if cfg.ChatAgentURL == "" {
		log.Printf("WARNING: CHAT_AGENT_URL is not set")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
