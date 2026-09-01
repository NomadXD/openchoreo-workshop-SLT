package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func migrate(ctx context.Context, db *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS customers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			msisdn TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS plans (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			data_gb INT NULL,
			price_cents INT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT true
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			customer_id TEXT PRIMARY KEY REFERENCES customers(id),
			plan_id TEXT NOT NULL REFERENCES plans(id),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("running migration statement: %w", err)
		}
	}
	return nil
}

func seed(ctx context.Context, db *pgxpool.Pool) error {
	var planCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM plans`).Scan(&planCount); err != nil {
		return fmt.Errorf("counting plans: %w", err)
	}
	if planCount == 0 {
		plans := []struct {
			id, name   string
			dataGb     *int
			priceCents int
		}{
			{"plan-basic-20", "Basic 20GB", intPtr(20), 149000},
			{"plan-family-100", "Family 100GB", intPtr(100), 349000},
			{"plan-unlimited", "Unlimited Data", nil, 599000},
		}
		for _, p := range plans {
			if _, err := db.Exec(ctx,
				`INSERT INTO plans (id, name, data_gb, price_cents, active) VALUES ($1, $2, $3, $4, true)`,
				p.id, p.name, p.dataGb, p.priceCents); err != nil {
				return fmt.Errorf("seeding plan %s: %w", p.id, err)
			}
		}
	}

	var customerCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM customers`).Scan(&customerCount); err != nil {
		return fmt.Errorf("counting customers: %w", err)
	}
	if customerCount == 0 {
		customers := []struct {
			id, name, msisdn, email string
		}{
			{"cust-001", "Amara Perera", "+94771234001", "amara.perera@example.lk"},
			{"cust-002", "Nadeesha Fernando", "+94771234002", "nadeesha.fernando@example.lk"},
			{"cust-003", "Kasun Silva", "+94771234003", "kasun.silva@example.lk"},
			{"cust-004", "Ishara Jayawardena", "+94771234004", "ishara.jayawardena@example.lk"},
		}
		for _, c := range customers {
			if _, err := db.Exec(ctx,
				`INSERT INTO customers (id, name, msisdn, email, created_at) VALUES ($1, $2, $3, $4, now())`,
				c.id, c.name, c.msisdn, c.email); err != nil {
				return fmt.Errorf("seeding customer %s: %w", c.id, err)
			}
		}

		subs := map[string]string{
			"cust-001": "plan-basic-20",
			"cust-002": "plan-family-100",
			"cust-003": "plan-basic-20",
			"cust-004": "plan-basic-20",
		}
		for custID, planID := range subs {
			if _, err := db.Exec(ctx,
				`INSERT INTO subscriptions (customer_id, plan_id, updated_at) VALUES ($1, $2, now())`,
				custID, planID); err != nil {
				return fmt.Errorf("seeding subscription for %s: %w", custID, err)
			}
		}
	}

	return nil
}

func intPtr(v int) *int {
	return &v
}
