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

				for _, sub := range peer.Subs {
					if err := servers.DeletePeerFromServer(ctx, sub.ServerID, peer.UUID); err != nil {
						log.Errorf("remove client on server %s for user %d: %v", sub.ServerID, s.UserID, err)
					}
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
