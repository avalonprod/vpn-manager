package payments

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"vpn-manager/plans"
)

type IStore interface {
	Create(ctx context.Context, invoice Invoice) (string, error)
	GetByID(ctx context.Context, userID int64, ID string) (Invoice, error)
	SetStatus(ctx context.Context, userID int64, ID, status string) error
	GetAllCompletedInvoices(ctx context.Context) ([]Invoice, error)
	List(ctx context.Context, f ListFilter) ([]Invoice, int64, error)
	GetByUserID(ctx context.Context, userID int64, limit int) ([]Invoice, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	RevenueSince(ctx context.Context, since time.Time) (float64, error)
	RevenueByDay(ctx context.Context, since time.Time) ([]DailyRevenue, error)
	RevenueByPlan(ctx context.Context, since time.Time) ([]PlanRevenue, error)
}

type IPlansService interface {
	GetByID(ctx context.Context, ID string) (plans.Plan, error)
}

type CloudPaymentsConfig struct {
	PublicID  string
	SecretKey string
	ApiUrl    string
}

type service struct {
	store               IStore
	plansService        IPlansService
	cloudPaymentsConfig CloudPaymentsConfig
}

func NewService(store IStore, plansService IPlansService, cloudPaymentsConfig CloudPaymentsConfig) *service {
	return &service{
		store:               store,
		plansService:        plansService,
		cloudPaymentsConfig: cloudPaymentsConfig,
	}
}

type CreateOrderRequest struct {
	Amount      float64 `json:"Amount"`
	Currency    string  `json:"Currency"`
	Description string  `json:"Description"`
	AccountId   string  `json:"AccountId"`
	InvoiceId   string  `json:"InvoiceId"`
}

type CreateOrderResponse struct {
	Success bool `json:"Success"`
	Model   struct {
		Id  string `json:"Id"`
		Url string `json:"Url"`
	} `json:"Model"`
	Message string `json:"Message"`
}

type CreateInvoiceInput struct {
	PlanID string
	UserID int64
}

func (s *service) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (string, error) {
	client := http.Client{}

	plan, err := s.plansService.GetByID(ctx, input.PlanID)
	if err != nil {
		return "", fmt.Errorf("failed to find plan: %w", err)
	}

	invoiceID, err := s.store.Create(ctx, Invoice{
		PlanID:    plan.ID,
		UserID:    input.UserID,
		Status:    StatusCreated,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create invoice: %w", err)
	}

	reqUrl := fmt.Sprintf("%s/orders/create", s.cloudPaymentsConfig.ApiUrl)

	payload := CreateOrderRequest{
		Amount:      plan.Price,
		Currency:    plan.Currency,
		Description: plan.Title,
		AccountId:   strconv.Itoa(int(input.UserID)),
		InvoiceId:   invoiceID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, reqUrl, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(s.cloudPaymentsConfig.PublicID + ":" + s.cloudPaymentsConfig.SecretKey))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var res CreateOrderResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return "", fmt.Errorf("unmarshal error: %w", err)
	}

	if !res.Success {
		return "", fmt.Errorf("cloudpayments error: %s", res.Message)
	}

	if err := s.store.SetStatus(ctx, input.UserID, invoiceID, StatusPending); err != nil {
		return "", fmt.Errorf("failed to set invoice status pending: %w", err)
	}

	return res.Model.Url, nil
}

func (s *service) GetInvoiceByID(ctx context.Context, userID int64, ID string) (Invoice, error) {
	return s.store.GetByID(ctx, userID, ID)
}

func (s *service) SetStatus(ctx context.Context, userID int64, ID, status string) error {
	return s.store.SetStatus(ctx, userID, ID, status)
}

func (s *service) List(ctx context.Context, f ListFilter) ([]Invoice, int64, error) {
	return s.store.List(ctx, f)
}

func (s *service) GetByUserID(ctx context.Context, userID int64, limit int) ([]Invoice, error) {
	return s.store.GetByUserID(ctx, userID, limit)
}

func (s *service) RevenueByDay(ctx context.Context, since time.Time) ([]DailyRevenue, error) {
	return s.store.RevenueByDay(ctx, since)
}

func (s *service) RevenueByPlan(ctx context.Context, since time.Time) ([]PlanRevenue, error) {
	return s.store.RevenueByPlan(ctx, since)
}

func (s *service) Totals(ctx context.Context) (Totals, error) {
	var totals Totals
	var err error

	if totals.Total, err = s.store.CountByStatus(ctx, ""); err != nil {
		return Totals{}, err
	}

	if totals.Completed, err = s.store.CountByStatus(ctx, StatusCompleted); err != nil {
		return Totals{}, err
	}

	if totals.Pending, err = s.store.CountByStatus(ctx, StatusPending); err != nil {
		return Totals{}, err
	}

	if totals.Revenue, err = s.store.RevenueSince(ctx, time.Time{}); err != nil {
		return Totals{}, err
	}

	return totals, nil
}

func (s *service) RevenueSince(ctx context.Context, since time.Time) (float64, error) {
	return s.store.RevenueSince(ctx, since)
}
