package analytics

import (
	"context"
	"vpn-manager/payments"
	"vpn-manager/subscriptions"
	"vpn-manager/users"
)

type ISheetsWriter interface {
	WriteData(ctx context.Context, values [][]interface{}) error
	ClearSheet(ctx context.Context) error
}

type IUsersStore interface {
	GetAll(ctx context.Context) ([]users.User, error)
	CountUsers(ctx context.Context) (int64, error)
}

type ISubscriptionsStore interface {
	GetSubscriptionsForUsers(ctx context.Context, userIDs []int64) (map[int64]subscriptions.Subscription, error)
	CountTrialSubscriptions(ctx context.Context) (int64, error)
}

type IPaymentsStore interface {
	GetAllCompletedInvoices(ctx context.Context) ([]payments.Invoice, error)
	CountCompletedInvoices(ctx context.Context) (int64, error)
}
