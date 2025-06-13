package subscriptions

import (
	"context"
	"time"
)

type IStore interface {
	Create(ctx context.Context, subscription Subscription) error
	DeactivateExpiredSubscriptions(ctx context.Context) error
	GetExpiredSubscriptions(ctx context.Context) ([]Subscription, error)
	HasTrialSubscription(ctx context.Context, userID int64) (bool, error)
}

type service struct {
	store IStore
}

func NewService(store IStore) *service {
	return &service{
		store: store,
	}
}

func (s *service) CreateTrialSubscription(ctx context.Context, userID int64) error {
	isHas, err := s.store.HasTrialSubscription(ctx, userID)
	if err != nil {
		return err
	}

	if isHas {
		return ErrTrialAccessAlreadyActivated
	}

	startsAt := time.Now().UTC()
	err = s.store.Create(ctx, Subscription{
		UserID:    userID,
		Plan:      PlanTrial,
		Active:    true,
		StartsAt:  startsAt,
		ExpiresAt: startsAt.Add(3 * 24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	return err
}

func (s *service) GetExpiredSubscriptions(ctx context.Context) ([]Subscription, error) {
	return s.store.GetExpiredSubscriptions(ctx)
}

func (s *service) DeactivateExpiredSubscriptions(ctx context.Context) error {
	return s.store.DeactivateExpiredSubscriptions(ctx)
}
