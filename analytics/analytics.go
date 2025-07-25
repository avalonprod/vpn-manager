package analytics

import (
	"context"
	"fmt"
	"vpn-manager/pkg/logger"
)

type Analytics struct {
	sheets             ISheetsWriter
	usersStore         IUsersStore
	subscriptionsStore ISubscriptionsStore
	paymentsStore      IPaymentsStore
	logger             logger.ILogger
}

func NewAnalytics(sheetsWriter ISheetsWriter, usersStore IUsersStore, subscriptionsStore ISubscriptionsStore, paymentsStore IPaymentsStore, logger logger.ILogger) *Analytics {
	return &Analytics{
		sheets:             sheetsWriter,
		usersStore:         usersStore,
		subscriptionsStore: subscriptionsStore,
		paymentsStore:      paymentsStore,
		logger:             logger,
	}
}

func (a *Analytics) UpdateAnalyticsData(ctx context.Context) error {
	usersCount, err := a.usersStore.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	subscriptionsCount, err := a.subscriptionsStore.CountTrialSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to count trial subscriptions: %w", err)
	}

	invoicesCount, err := a.paymentsStore.CountCompletedInvoices(ctx)
	if err != nil {
		return fmt.Errorf("failed to count completed invoices: %w", err)
	}

	if err := a.sheets.ClearSheet(ctx); err != nil {
		return fmt.Errorf("failed to clear analytics sheet: %w", err)
	}

	if err := a.sheets.WriteData(ctx, [][]interface{}{
		{"👥 Пользователи", "🎁 Пробные подписки", "💳 Оплаченные подписки"},
		{usersCount, subscriptionsCount, invoicesCount},
	}); err != nil {
		return fmt.Errorf("failed to write data to analytics sheet: %w", err)
	}

	return nil
}
