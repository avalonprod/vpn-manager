package plans

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type IStore interface {
	GetAll(ctx context.Context) ([]Plan, error)
	GetAllIncludingInactive(ctx context.Context) ([]Plan, error)
	GetByID(ctx context.Context, ID string) (Plan, error)
	Create(ctx context.Context, plan Plan) (string, error)
	Update(ctx context.Context, ID string, fields bson.M) error
	Delete(ctx context.Context, ID string) error
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

func (s *service) GetAllIncludingInactive(ctx context.Context) ([]Plan, error) {
	plans, err := s.store.GetAllIncludingInactive(ctx)
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

func validateCreateInput(input CreateInput) error {
	switch {
	case strings.TrimSpace(input.Title) == "":
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	case input.Price < 0:
		return fmt.Errorf("%w: price must not be negative", ErrInvalidInput)
	case input.DurationDays <= 0:
		return fmt.Errorf("%w: duration_days must be positive", ErrInvalidInput)
	case strings.TrimSpace(input.Currency) == "":
		return fmt.Errorf("%w: currency is required", ErrInvalidInput)
	}

	return nil
}

func (s *service) Create(ctx context.Context, input CreateInput) (Plan, error) {
	if err := validateCreateInput(input); err != nil {
		return Plan{}, err
	}

	plan := Plan{
		ID:           strings.TrimSpace(input.ID),
		Title:        strings.TrimSpace(input.Title),
		SubTitle:     strings.TrimSpace(input.SubTitle),
		Price:        input.Price,
		Currency:     strings.ToUpper(strings.TrimSpace(input.Currency)),
		DurationDays: input.DurationDays,
		IsActive:     input.IsActive,
		Order:        input.Order,
		CreatedAt:    time.Now().UTC(),
	}

	id, err := s.store.Create(ctx, plan)
	if err != nil {
		return Plan{}, err
	}

	plan.ID = id

	return plan, nil
}

func (s *service) Update(ctx context.Context, ID string, input UpdateInput) (Plan, error) {
	fields := bson.M{}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return Plan{}, fmt.Errorf("%w: title must not be empty", ErrInvalidInput)
		}
		fields["title"] = title
	}
	if input.SubTitle != nil {
		fields["sub_title"] = strings.TrimSpace(*input.SubTitle)
	}
	if input.Price != nil {
		if *input.Price < 0 {
			return Plan{}, fmt.Errorf("%w: price must not be negative", ErrInvalidInput)
		}
		fields["price"] = *input.Price
	}
	if input.Currency != nil {
		fields["currency"] = strings.ToUpper(strings.TrimSpace(*input.Currency))
	}
	if input.DurationDays != nil {
		if *input.DurationDays <= 0 {
			return Plan{}, fmt.Errorf("%w: duration_days must be positive", ErrInvalidInput)
		}
		fields["duration_days"] = *input.DurationDays
	}
	if input.IsActive != nil {
		fields["is_active"] = *input.IsActive
	}
	if input.Order != nil {
		fields["order"] = *input.Order
	}

	if err := s.store.Update(ctx, ID, fields); err != nil {
		return Plan{}, err
	}

	return s.store.GetByID(ctx, ID)
}

func (s *service) Delete(ctx context.Context, ID string) error {
	return s.store.Delete(ctx, ID)
}
