package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vpn-manager/analytics"
	"vpn-manager/api"
	"vpn-manager/bot"
	"vpn-manager/core/config"
	"vpn-manager/notifier"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/pkg/db/mongodb"
	"vpn-manager/pkg/logger"
	"vpn-manager/pkg/server"
	"vpn-manager/plans"
	"vpn-manager/scheduler"
	"vpn-manager/servers"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"gopkg.in/telebot.v4"
)

const timeout = 5 * time.Second

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pref := telebot.Settings{
		Token: cfg.TelegramAccessToken,
		Poller: &telebot.Webhook{
			Listen:           fmt.Sprintf(":%s", cfg.TelegramWebhookPort),
			SecretToken:      cfg.TelegramWebhookToken,
			IgnoreSetWebhook: true,
			Endpoint: &telebot.WebhookEndpoint{
				PublicURL: fmt.Sprintf("%s/webhook", cfg.ApiUrl),
			},
		},
		ParseMode: telebot.ModeHTML,
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	mongodbClient, err := mongodb.NewConnection(cfg.MongoDB.URI, cfg.MongoDB.Username, cfg.MongoDB.Password)
	if err != nil {
		panic(err)
	}

	mongodb := mongodbClient.Database(cfg.MongoDB.Name)
	logger := logger.NewLogger()

	notifier := notifier.NewNotifier(b)
	usersService := users.NewService(users.NewStore(mongodb))
	peersService := peers.NewService(peers.NewStore(mongodb))
	serversService := servers.NewService(servers.NewStore(mongodb), peersService, cfg.ServerPanelPassword, cfg.ApiUrl)
	plansService := plans.NewService(plans.NewStore(mongodb))
	paymentsService := payments.NewService(payments.NewStore(mongodb), plansService,
		payments.CloudPaymentsConfig{
			PublicID:  cfg.CloudPayments.PublicID,
			SecretKey: cfg.CloudPayments.SecretKey,
			ApiUrl:    cfg.CloudPayments.ApiUrl,
		})
	subscriptionsService := subscriptions.NewService(subscriptions.NewStore(mongodb), plansService, paymentsService)

	scheduler := scheduler.NewScheduler(subscriptionsService, peersService, serversService, notifier, logger)
	analytics := analytics.NewAnalytics(b, usersService, subscriptionsService, paymentsService, logger, int64(cfg.AnalyticsChanelID))

	stackStore := bot.NewStackScreens(mongodb)
	bot := bot.NewBot(b, *stackStore, logger, usersService, serversService, peersService, plansService, subscriptionsService, cfg.ApiUrl)

	handler := api.NewHandler(peersService, plansService, paymentsService, subscriptionsService, cfg.CloudPayments.SecretKey, bot, logger, cfg.ApiUrl, cfg.Apps)

	srv := server.NewServer(&server.HttpConfig{
		Port: cfg.Port,
	}, handler.RegisterRoutes())

	go runAnalyticsUpdater(ctx, analytics)
	go runScheduler(ctx, scheduler)
	go srv.Run()
	go bot.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	ctx, shutdown := context.WithTimeout(context.Background(), timeout)
	defer shutdown()

	if err := srv.Stop(ctx); err != nil {
		log.Printf("error stopping server: %v", err)
	}

	if err := mongodbClient.Disconnect(ctx); err != nil {
		log.Printf("error disconnect to mongodbClient. err: %v", err)
	}
}

func runScheduler(ctx context.Context, scheduler *scheduler.Scheduler) {
	ticker := time.NewTicker(3 * time.Hour)
	for {
		select {
		case <-ticker.C:
			// scheduler.CheckExpiredSubscriptions(ctx)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func runAnalyticsUpdater(ctx context.Context, analytics *analytics.Analytics) {
	ticker := time.NewTicker(30 * time.Minute)
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Recovered in analytics update: %v\n", r)
					}
				}()

				analytics.UpdateAnalyticsData(ctx)

			}()
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
