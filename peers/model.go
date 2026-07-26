package peers

import "time"

type Sub struct {
	Location string `bson:"location"`
	ServerID string `bson:"server_id"`
	URL      string `bson:"url"`
	Enabled  bool   `bson:"enabled"`
}

type Peer struct {
	ID          string    `bson:"_id,omitempty"`
	UserID      int64     `bson:"user_id"`
	Email       string    `bson:"email"`
	UUID        string    `bson:"uuid"`
	AccessToken string    `bson:"access_token,omitempty"`
	Subs        []Sub     `bson:"subs"`
	IsActive    bool      `bson:"is_active"`
	CreatedAt   time.Time `bson:"created_at"`
	IsImported  bool      `bson:"is_imported"`
	ImportedAt  time.Time `bson:"imported_at,omitempty"`
}

type Totals struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Imported int64 `json:"imported"`
}

type LocationCount struct {
	ServerID string `bson:"server_id" json:"server_id"`
	Location string `bson:"location" json:"location"`
	Count    int64  `bson:"count" json:"count"`
}
