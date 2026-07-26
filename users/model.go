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
	IsBlocked    bool      `bson:"is_blocked"`
	BlockReason  string    `bson:"block_reason,omitempty"`
	BlockedAt    time.Time `bson:"blocked_at,omitempty"`
}

// BlockedFilter задаёт, по какому признаку блокировки фильтровать выборку.
type BlockedFilter string

const (
	BlockedAny     BlockedFilter = ""
	BlockedOnly    BlockedFilter = "blocked"
	BlockedExclude BlockedFilter = "active"
)

// ListFilter описывает выборку пользователей для админ-панели.
type ListFilter struct {
	// Search ищет по username, first_name и по ID (если строка — число).
	Search  string
	Blocked BlockedFilter
	// SortField: created_at | last_active | id. SortAsc задаёт направление.
	SortField string
	SortAsc   bool
	Limit     int
	Offset    int
}

// DailyCount — одна точка временного ряда «сколько за день».
type DailyCount struct {
	Date  string `bson:"_id" json:"date"`
	Count int64  `bson:"count" json:"count"`
}

// Totals — сводные счётчики по пользователям.
type Totals struct {
	Total    int64 `json:"total"`
	Blocked  int64 `json:"blocked"`
	NewToday int64 `json:"new_today"`
	Active24 int64 `json:"active_24h"`
	Active7d int64 `json:"active_7d"`
}
