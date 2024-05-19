package model

import (
	"time"
)

type BankRate struct {
	Bank        string    `json:"bank"`
	Buy         float64   `json:"buy"`
	BuyOnline   float64   `json:"buy_online"`
	Sell        float64   `json:"sell"`
	SellOnline  float64   `json:"sell_online"`
	LastUpdated time.Time `json:"update_at"`
	// last update source url
	Source  string `json:"source"`
	SiteUrl string `json:"site_url"`
}
