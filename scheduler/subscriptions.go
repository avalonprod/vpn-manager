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
	GetPeerByUserID(ctx context.Context, userID int64) (peers.Peer, error)
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
