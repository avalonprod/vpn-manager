package subscriptions

import "time"

type Subscription struct {
	ID          string    `bson:"_id,omitempty"`
	UserID      int64     `bson:"user_id"`
	PlanID      string    `bson:"plan_id"`
	Active      bool      `bson:"active"`
	IsTrial     bool      `bson:"is_trial"`
	AutoRenewal bool      `bson:"auto_renewal"`
	ExpiresAt   time.Time `bson:"expires_at"`
	CreatedAt   time.Time `bson:"created_at"`
}

// Totals — сводка по подпискам для дашборда.
type Totals struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	ActiveTrial  int64 `json:"active_trial"`
	ActivePaid   int64 `json:"active_paid"`
	AutoRenewal  int64 `json:"auto_renewal"`
	ExpiringIn3d int64 `json:"expiring_in_3d"`
}

// PlanCount — распределение подписок по тарифам.
type PlanCount struct {
	PlanID string `bson:"_id" json:"plan_id"`
	Count  int64  `bson:"count" json:"count"`
}

// DailyCount — одна точка временного ряда «сколько за день».
type DailyCount struct {
	Date  string `bson:"_id" json:"date"`
	Count int64  `bson:"count" json:"count"`
}
