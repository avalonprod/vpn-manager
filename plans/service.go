package plans

import (
	"context"
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
	return s.store.GetAll(ctx)
}

func (s *service) GetByID(ctx context.Context, ID string) (Plan, error) {
	return s.store.GetByID(ctx, ID)
}
