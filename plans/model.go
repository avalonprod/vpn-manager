package plans

import "time"

type Plan struct {
	ID           string    `bson:"_id,omitempty"`
	Title        string    `bson:"title"`
	SubTitle     string    `bson:"sub_title"`
	Price        float64   `bson:"price"`
	Currency     string    `bson:"currency"`
	DurationDays int       `bson:"duration_days"`
	IsActive     bool      `bson:"is_active"`
	Order        int       `bson:"order"`
	CreatedAt    time.Time `bson:"created_at"`
}

type CreateInput struct {
	ID           string
	Title        string
	SubTitle     string
	Price        float64
	Currency     string
	DurationDays int
	IsActive     bool
	Order        int
}

type UpdateInput struct {
	Title        *string
	SubTitle     *string
	Price        *float64
	Currency     *string
	DurationDays *int
	IsActive     *bool
	Order        *int
}
