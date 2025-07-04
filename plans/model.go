package plans

import "time"

type Plan struct {
	ID           string    `bson:"_id,omitempty"`
	Title        string    `bson:"title"`
	Price        float64   `bson:"price"`
	Currency     string    `bson:"currency"`
	DurationDays int       `bson:"duration_days"`
	IsActive     bool      `bson:"is_active"`
	CreatedAt    time.Time `bson:"created_at"`
}
