package scheduler

import (
	"context"
	"vpn-manager/peers"
	"vpn-manager/pkg/logger"
	"vpn-manager/subscriptions"
)

type ISubscriptionsService interface {
	GetExpiredSubscriptions(ctx context.Context) ([]subscriptions.Subscription, error)
	DeactivateExpiredSubscriptions(ctx context.Context) error
}

type IPeersService interface {
	GetPeersByUserID(ctx context.Context, userID int64) ([]peers.Peer, error)
	DeletePeersByUserID(ctx context.Context, userID int64) error
}

type IServersService interface {
	DeletePeerFromServer(ctx context.Context, serverID, publicKey string) error
}

type INotifier interface {
	Notify(ctx context.Context, userID int64, msg string) error
}

type Scheduler struct {
	subscriptionsService ISubscriptionsService
	peersService         IPeersService
	serversService       IServersService
	notifier             INotifier
	logger               logger.ILogger
}

func NewScheduler(subscriptionsService ISubscriptionsService, peersService IPeersService, serversService IServersService, notifier INotifier, logger logger.ILogger) *Scheduler {
	return &Scheduler{
		subscriptionsService: subscriptionsService,
		peersService:         peersService,
		serversService:       serversService,
		notifier:             notifier,
		logger:               logger,
	}
}

func (s *Scheduler) CheckExpiredSubscriptions(ctx context.Context) {
	subs, err := s.subscriptionsService.GetExpiredSubscriptions(ctx)
	if err != nil {
		s.logger.Errorf("failed to get expired subs: %v", err)
		return
	}

	for _, sub := range subs {
		peers, err := s.peersService.GetPeersByUserID(ctx, sub.UserID)
		if err != nil {
			s.logger.Errorf("failed to get peers: %v", err)
		}

		for _, peer := range peers {
			if err := s.serversService.DeletePeerFromServer(ctx, peer.ServerID, peer.PublicKey); err != nil {
				s.logger.Errorf("failed to delete peer from server for user %d err: %v", sub.UserID, err)
				continue
			}
		}

		if err := s.peersService.DeletePeersByUserID(ctx, sub.UserID); err != nil {
			s.logger.Errorf("failed to delete peer for user %d err: %v", sub.UserID, err)
		}

		err = s.notifier.Notify(ctx, sub.UserID, "Ваша подписка истекла пожалуйста продлите подписку чтобы продолжить использовать наш VPN сервис")
		if err != nil {
			s.logger.Errorf("failed to notify user: %d err: %v", sub.UserID, err)
		}
	}

	if err := s.subscriptionsService.DeactivateExpiredSubscriptions(ctx); err != nil {
		s.logger.Errorf("failed to deactivate expired subs: %v", err)
	}
}
