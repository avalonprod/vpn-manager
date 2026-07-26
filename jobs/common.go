package jobs

import (
	"context"
	"vpn-manager/peers"
	"vpn-manager/servers"
)

type IBot interface {
	SendTrialSubscriptionsExpiryReminder(userID int64) error
}

type IServersService interface {
	RevokeAccessEverywhere(ctx context.Context, email string) (servers.RevocationResult, error)
}

type IPeersService interface {
	GetPeerByUserID(ctx context.Context, userID int64) (peers.Peer, error)
	DeactivatePeer(ctx context.Context, userID int64) error
}
