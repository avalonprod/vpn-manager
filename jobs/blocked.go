package jobs

import (
	"context"
	"time"
	"vpn-manager/pkg/logger"
	"vpn-manager/users"
)

type IUsersService interface {
	List(ctx context.Context, f users.ListFilter) ([]users.User, int64, error)
}

const (
	reconcileInterval  = 5 * time.Minute
	reconcileBatchSize = 200
)

func RunRevokeBlockedAccess(
	ctx context.Context,
	usersService IUsersService,
	peersService IPeersService,
	serversService IServersService,
	log logger.ILogger,
) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			reconcileBlocked(ctx, usersService, peersService, serversService, log)
		case <-ctx.Done():
			return
		}
	}
}

func reconcileBlocked(
	ctx context.Context,
	usersService IUsersService,
	peersService IPeersService,
	serversService IServersService,
	log logger.ILogger,
) {
	for offset := 0; ; offset += reconcileBatchSize {
		blocked, total, err := usersService.List(ctx, users.ListFilter{
			Blocked: users.BlockedOnly,
			Limit:   reconcileBatchSize,
			Offset:  offset,
		})
		if err != nil {
			log.Errorf("reconcile blocked users: %v", err)
			return
		}

		for _, user := range blocked {
			peer, err := peersService.GetPeerByUserID(ctx, user.ID)
			if err != nil {
				continue
			}

			if peer.IsActive {
				if err := peersService.DeactivatePeer(ctx, user.ID); err != nil {
					log.Errorf("reconcile: deactivate peer of blocked user %d: %v", user.ID, err)
				}
			}

			result, err := serversService.RevokeAccessEverywhere(ctx, peer.Email)
			if err != nil {
				log.Errorf("reconcile: revoke access for blocked user %d: %v", user.ID, err)
				continue
			}

			if len(result.Failed) > 0 {
				log.Errorf("reconcile: access of blocked user %d is still live on: %v", user.ID, result.Failed)
			}
		}

		if int64(offset+len(blocked)) >= total || len(blocked) == 0 {
			return
		}
	}
}
