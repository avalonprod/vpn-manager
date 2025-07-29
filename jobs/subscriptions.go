package jobs

import (
	"context"
	"log"
	"time"
	"vpn-manager/subscriptions"
)

type ISubscriptionsService interface {
	GetExpiredSubscriptions(ctx context.Context) ([]subscriptions.Subscription, error)
	DeactivateExpiredSubscriptions(ctx context.Context) error
}

type INotifier interface {
	Notify(ctx context.Context, userID int64, msg string) error
}

func RunSubscriptionDeactivation(ctx context.Context, subscriptionsService ISubscriptionsService) {
	ticker := time.NewTicker(3 * time.Hour)
	for {
		select {
		case <-ticker.C:
			if err := subscriptionsService.DeactivateExpiredSubscriptions(ctx); err != nil {
				log.Printf("error deactivating expired subscriptions: %v", err)
				continue
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
