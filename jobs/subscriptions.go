package jobs

import (
	"context"
	"time"
	"vpn-manager/pkg/logger"
	"vpn-manager/subscriptions"
)

type ISubscriptionsService interface {
	GetExpiredSubscriptions(ctx context.Context) ([]subscriptions.Subscription, error)
	DeactivateExpiredSubscriptions(ctx context.Context) error
}

func RunDisableExpiredAccess(
	ctx context.Context,
	subs ISubscriptionsService,
	peers IPeersService,
	servers IServersService,
	bot IBot,
	log logger.ILogger,
) {
	ticker := time.NewTicker(3 * time.Minute)

	for {
		run := func() {
			expired, err := subs.GetExpiredSubscriptions(ctx)
			if err != nil {
				log.Errorf("get expired subs: %v", err)
				return
			}

			for _, s := range expired {
				if s.IsTrial {
					err := bot.SendTrialSubscriptionsExpiryReminder(s.UserID)
					if err != nil {
						log.Errorf("error sending trial subscription expiry reminder to user %d: %v", s.UserID, err)
						continue
					}
				}

				peer, err := peers.GetPeerByUserID(ctx, s.UserID)
				if err != nil {
					log.Errorf("get peer by user %d: %v", s.UserID, err)
					continue
				}

				result, err := servers.RevokeAccessEverywhere(ctx, peer.Email)
				if err != nil {
					log.Errorf("revoke access for user %d: %v", s.UserID, err)
				} else if len(result.Failed) > 0 {
					log.Errorf("access for user %d is still live on: %v", s.UserID, result.Failed)
				}

				if err := peers.DeactivatePeer(ctx, s.UserID); err != nil {
					log.Errorf("deactivate peers for user %d: %v", s.UserID, err)
				}
			}

			if err := subs.DeactivateExpiredSubscriptions(ctx); err != nil {
				log.Errorf("deactivate expired subs: %v", err)
			}
		}

		select {
		case <-ticker.C:
			run()
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
