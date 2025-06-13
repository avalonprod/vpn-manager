package users

import (
	"context"
	"time"
)

type IStore interface {
	Create(ctx context.Context, user User) error
}

type service struct {
	store IStore
}

func NewService(store IStore) *service {
	return &service{
		store: store,
	}
}

type CreateUserInput struct {
	ID        int64  `bson:"origin_id"`
	Username  string `bson:"username"`
	FirstName string `bson:"first_name"`
}

func (s *service) Register(ctx context.Context, input CreateUserInput) error {
	user := User{
		ID:           input.ID,
		Username:     input.Username,
		FirstName:    input.FirstName,
		CreatedAt:    time.Now().UTC(),
		LastActiveAt: time.Now().UTC(),
	}

	if err := s.store.Create(ctx, user); err != nil && err != ErrUserAlreadyExists {
		return err
	}

	return nil
}
