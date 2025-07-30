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

func RunSubscriptionDeactivation(ctx context.Context, subscriptionsService ISubscriptionsService, bot IBot) {
	ticker := time.NewTicker(3 * time.Hour)
	for {
		select {
		case <-ticker.C:
			subscriptions, err := subscriptionsService.GetExpiredSubscriptions(ctx)
			if err != nil {
				log.Printf("error getting expired subscriptions: %v", err)
				continue
			}
			for _, subscription := range subscriptions {
				if subscription.IsTrial {
					err := bot.SendTrialSubscriptionsExpiryReminder(subscription.UserID)
					if err != nil {
						log.Printf("error sending trial subscription expiry reminder to user %d: %v", subscription.UserID, err)
						continue
					}
				}
			}

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
