package bot

import (
	"context"
	"vpn-manager/pkg/logger"
	"vpn-manager/plans"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"gopkg.in/telebot.v4"
)

type IUsersService interface {
	Register(ctx context.Context, input users.CreateUserInput) error
}

type ISubscriptionsService interface {
	CreateTrialSubscription(ctx context.Context, userID int64) error
	GetByUserID(ctx context.Context, userID int64) (subscriptions.Subscription, error)
	CancelSubscription(ctx context.Context, userID int64) error
}

type IServersService interface {
	RegisterNewPeers(ctx context.Context, userID int64) error
}

type IPeersService interface {
	ActivatePeer(ctx context.Context, userID int64) error
}

type IPlansService interface {
	GetAll(ctx context.Context) ([]plans.Plan, error)
	GetByID(ctx context.Context, ID string) (plans.Plan, error)
}

type Bot struct {
	bot                  *telebot.Bot
	stackStore           StackStore
	logger               logger.ILogger
	usersService         IUsersService
	serversService       IServersService
	peersService         IPeersService
	plansService         IPlansService
	subscriptionsService ISubscriptionsService
	apiUrl               string
}

func NewBot(
	bot *telebot.Bot,
	stackStore StackStore,
	logger logger.ILogger,
	usersService IUsersService,
	serversService IServersService,
	peersService IPeersService,
	plansService IPlansService,
	subscriptionsService ISubscriptionsService,
	apiUrl string,
) *Bot {
	return &Bot{
		bot:                  bot,
		stackStore:           stackStore,
		logger:               logger,
		usersService:         usersService,
		serversService:       serversService,
		peersService:         peersService,
		plansService:         plansService,
		subscriptionsService: subscriptionsService,
		apiUrl:               apiUrl,
	}
}

func (b *Bot) Run() {

	handler := b.bot.Group()

	handler.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			b.pushMessage(c.Sender().ID, c.Message())
			return next(c)
		}
	})
	handler.Handle(telebot.OnText, func(c telebot.Context) error {
		c.Delete()
		return nil
	})

	handler.Handle("/start", b.handleStart)
	handler.Handle(&telebot.Btn{Unique: TrialAccessScreen}, b.handleTrialAccess)
	handler.Handle(&telebot.Btn{Unique: AppListScreen}, b.handleAppsList)
	handler.Handle(&telebot.Btn{Unique: SubscriptionsScreen}, b.handleSubscribe)
	handler.Handle(&telebot.Btn{Unique: SuccessScreen}, b.handleSuccess)
	handler.Handle(&telebot.Btn{Unique: ManualSetupScreen}, b.handleManualSetup)
	handler.Handle(&telebot.Btn{Unique: CancelSubscriptionScreen}, b.handleCancelSubscription)
	handler.Handle(&telebot.Btn{Unique: SubscriptionManagementScreen}, b.handleSubscriptionManagement)
	handler.Handle(&telebot.Btn{Unique: "back"}, b.handleBack)

	b.bot.Start()
}
