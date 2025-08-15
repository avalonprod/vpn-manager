package jobs

import (
	"context"
	"vpn-manager/peers"
)

type IBot interface {
	SendTrialSubscriptionsExpiryReminder(userID int64) error
}

type IServersService interface {
	DeletePeerFromServer(ctx context.Context, serverID, UUID string) error
}

type IPeersService interface {
	GetPeerByUserID(ctx context.Context, userID int64) (peers.Peer, error)
	DeactivatePeer(ctx context.Context, userID int64) error
}
