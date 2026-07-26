package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vpn-manager/payments"
	"vpn-manager/plans"

	"github.com/gorilla/mux"
)

type planResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	SubTitle     string    `json:"sub_title"`
	Price        float64   `json:"price"`
	Currency     string    `json:"currency"`
	DurationDays int       `json:"duration_days"`
	IsActive     bool      `json:"is_active"`
	Order        int       `json:"order"`
	CreatedAt    time.Time `json:"created_at"`
}

func toPlanResponse(plan plans.Plan) planResponse {
	return planResponse{
		ID:           plan.ID,
		Title:        plan.Title,
		SubTitle:     plan.SubTitle,
		Price:        plan.Price,
		Currency:     plan.Currency,
		DurationDays: plan.DurationDays,
		IsActive:     plan.IsActive,
		Order:        plan.Order,
		CreatedAt:    plan.CreatedAt,
	}
}

func (h *Handler) handleListPlans(w http.ResponseWriter, r *http.Request) {
	list, err := h.plansService.GetAllIncludingInactive(r.Context())
	if err != nil {
		h.logger.Errorf("admin: failed to list plans: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load plans")
		return
	}

	// Дополняем тариф числом активных подписок — так видно, что удалять нельзя.
	subsByPlan := map[string]int64{}
	if counts, err := h.subscriptionsService.CountByPlan(r.Context()); err == nil {
		for _, c := range counts {
			subsByPlan[c.PlanID] = c.Count
		}
	} else {
		h.logger.Errorf("admin: failed to count subscriptions by plan: %v", err)
	}

	type item struct {
		planResponse
		ActiveSubscriptions int64 `json:"active_subscriptions"`
	}

	items := make([]item, 0, len(list))
	for _, plan := range list {
		items = append(items, item{
			planResponse:        toPlanResponse(plan),
			ActiveSubscriptions: subsByPlan[plan.ID],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createPlanRequest struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	SubTitle     string  `json:"sub_title"`
	Price        float64 `json:"price"`
	Currency     string  `json:"currency"`
	DurationDays int     `json:"duration_days"`
	IsActive     bool    `json:"is_active"`
	Order        int     `json:"order"`
}

func (h *Handler) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plan, err := h.plansService.Create(r.Context(), plans.CreateInput{
		ID:           req.ID,
		Title:        req.Title,
		SubTitle:     req.SubTitle,
		Price:        req.Price,
		Currency:     req.Currency,
		DurationDays: req.DurationDays,
		IsActive:     req.IsActive,
		Order:        req.Order,
	})
	if err != nil {
		switch {
		case errors.Is(err, plans.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, plans.ErrPlanAlreadyExists):
			writeError(w, http.StatusConflict, "plan with this id already exists")
		default:
			h.logger.Errorf("admin: failed to create plan: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to create plan")
		}
		return
	}

	claims, _ := ClaimsFrom(r.Context())
	h.audit(r.Context(), claims.Subject, clientIP(r), "plan.create", plan.ID, plan.Title)

	writeJSON(w, http.StatusCreated, toPlanResponse(plan))
}

type updatePlanRequest struct {
	Title        *string  `json:"title"`
	SubTitle     *string  `json:"sub_title"`
	Price        *float64 `json:"price"`
	Currency     *string  `json:"currency"`
	DurationDays *int     `json:"duration_days"`
	IsActive     *bool    `json:"is_active"`
	Order        *int     `json:"order"`
}

func (h *Handler) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	planID := mux.Vars(r)["id"]

	var req updatePlanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plan, err := h.plansService.Update(r.Context(), planID, plans.UpdateInput{
		Title:        req.Title,
		SubTitle:     req.SubTitle,
		Price:        req.Price,
		Currency:     req.Currency,
		DurationDays: req.DurationDays,
		IsActive:     req.IsActive,
		Order:        req.Order,
	})
	if err != nil {
		switch {
		case errors.Is(err, plans.ErrPlanNotFound):
			writeError(w, http.StatusNotFound, "plan not found")
		case errors.Is(err, plans.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.Errorf("admin: failed to update plan %s: %v", planID, err)
			writeError(w, http.StatusInternalServerError, "failed to update plan")
		}
		return
	}

	claims, _ := ClaimsFrom(r.Context())
	h.audit(r.Context(), claims.Subject, clientIP(r), "plan.update", plan.ID, plan.Title)

	writeJSON(w, http.StatusOK, toPlanResponse(plan))
}

// handleDeletePlan запрещает удалять тариф, на котором висят активные
// подписки: иначе продления и счета остались бы без цены.
func (h *Handler) handleDeletePlan(w http.ResponseWriter, r *http.Request) {
	planID := mux.Vars(r)["id"]

	counts, err := h.subscriptionsService.CountByPlan(r.Context())
	if err != nil {
		h.logger.Errorf("admin: failed to count subscriptions by plan: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete plan")
		return
	}

	for _, c := range counts {
		if c.PlanID == planID && c.Count > 0 {
			writeError(w, http.StatusConflict,
				"plan has "+strconv.FormatInt(c.Count, 10)+" active subscriptions, deactivate it instead")
			return
		}
	}

	if err := h.plansService.Delete(r.Context(), planID); err != nil {
		if errors.Is(err, plans.ErrPlanNotFound) {
			writeError(w, http.StatusNotFound, "plan not found")
			return
		}
		h.logger.Errorf("admin: failed to delete plan %s: %v", planID, err)
		writeError(w, http.StatusInternalServerError, "failed to delete plan")
		return
	}

	claims, _ := ClaimsFrom(r.Context())
	h.audit(r.Context(), claims.Subject, clientIP(r), "plan.delete", planID, "")

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type paymentResponse struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	PlanID    string    `json:"plan_id"`
	PlanTitle string    `json:"plan_title,omitempty"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) handleListPayments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, offset := pagination(r)

	filter := payments.ListFilter{
		Status: strings.TrimSpace(query.Get("status")),
		Limit:  limit,
		Offset: offset,
	}

	if raw := query.Get("user_id"); raw != "" {
		userID, err := userIDParam(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user id")
			return
		}
		filter.UserID = userID
	}

	switch filter.Status {
	case "", payments.StatusCreated, payments.StatusPending, payments.StatusCompleted:
	default:
		writeError(w, http.StatusBadRequest, "invalid status filter")
		return
	}

	invoices, total, err := h.paymentsService.List(r.Context(), filter)
	if err != nil {
		h.logger.Errorf("admin: failed to list payments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load payments")
		return
	}

	planList, err := h.plansService.GetAllIncludingInactive(r.Context())
	if err != nil {
		h.logger.Errorf("admin: failed to load plans: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load payments")
		return
	}

	planByID := make(map[string]plans.Plan, len(planList))
	for _, plan := range planList {
		planByID[plan.ID] = plan
	}

	items := make([]paymentResponse, 0, len(invoices))
	for _, invoice := range invoices {
		plan := planByID[invoice.PlanID]
		items = append(items, paymentResponse{
			ID:        invoice.ID,
			UserID:    invoice.UserID,
			PlanID:    invoice.PlanID,
			PlanTitle: plan.Title,
			Price:     plan.Price,
			Currency:  plan.Currency,
			Status:    invoice.Status,
			CreatedAt: invoice.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, page[paymentResponse]{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
