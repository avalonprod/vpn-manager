package users

import (
	"context"
	"errors"
	"time"
)

type IStore interface {
	Create(ctx context.Context, user User) error
	GetAll(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, ID int64) (User, error)
	List(ctx context.Context, f ListFilter) ([]User, error)
	Count(ctx context.Context, f ListFilter) (int64, error)
	CountUsers(ctx context.Context) (int64, error)
	CountBlocked(ctx context.Context) (int64, error)
	CountCreatedSince(ctx context.Context, since time.Time) (int64, error)
	CountActiveSince(ctx context.Context, since time.Time) (int64, error)
	SetBlocked(ctx context.Context, userID int64, blocked bool, reason string) error
	TouchLastActive(ctx context.Context, userID int64) error
	SignupsByDay(ctx context.Context, since time.Time) ([]DailyCount, error)
	GetManyByIDs(ctx context.Context, ids []int64) (map[int64]User, error)
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

	err := s.store.Create(ctx, user)
	if errors.Is(err, ErrUserAlreadyExists) {
		// Уже знакомый пользователь — просто отмечаем активность.
		return s.store.TouchLastActive(ctx, input.ID)
	}

	return err
}

func (s *service) GetByID(ctx context.Context, ID int64) (User, error) {
	return s.store.GetByID(ctx, ID)
}

func (s *service) GetAll(ctx context.Context) ([]User, error) {
	return s.store.GetAll(ctx)
}

func (s *service) List(ctx context.Context, f ListFilter) ([]User, int64, error) {
	list, err := s.store.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.store.Count(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (s *service) SetBlocked(ctx context.Context, userID int64, blocked bool, reason string) error {
	return s.store.SetBlocked(ctx, userID, blocked, reason)
}

// IsBlocked возвращает true только для существующего заблокированного
// пользователя: неизвестный ID не считается заблокированным.
func (s *service) IsBlocked(ctx context.Context, userID int64) (bool, error) {
	user, err := s.store.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}

	return user.IsBlocked, nil
}

func (s *service) TouchLastActive(ctx context.Context, userID int64) error {
	return s.store.TouchLastActive(ctx, userID)
}

func (s *service) SignupsByDay(ctx context.Context, since time.Time) ([]DailyCount, error) {
	return s.store.SignupsByDay(ctx, since)
}

func (s *service) GetManyByIDs(ctx context.Context, ids []int64) (map[int64]User, error) {
	return s.store.GetManyByIDs(ctx, ids)
}

func (s *service) Totals(ctx context.Context) (Totals, error) {
	var totals Totals
	var err error

	if totals.Total, err = s.store.CountUsers(ctx); err != nil {
		return Totals{}, err
	}

	if totals.Blocked, err = s.store.CountBlocked(ctx); err != nil {
		return Totals{}, err
	}

	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if totals.NewToday, err = s.store.CountCreatedSince(ctx, startOfDay); err != nil {
		return Totals{}, err
	}

	if totals.Active24, err = s.store.CountActiveSince(ctx, now.Add(-24*time.Hour)); err != nil {
		return Totals{}, err
	}

	if totals.Active7d, err = s.store.CountActiveSince(ctx, now.AddDate(0, 0, -7)); err != nil {
		return Totals{}, err
	}

	return totals, nil
}
