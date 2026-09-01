package main

type Plan struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	DataGb     *int   `json:"dataGb"`
	PriceCents int    `json:"priceCents"`
}

type Customer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Msisdn string `json:"msisdn"`
	Email  string `json:"email"`
}

type Subscription struct {
	CustomerID string `json:"customerId"`
	Plan       Plan   `json:"plan"`
}
