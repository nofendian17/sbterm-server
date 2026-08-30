package domain

import "time"

type Watchlist struct {
	ID        string
	UserID    string
	Symbol    string
	Label     string
	CreatedAt time.Time
}

type AddWatchlistInput struct {
	Symbol string `validate:"required"`
	Label  string
}
