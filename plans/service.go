package plans

import (
	"context"
	"sort"
)

type IStore interface {
	GetAll(ctx context.Context) ([]Plan, error)
	GetByID(ctx context.Context, ID string) (Plan, error)
}

type service struct {
	store IStore
}

func NewService(store IStore) *service {
	return &service{
		store: store,
	}
}

func (s *service) GetAll(ctx context.Context) ([]Plan, error) {
	plans, err := s.store.GetAll(ctx)
	if err != nil {
		return []Plan{}, err
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Order < plans[j].Order
	})

	return plans, nil
}

func (s *service) GetByID(ctx context.Context, ID string) (Plan, error) {
	return s.store.GetByID(ctx, ID)
}
