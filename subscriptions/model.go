package subscriptions

import "time"

type Subscription struct {
	ID        string    `bson:"_id,omitempty"`
	UserID    int64     `bson:"user_id"`
	IsTrial   bool      `bson:"is_trial"`
	Active    bool      `bson:"active"`
	StartsAt  time.Time `bson:"starts_at"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}
