package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func migrate(ctx context.Context, db *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS usage_records (
			customer_id TEXT NOT NULL,
			date DATE NOT NULL,
			browsing_mb INT NOT NULL,
			streaming_mb INT NOT NULL,
			social_mb INT NOT NULL,
			other_mb INT NOT NULL,
			PRIMARY KEY (customer_id, date)
		)`,
		`CREATE TABLE IF NOT EXISTS service_reports (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL,
			category TEXT NOT NULL,
			description TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			resolution_notes TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
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

var demoCustomerIDs = []string{"cust-001", "cust-002", "cust-003", "cust-004"}

func seed(ctx context.Context, db *pgxpool.Pool) error {
	var usageCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM usage_records`).Scan(&usageCount); err != nil {
		return fmt.Errorf("counting usage_records: %w", err)
	}
	if usageCount == 0 {
		now := time.Now()
		for _, custID := range demoCustomerIDs {
			for daysAgo := 0; daysAgo < 7; daysAgo++ {
				date := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
				browsing := randMb(300, 3000)
				streaming := randMb(500, 6000)
				social := randMb(100, 1500)
				other := randMb(50, 800)
				if _, err := db.Exec(ctx,
					`INSERT INTO usage_records (customer_id, date, browsing_mb, streaming_mb, social_mb, other_mb)
					 VALUES ($1, $2, $3, $4, $5, $6)
					 ON CONFLICT (customer_id, date) DO NOTHING`,
					custID, date, browsing, streaming, social, other); err != nil {
					return fmt.Errorf("seeding usage for %s on %s: %w", custID, date, err)
				}
			}
		}
	}

	var reportCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM service_reports`).Scan(&reportCount); err != nil {
		return fmt.Errorf("counting service_reports: %w", err)
	}
	if reportCount == 0 {
		refunded := "Refunded — found to be a rating error"
		reports := []struct {
			id, customerID, category, description, status string
			resolutionNotes                               *string
		}{
			{
				id: mustReportID(), customerID: "cust-001", category: "connectivity",
				description: "Intermittent 4G drop in Nugegoda area", status: "open",
			},
			{
				id: mustReportID(), customerID: "cust-003", category: "billing",
				description: "Unexpected data charge", status: "resolved",
				resolutionNotes: &refunded,
			},
		}
		for _, rep := range reports {
			if _, err := db.Exec(ctx,
				`INSERT INTO service_reports (id, customer_id, category, description, status, resolution_notes, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, now(), now())`,
				rep.id, rep.customerID, rep.category, rep.description, rep.status, rep.resolutionNotes); err != nil {
				return fmt.Errorf("seeding report for %s: %w", rep.customerID, err)
			}
		}
	}

	return nil
}

// randMb returns a pseudo-random megabyte value in [min, max), varied per call.
func randMb(min, max int) int {
	return min + mrand.Intn(max-min)
}

func newReportID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "rep-" + hex.EncodeToString(b), nil
}

func mustReportID() string {
	id, err := newReportID()
	if err != nil {
		// crypto/rand failures are effectively unrecoverable; fall back to a
		// time-based suffix so seeding never blocks startup on this.
		return fmt.Sprintf("rep-%x", time.Now().UnixNano())
	}
	return id
}
