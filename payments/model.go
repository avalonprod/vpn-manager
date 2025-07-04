package payments

import "time"

const (
	StatusCreated   = "created"
	StatusPending   = "pending"
	StatusCompleted = "completed"
)

type Invoice struct {
	ID        string    `bson:"_id,omitempty"`
	PlanID    string    `bson:"plan_id"`
	UserID    int64     `bson:"user_id"`
	Status    string    `bson:"status"`
	CreatedAt time.Time `bson:"created_at"`
}
