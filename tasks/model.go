package tasks

import "time"

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

type Task struct {
	ID          string    `bson:"_id,omitempty"`
	Type        string    `bson:"type"`
	UserID      int64     `bson:"user_id,omitempty"`
	Payload     []byte    `bson:"payload,omitempty"`
	RunAt       time.Time `bson:"run_at"`
	Status      Status    `bson:"status"`
	Attempts    int       `bson:"attempts"`
	MaxAttempts int       `bson:"max_attempts"`
	DedupeKey   string    `bson:"dedupe_key,omitempty"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}
