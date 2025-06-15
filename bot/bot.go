package bot

import (
	"context"
	"net/http"
	"vpn-manager/pkg/logger"
	"vpn-manager/servers"
	"vpn-manager/users"

	"github.com/go-telegram/bot"
	"github.com/gorilla/mux"
)

type IUsersService interface {
	Register(ctx context.Context, input users.CreateUserInput) error
}

type ISubscriptionsService interface {
	CreateTrialSubscription(ctx context.Context, userID int64) error
}

type IServersService interface {
	RegisterNewPeer(ctx context.Context, userID int64, serverID string) (servers.RegisterNewPeerOutput, error)
	GetAllServers(ctx context.Context) ([]servers.Server, error)
}

type TGBot struct {
	bot                  *bot.Bot
	logger               logger.ILogger
	usersService         IUsersService
	serversService       IServersService
	subscriptionsService ISubscriptionsService
}

func NewTGBot(bot *bot.Bot, logger logger.ILogger, usersService IUsersService, serversService IServersService, subscriptionsService ISubscriptionsService) *TGBot {
	return &TGBot{
		bot:                  bot,
		logger:               logger,
		usersService:         usersService,
		serversService:       serversService,
		subscriptionsService: subscriptionsService,
	}
}

func (tb *TGBot) Run(ctx context.Context, port string) {
	tb.bot.RegisterHandler(bot.HandlerTypeMessageText, StartCommand, bot.MatchTypeExact, tb.handleStart)
	tb.bot.RegisterHandler(bot.HandlerTypeMessageText, SupportCommand, bot.MatchTypeExact, tb.handleSupport)
	tb.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, ActivateTrialAccessCallback, bot.MatchTypeExact, tb.handleActivateTrialAccessCallback)
	tb.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, SupportCallback, bot.MatchTypeExact, tb.handleSupportCallback)
	tb.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, SubscribeCallback, bot.MatchTypeExact, tb.handleSubscribeCallback)
	tb.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, ConnectCallback, bot.MatchTypePrefix, tb.handleConnectCallback)
	tb.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, LocationListCallback, bot.MatchTypePrefix, tb.handleLocationsList)

	go tb.bot.StartWebhook(ctx)

	r := mux.NewRouter()

	r.HandleFunc("/webhook", tb.bot.WebhookHandler())

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	srv.ListenAndServe()
}
