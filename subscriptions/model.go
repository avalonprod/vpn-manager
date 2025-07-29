package subscriptions

import "time"

type Subscription struct {
	ID          string    `bson:"_id,omitempty"`
	UserID      int64     `bson:"user_id"`
	PlanID      string    `bson:"plan_id"`
	Active      bool      `bson:"active"`
	IsTrial     bool      `bson:"is_trial"`
	AutoRenewal bool      `bson:"auto_renewal"`
	ExpiresAt   time.Time `bson:"expires_at"`
	CreatedAt   time.Time `bson:"created_at"`
}
