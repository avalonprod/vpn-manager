package peers

import "time"

type Sub struct {
	Location string `bson:"location"`
	ServerID string `bson:"server_id"`
	URL      string `bson:"url"`
	Enabled  bool   `bson:"enabled"`
}

type Peer struct {
	ID         string     `bson:"_id,omitempty"`
	UserID     int64      `bson:"user_id"`
	Email      string     `bson:"email"`
	UUID       string     `bson:"uuid"`
	Subs       []Sub      `bson:"subs"`
	IsActive   bool       `bson:"is_active"`
	CreatedAt  time.Time  `bson:"created_at"`
	IsImported bool       `bson:"is_imported"`
	ImportedAt *time.Time `bson:"imported_at,omitempty"`
}
