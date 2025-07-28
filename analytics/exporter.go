package analytics

import (
	"context"
	"fmt"
)

type Exporter struct {
	sheets             ISheetsWriter
	usersStore         IUsersStore
	subscriptionsStore ISubscriptionsStore
	paymentsStore      IPaymentsStore
	peersStore         IPeersStore
}

func NewExporter(
	sheetsWriter ISheetsWriter,
	usersStore IUsersStore,
	subscriptionsStore ISubscriptionsStore,
	paymentsStore IPaymentsStore,
	peersStore IPeersStore,
) *Exporter {
	return &Exporter{
		sheets:             sheetsWriter,
		usersStore:         usersStore,
		subscriptionsStore: subscriptionsStore,
		paymentsStore:      paymentsStore,
		peersStore:         peersStore,
	}
}

func (e *Exporter) ExportOverview(ctx context.Context) error {
	var rows [][]interface{}
	rows = append(rows, []interface{}{"First Name", "Username", "Started At", "Plan", "Subscription Is Active", "Expires At", "Subscription Created At", "Is Imported Peer", "Imported At"})

	users, err := e.usersStore.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	userIds := make([]int64, 0, len(users))

	for _, user := range users {
		userIds = append(userIds, user.ID)
	}

	subscriptions, err := e.subscriptionsStore.GetSubscriptionsForUsers(ctx, userIds)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	peers, err := e.peersStore.GetPeersForUsers(ctx, userIds)
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	for _, user := range users {
		rows = append(rows, []interface{}{
			user.FirstName,
			user.Username,
			user.CreatedAt.Format("2006-01-02 15:04:05"),
			subscriptions[user.ID].PlanID,
			subscriptions[user.ID].Active,
			subscriptions[user.ID].ExpiresAt.Format("2006-01-02 15:04:05"),
			subscriptions[user.ID].CreatedAt.Format("2006-01-02 15:04:05"),
			peers[user.ID].IsImported,
			peers[user.ID].ImportedAt.Format("2006-01-02 15:04:05"),
		})
	}

	if err := e.sheets.ClearSheet(ctx); err != nil {
		return fmt.Errorf("failed to clear sheet: %w", err)
	}

	return e.sheets.WriteData(ctx, rows)
}
