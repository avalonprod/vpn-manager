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
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle(&trialAccessButton, b.handleTrialAccess)
	b.bot.Handle(&appsListButton, b.handleAppsList)
	b.bot.Handle(&subscribeButton, b.handleSubscribe)
	b.bot.Handle(&renewSubscriptionButton, b.handleSubscribe)
	b.bot.Handle(&successButton, b.handleSuccess)

	b.bot.Handle(&backButton, b.handleBack)

	b.bot.Start()
}

func (b *Bot) replyError(c telebot.Context, msg string) error {
	return c.Send(msg)
}
