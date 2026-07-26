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

type ListFilter struct {
	Status string
	UserID int64
	Limit  int
	Offset int
}

type Totals struct {
	Total     int64   `json:"total"`
	Completed int64   `json:"completed"`
	Pending   int64   `json:"pending"`
	Revenue   float64 `json:"revenue"`
}

type DailyRevenue struct {
	Date    string  `bson:"_id" json:"date"`
	Revenue float64 `bson:"revenue" json:"revenue"`
	Count   int64   `bson:"count" json:"count"`
}

type PlanRevenue struct {
	PlanID  string  `bson:"_id" json:"plan_id"`
	Revenue float64 `bson:"revenue" json:"revenue"`
	Count   int64   `bson:"count" json:"count"`
}
