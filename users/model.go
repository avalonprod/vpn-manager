package users

import (
	"time"
)

type User struct {
	ID           int64     `bson:"_id,omitempty"`
	Username     string    `bson:"username"`
	FirstName    string    `bson:"first_name"`
	CreatedAt    time.Time `bson:"created_at"`
	LastActiveAt time.Time `bson:"last_active"`
}
