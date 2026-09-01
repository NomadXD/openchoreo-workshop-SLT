package main

import "time"

type UsageRecord struct {
	CustomerID  string `json:"customerId"`
	Date        string `json:"date"`
	BrowsingMb  int    `json:"browsingMb"`
	StreamingMb int    `json:"streamingMb"`
	SocialMb    int    `json:"socialMb"`
	OtherMb     int    `json:"otherMb"`
	TotalMb     int    `json:"totalMb"`
}

type ServiceReport struct {
	ID              string    `json:"id"`
	CustomerID      string    `json:"customerId"`
	Category        string    `json:"category"`
	Description     string    `json:"description"`
	Status          string    `json:"status"`
	ResolutionNotes *string   `json:"resolutionNotes"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
