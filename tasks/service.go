package tasks

import (
	"context"
	"time"
)

type IStore interface {
	Enqueue(ctx context.Context, t Task) error
	ClaimDue(ctx context.Context) (*Task, error)
	Reschedule(ctx context.Context, ID string, runAt time.Time) error
	SetStatus(ctx context.Context, ID string, status Status) error
}

type service struct {
	store IStore
}

func NewService(store IStore) *service {
	return &service{
		store: store,
	}
}

func (s *service) Enqueue(ctx context.Context, task Task) error {
	return s.store.Enqueue(ctx, task)
}

func (s *service) ClaimDue(ctx context.Context) (*Task, error) {
	return s.store.ClaimDue(ctx)
}

func (s *service) MarkDone(ctx context.Context, ID string) error {
	return s.store.SetStatus(ctx, ID, StatusDone)
}

func (s *service) Reschedule(ctx context.Context, ID string, runAt time.Time) error {
	return s.store.Reschedule(ctx, ID, runAt)
}

func (s *service) MarkFailed(ctx context.Context, ID string) error {
	return s.store.SetStatus(ctx, ID, StatusFailed)
}
