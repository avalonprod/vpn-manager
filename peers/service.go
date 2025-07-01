package peers

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IStore interface {
	Create(ctx context.Context, peer Peer) (string, error)
	GetByID(ctx context.Context, ID string) (Peer, error)
	DeletePeersByUserID(ctx context.Context, userID int64) error
	GetPeerByUserID(ctx context.Context, userID int64) (Peer, error)
	UpdateSubs(ctx context.Context, id string, subs []Sub) error
	GetActivePeerByUserID(ctx context.Context, userID int64) (Peer, error)
	SetActive(ctx context.Context, userID int64) error
}

type service struct {
	store IStore
}

func NewService(store IStore) *service {
	return &service{
		store: store,
	}
}

func (s *service) Create(ctx context.Context, userID int64) (Peer, error) {
	peer, err := s.store.GetPeerByUserID(ctx, userID)
	if err != nil {
		if err == ErrPeerNotFound {
			uuid := uuid.New().String()
			email := uuid[:7]

			peer = Peer{
				UserID:    userID,
				Email:     email,
				UUID:      uuid,
				IsActive:  false,
				CreatedAt: time.Now().UTC(),
			}
			id, err := s.store.Create(ctx, peer)
			if err != nil {
				return Peer{}, err
			}

			peer.ID = id

			return peer, nil
		}

		return Peer{}, err
	}

	return peer, nil
}

func (s *service) UpdateSubs(ctx context.Context, id string, subs []Sub) error {
	return s.store.UpdateSubs(ctx, id, subs)
}

func (s *service) GetPeerByUserID(ctx context.Context, userID int64) (Peer, error) {
	return s.store.GetPeerByUserID(ctx, userID)
}

func (s *service) GetActivePeerByUserID(ctx context.Context, userID int64) (Peer, error) {
	peer, err := s.store.GetActivePeerByUserID(ctx, userID)
	if err != nil {
		return Peer{}, err
	}

	if !peer.IsActive {
		return Peer{}, ErrPeerNotActive
	}

	return peer, nil
}

func (s *service) ActivatePeer(ctx context.Context, userID int64) error {
	return s.store.SetActive(ctx, userID)
}

func (s *service) GetByID(ctx context.Context, peerID string) (Peer, error) {
	return s.store.GetByID(ctx, peerID)
}

func (s *service) DeletePeersByUserID(ctx context.Context, userID int64) error {
	return s.store.DeletePeersByUserID(ctx, userID)
}
