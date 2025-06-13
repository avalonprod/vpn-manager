package servers

import "time"

type Server struct {
	ID           string    `bson:"_id"`
	Location     string    `bson:"location"`
	ServerApiUrl string    `bson:"server_api_url"`
	AuthToken    string    `bson:"auth_token"`
	MaxPeers     int       `bson:"max_peers,omitempty"`
	IsActive     bool      `bson:"is_active"`
	CreatedAt    time.Time `bson:"created_at"`
}
