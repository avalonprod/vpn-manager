package subscriptions

import "time"

const (
	PlanTrial = "TRIAL"
)

type Subscription struct {
	ID        string    `bson:"_id,omitempty"`
	UserID    int64     `bson:"user_id"`
	Plan      string    `bson:"plan"`
	Active    bool      `bson:"active"`
	StartsAt  time.Time `bson:"starts_at"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}
