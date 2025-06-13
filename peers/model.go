package peers

import "time"

type Peer struct {
	ID        string    `bson:"_id,omitempty"`
	UserID    int64     `bson:"user_id"`
	ServerID  string    `bson:"server_id"`
	PublicKey string    `bson:"public_key"`
	Location  string    `bson:"location"`
	Config    string    `bson:"config"`
	IsActive  bool      `bson:"is_active"`
	CreatedAt time.Time `bson:"created_at"`
}
