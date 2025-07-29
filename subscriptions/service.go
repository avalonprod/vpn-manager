package subscriptions

import (
	"context"
	"errors"
	"time"
	"vpn-manager/payments"
	"vpn-manager/plans"
)

type IStore interface {
	Create(ctx context.Context, subscription Subscription) error
	Update(ctx context.Context, userID int64, ID string, input Subscription) error
	DeactivateExpiredSubscriptions(ctx context.Context) error
	GetExpiredSubscriptions(ctx context.Context) ([]Subscription, error)
	GetByUserID(ctx context.Context, userID int64) (*Subscription, error)
	DeactivateSubscription(ctx context.Context, userID int64, ID string) error
	CancelSubscription(ctx context.Context, userID int64) error
	GetAllTrialSubscriptions(ctx context.Context) ([]Subscription, error)
}

type IPlansService interface {
	GetByID(ctx context.Context, ID string) (plans.Plan, error)
}

type IPaymentsService interface {
	GetInvoiceByID(ctx context.Context, userID int64, ID string) (payments.Invoice, error)
	SetStatus(ctx context.Context, userID int64, ID, status string) error
}

type service struct {
	store           IStore
	plansService    IPlansService
	paymentsService IPaymentsService
}

func NewService(store IStore, plansService IPlansService, paymentsService IPaymentsService) *service {
	return &service{
		store:           store,
		plansService:    plansService,
		paymentsService: paymentsService,
	}
}

func (s *service) CreateOrExtend(ctx context.Context, userID int64, invoiceID string) error {
	invoice, err := s.paymentsService.GetInvoiceByID(ctx, userID, invoiceID)
	if err != nil {
		return err
	}

	if invoice.Status == payments.StatusCompleted {
		return nil
	}

	plan, err := s.plansService.GetByID(ctx, invoice.PlanID)
	if err != nil {
		return err
	}

	subscription, err := s.store.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, ErrSubscriptionNotFound) {
		return err
	}

	if errors.Is(err, ErrSubscriptionNotFound) {
		if err := s.store.Create(ctx, Subscription{
			UserID:      userID,
			PlanID:      plan.ID,
			Active:      true,
			AutoRenewal: true,
			IsTrial:     false,
			ExpiresAt:   time.Now().UTC().Add(time.Duration(plan.DurationDays) * 24 * time.Hour),
			CreatedAt:   time.Now().UTC(),
		}); err != nil {
			return err
		}
	} else {
		if err := s.store.Update(ctx, userID, subscription.ID, Subscription{
			PlanID:      plan.ID,
			Active:      true,
			AutoRenewal: true,
			IsTrial:     false,
			ExpiresAt:   subscription.ExpiresAt.Add(time.Duration(plan.DurationDays) * 24 * time.Hour),
		}); err != nil {
			return err
		}
	}

	err = s.paymentsService.SetStatus(ctx, userID, invoiceID, payments.StatusCompleted)

	return err
}

func (s *service) CreateTrialSubscription(ctx context.Context, userID int64) error {
	sub, err := s.store.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, ErrSubscriptionNotFound) {
		return err
	}

	if sub != nil {
		return nil
	}

	err = s.store.Create(ctx, Subscription{
		UserID:      userID,
		PlanID:      "trial",
		Active:      true,
		AutoRenewal: false,
		IsTrial:     true,
		ExpiresAt:   time.Now().UTC().Add(3 * 24 * time.Hour),
		CreatedAt:   time.Now().UTC(),
	})

	return err
}

func (s *service) GetByUserID(ctx context.Context, userID int64) (Subscription, error) {
	subscription, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return Subscription{}, err
	}

	return *subscription, nil
}

func (s *service) GetExpiredSubscriptions(ctx context.Context) ([]Subscription, error) {
	return s.store.GetExpiredSubscriptions(ctx)
}

func (s *service) DeactivateExpiredSubscriptions(ctx context.Context) error {
	return s.store.DeactivateExpiredSubscriptions(ctx)
}

func (s *service) IsSubscriptionActive(ctx context.Context, userID int64) (bool, error) {
	subscription, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return false, ErrSubscriptionNotFound
		}
		return false, err
	}

	if !subscription.Active {
		return false, nil
	}

	return true, nil
}

func (s *service) CancelSubscription(ctx context.Context, userID int64) error {
	return s.store.CancelSubscription(ctx, userID)
}
