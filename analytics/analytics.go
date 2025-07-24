package analytics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"vpn-manager/payments"
	"vpn-manager/pkg/logger"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"gopkg.in/telebot.v4"
)

type IUsersService interface {
	GetAll(ctx context.Context) ([]users.User, error)
}

type ISubscriptionsService interface {
	GetAllTrialSubscriptions(ctx context.Context) ([]subscriptions.Subscription, error)
}

type IPaymentsService interface {
	GetAllCompletedInvoices(ctx context.Context) ([]payments.Invoice, error)
}

type Analytics struct {
	bot                  *telebot.Bot
	userService          IUsersService
	subscriptionsService ISubscriptionsService
	paymentsService      IPaymentsService
	logger               logger.ILogger
	analyticsChatID      int64
	messageID            int
	mu                   sync.Mutex
}

func NewAnalytics(bot *telebot.Bot, userService IUsersService, subscriptionsService ISubscriptionsService, paymentsService IPaymentsService, logger logger.ILogger, analyticsChatID int64) *Analytics {
	return &Analytics{
		bot:                  bot,
		userService:          userService,
		subscriptionsService: subscriptionsService,
		paymentsService:      paymentsService,
		logger:               logger,
		analyticsChatID:      analyticsChatID,
		mu:                   sync.Mutex{},
	}
}

func (a *Analytics) UpdateAnalyticsData(ctx context.Context) {
	users, err := a.userService.GetAll(ctx)
	if err != nil {
		a.logger.Error("Failed to get users:", err)
		return
	}

	subscriptions, err := a.subscriptionsService.GetAllTrialSubscriptions(ctx)
	if err != nil {
		a.logger.Error("Failed to get trial subscriptions:", err)
		return
	}

	invoices, err := a.paymentsService.GetAllCompletedInvoices(ctx)
	if err != nil {
		a.logger.Error("Failed to get completed invoices:", err)
		return
	}

	message := fmt.Sprintf(`
📊 Статистика VPN

👥 Пользователи: %d
🎁 Пробные подписки: %d
💳 Оплаченные подписки: %d
	`, len(users), len(subscriptions), len(invoices))
	if a.messageID == 0 {
		msg, err := a.bot.Send(&telebot.Chat{ID: a.analyticsChatID}, message)
		if err != nil {
			a.logger.Error("Failed to send analytics message:", err)
		}

		a.mu.Lock()
		a.messageID = msg.ID
		a.mu.Unlock()

		return
	}

	_, err = a.bot.Edit(&telebot.Message{
		ID: a.messageID,
		Chat: &telebot.Chat{
			ID: a.analyticsChatID,
		},
	}, message)
	if err != nil {
		if errors.Is(err, telebot.ErrMessageNotModified) {
			return
		}
		a.logger.Error("Failed to edit analytics message:", err)

		return
	}
}
