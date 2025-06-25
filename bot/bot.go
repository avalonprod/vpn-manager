package bot

import (
	"context"
	"vpn-manager/pkg/logger"
	"vpn-manager/servers"
	"vpn-manager/users"

	"gopkg.in/telebot.v4"
)

type IUsersService interface {
	Register(ctx context.Context, input users.CreateUserInput) error
}

type ISubscriptionsService interface {
	CreateTrialSubscription(ctx context.Context, userID int64) error
}

type IServersService interface {
	GetAllActiveServers(ctx context.Context) ([]servers.Server, error)
}

type Bot struct {
	bot                  *telebot.Bot
	logger               logger.ILogger
	usersService         IUsersService
	serversService       IServersService
	subscriptionsService ISubscriptionsService
	apiUrl               string
}

func NewBot(
	bot *telebot.Bot,
	logger logger.ILogger,
	usersService IUsersService,
	serversService IServersService,
	subscriptionsService ISubscriptionsService,
	apiUrl string,
) *Bot {
	return &Bot{
		bot:                  bot,
		logger:               logger,
		usersService:         usersService,
		serversService:       serversService,
		subscriptionsService: subscriptionsService,
		apiUrl:               apiUrl,
	}
}

func (b *Bot) Run() {

	handler := b.bot.Group()

	handler.Use(b.pushScreen)
	handler.Handle("/start", b.handleStart)
	handler.Handle(&trialAccessButton, b.handleTrialAccess)
	handler.Handle(&appsListButton, b.handleAppsList)
	handler.Handle(&subscribeButton, b.handleSubscribe)
	handler.Handle(&renewSubscriptionButton, b.handleRenewSubscribe)
	handler.Handle(&successButton, b.handleSuccess)
	handler.Handle(&supportButton, b.handleSupport)

	handler.Handle(&telebot.Btn{Unique: "back"}, b.handleBack)

	b.bot.Start()
}

func (b *Bot) pushScreen(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if msg := c.Message(); msg != nil {
			b.pushMessage(c.Sender().ID, msg)
		}
		if cb := c.Callback(); cb != nil {
			b.pushMessage(c.Sender().ID, cb.Message)
		}
		return next(c)
	}
}

func (b *Bot) replyError(c telebot.Context, msg string) error {
	return c.Send(msg)
}
