package admin

import (
	"net/http"
	"sort"
	"time"
	"vpn-manager/plans"
)

func sortPlanBreakdown(items []planBreakdown) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Subscriptions != items[j].Subscriptions {
			return items[i].Subscriptions > items[j].Subscriptions
		}
		return items[i].PlanID < items[j].PlanID
	})
}

type overviewUsers struct {
	Total    int64 `json:"total"`
	NewToday int64 `json:"new_today"`
	Active24 int64 `json:"active_24h"`
	Active7d int64 `json:"active_7d"`
	Blocked  int64 `json:"blocked"`
}

type overviewSubscriptions struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	ActiveTrial  int64 `json:"active_trial"`
	ActivePaid   int64 `json:"active_paid"`
	AutoRenewal  int64 `json:"auto_renewal"`
	ExpiringIn3d int64 `json:"expiring_in_3d"`
}

type overviewRevenue struct {
	Total     float64 `json:"total"`
	Last30d   float64 `json:"last_30d"`
	Last7d    float64 `json:"last_7d"`
	Today     float64 `json:"today"`
	Invoices  int64   `json:"invoices"`
	Completed int64   `json:"completed"`
	Pending   int64   `json:"pending"`
	ARPU      float64 `json:"arpu"`
}

type overviewPeers struct {
	Total      int64   `json:"total"`
	Active     int64   `json:"active"`
	Imported   int64   `json:"imported"`
	ImportRate float64 `json:"import_rate"`
}

type overviewServers struct {
	Total  int64 `json:"total"`
	Active int64 `json:"active"`
}

type overviewResponse struct {
	Users          overviewUsers         `json:"users"`
	Subscriptions  overviewSubscriptions `json:"subscriptions"`
	Revenue        overviewRevenue       `json:"revenue"`
	Peers          overviewPeers         `json:"peers"`
	Servers        overviewServers       `json:"servers"`
	ConversionRate float64               `json:"conversion_rate"`
	GeneratedAt    time.Time             `json:"generated_at"`
}

func percent(part, total int64) float64 {
	if total == 0 {
		return 0
	}

	return round2(float64(part) / float64(total) * 100)
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func (h *Handler) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	userTotals, err := h.usersService.Totals(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load user totals: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	subTotals, err := h.subscriptionsService.Totals(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load subscription totals: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	paymentTotals, err := h.paymentsService.Totals(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load payment totals: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	peerTotals, err := h.peersService.Totals(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load peer totals: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	serversTotal, serversActive, err := h.serversService.Count(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to count servers: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	revenue30d, err := h.paymentsService.RevenueSince(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		h.logger.Errorf("admin: failed to load 30d revenue: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	revenue7d, err := h.paymentsService.RevenueSince(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		h.logger.Errorf("admin: failed to load 7d revenue: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	revenueToday, err := h.paymentsService.RevenueSince(ctx, startOfDay)
	if err != nil {
		h.logger.Errorf("admin: failed to load today revenue: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	response := overviewResponse{
		Users: overviewUsers{
			Total:    userTotals.Total,
			NewToday: userTotals.NewToday,
			Active24: userTotals.Active24,
			Active7d: userTotals.Active7d,
			Blocked:  userTotals.Blocked,
		},
		Subscriptions: overviewSubscriptions{
			Total:        subTotals.Total,
			Active:       subTotals.Active,
			ActiveTrial:  subTotals.ActiveTrial,
			ActivePaid:   subTotals.ActivePaid,
			AutoRenewal:  subTotals.AutoRenewal,
			ExpiringIn3d: subTotals.ExpiringIn3d,
		},
		Revenue: overviewRevenue{
			Total:     round2(paymentTotals.Revenue),
			Last30d:   round2(revenue30d),
			Last7d:    round2(revenue7d),
			Today:     round2(revenueToday),
			Invoices:  paymentTotals.Total,
			Completed: paymentTotals.Completed,
			Pending:   paymentTotals.Pending,
		},
		Peers: overviewPeers{
			Total:      peerTotals.Total,
			Active:     peerTotals.Active,
			Imported:   peerTotals.Imported,
			ImportRate: percent(peerTotals.Imported, peerTotals.Total),
		},
		Servers: overviewServers{
			Total:  serversTotal,
			Active: serversActive,
		},

		ConversionRate: percent(subTotals.ActivePaid, userTotals.Total),
		GeneratedAt:    now,
	}

	if userTotals.Total > 0 {
		response.Revenue.ARPU = round2(paymentTotals.Revenue / float64(userTotals.Total))
	}

	writeJSON(w, http.StatusOK, response)
}

type timeseriesPoint struct {
	Date          string  `json:"date"`
	Signups       int64   `json:"signups"`
	Subscriptions int64   `json:"subscriptions"`
	Payments      int64   `json:"payments"`
	Revenue       float64 `json:"revenue"`
}

func (h *Handler) handleTimeseries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := daysParam(r)

	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))

	signups, err := h.usersService.SignupsByDay(ctx, start)
	if err != nil {
		h.logger.Errorf("admin: failed to load signups series: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	subs, err := h.subscriptionsService.CreatedByDay(ctx, start, nil)
	if err != nil {
		h.logger.Errorf("admin: failed to load subscriptions series: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	revenue, err := h.paymentsService.RevenueByDay(ctx, start)
	if err != nil {
		h.logger.Errorf("admin: failed to load revenue series: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	signupsByDate := make(map[string]int64, len(signups))
	for _, point := range signups {
		signupsByDate[point.Date] = point.Count
	}

	subsByDate := make(map[string]int64, len(subs))
	for _, point := range subs {
		subsByDate[point.Date] = point.Count
	}

	type revenuePoint struct {
		revenue float64
		count   int64
	}

	revenueByDate := make(map[string]revenuePoint, len(revenue))
	for _, point := range revenue {
		revenueByDate[point.Date] = revenuePoint{revenue: point.Revenue, count: point.Count}
	}

	points := make([]timeseriesPoint, 0, days)
	for i := range days {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		rev := revenueByDate[date]

		points = append(points, timeseriesPoint{
			Date:          date,
			Signups:       signupsByDate[date],
			Subscriptions: subsByDate[date],
			Payments:      rev.count,
			Revenue:       round2(rev.revenue),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": points,
		"days":  days,
		"from":  start.Format("2006-01-02"),
	})
}

type planBreakdown struct {
	PlanID        string  `json:"plan_id"`
	Title         string  `json:"title"`
	Subscriptions int64   `json:"subscriptions"`
	Revenue       float64 `json:"revenue"`
	Payments      int64   `json:"payments"`
}

type locationBreakdown struct {
	Location string `json:"location"`
	Peers    int64  `json:"peers"`
}

func (h *Handler) handleBreakdown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := daysParam(r)
	since := time.Now().UTC().AddDate(0, 0, -days)

	subsByPlan, err := h.subscriptionsService.CountByPlan(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load subscriptions by plan: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	revenueByPlan, err := h.paymentsService.RevenueByPlan(ctx, since)
	if err != nil {
		h.logger.Errorf("admin: failed to load revenue by plan: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	planList, err := h.plansService.GetAllIncludingInactive(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load plans: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	planByID := make(map[string]plans.Plan, len(planList))
	for _, plan := range planList {
		planByID[plan.ID] = plan
	}

	breakdown := make(map[string]*planBreakdown)

	get := func(planID string) *planBreakdown {
		if item, ok := breakdown[planID]; ok {
			return item
		}

		title := planByID[planID].Title
		if title == "" {

			title = planID
		}

		item := &planBreakdown{PlanID: planID, Title: title}
		breakdown[planID] = item

		return item
	}

	for _, c := range subsByPlan {
		get(c.PlanID).Subscriptions = c.Count
	}

	for _, c := range revenueByPlan {
		item := get(c.PlanID)
		item.Revenue = round2(c.Revenue)
		item.Payments = c.Count
	}

	plansResult := make([]planBreakdown, 0, len(breakdown))
	for _, item := range breakdown {
		plansResult = append(plansResult, *item)
	}

	sortPlanBreakdown(plansResult)

	locations, err := h.peersService.CountByLocation(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load peers by location: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	serverList, err := h.serversService.GetAll(ctx)
	if err != nil {
		h.logger.Errorf("admin: failed to load servers: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load analytics")
		return
	}

	locationByServer := make(map[string]string, len(serverList))
	for _, server := range serverList {
		locationByServer[server.ID] = server.Location
	}

	peersByLocation := make(map[string]int64, len(serverList))
	for _, c := range locations {
		location, exists := locationByServer[c.ServerID]
		if !exists {
			continue
		}
		peersByLocation[location] += c.Count
	}

	locationsResult := make([]locationBreakdown, 0, len(peersByLocation))
	for location, count := range peersByLocation {
		locationsResult = append(locationsResult, locationBreakdown{Location: location, Peers: count})
	}

	sort.Slice(locationsResult, func(i, j int) bool {
		if locationsResult[i].Peers != locationsResult[j].Peers {
			return locationsResult[i].Peers > locationsResult[j].Peers
		}
		return locationsResult[i].Location < locationsResult[j].Location
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"plans":     plansResult,
		"locations": locationsResult,
		"days":      days,
	})
}
