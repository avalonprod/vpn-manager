package peers

import "time"

type Peer struct {
	ID            string    `bson:"_id,omitempty"`
	UserID        int64     `bson:"user_id"`
	ServerID      string    `bson:"server_id"`
	Location      string    `bson:"location"`
	ConnectionURI string    `bson:"connection_uri"`
	IsActive      bool      `bson:"is_active"`
	CreatedAt     time.Time `bson:"created_at"`
}
