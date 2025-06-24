package peers

import (
	"context"
	"time"
)

type IStore interface {
	Create(ctx context.Context, peer Peer) error
	GetByID(ctx context.Context, ID string) (Peer, error)
	DeletePeersByUserID(ctx context.Context, userID int64) error
	GetPeersByUserID(ctx context.Context, userID int64) ([]Peer, error)
}

type service struct {
	store IStore
}

func NewService(store IStore) *service {
	return &service{
		store: store,
	}
}

type CreatePeerInput struct {
	UserId        int64
	ServerId      string
	Location      string
	ConnectionURI string
}

func (s *service) CreatePeer(ctx context.Context, input CreatePeerInput) error {
	return s.store.Create(ctx, Peer{
		UserID:        input.UserId,
		ServerID:      input.ServerId,
		Location:      input.Location,
		ConnectionURI: input.ConnectionURI,
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *service) GetPeersByUserID(ctx context.Context, userID int64) ([]Peer, error) {
	return s.store.GetPeersByUserID(ctx, userID)
}

func (s *service) GetByID(ctx context.Context, peerID string) (Peer, error) {
	return s.store.GetByID(ctx, peerID)
}

func (s *service) DeletePeersByUserID(ctx context.Context, userID int64) error {
	return s.store.DeletePeersByUserID(ctx, userID)
}
