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
	Deactivate(ctx context.Context, userID int64) error
	SetImported(ctx context.Context, userID int64, val bool, importedAt time.Time) error
	Totals(ctx context.Context) (Totals, error)
	CountByLocation(ctx context.Context) ([]LocationCount, error)
	GetPeersForUsers(ctx context.Context, userIDs []int64) (map[int64]Peer, error)
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
				UserID:     userID,
				Email:      email,
				UUID:       uuid,
				IsActive:   true,
				CreatedAt:  time.Now().UTC(),
				IsImported: false,
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

func (s *service) DeactivatePeer(ctx context.Context, userID int64) error {
	return s.store.Deactivate(ctx, userID)
}

func (s *service) GetByID(ctx context.Context, peerID string) (Peer, error) {
	return s.store.GetByID(ctx, peerID)
}

func (s *service) DeletePeersByUserID(ctx context.Context, userID int64) error {
	return s.store.DeletePeersByUserID(ctx, userID)
}

func (s *service) SetImported(ctx context.Context, userID int64) error {
	return s.store.SetImported(ctx, userID, true, time.Now().UTC())
}

func (s *service) Totals(ctx context.Context) (Totals, error) {
	return s.store.Totals(ctx)
}

func (s *service) CountByLocation(ctx context.Context) ([]LocationCount, error) {
	return s.store.CountByLocation(ctx)
}

func (s *service) GetPeersForUsers(ctx context.Context, userIDs []int64) (map[int64]Peer, error) {
	return s.store.GetPeersForUsers(ctx, userIDs)
}
