package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vpn-manager/api"
	"vpn-manager/bot"
	"vpn-manager/core/config"
	"vpn-manager/jobs"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/pkg/db/mongodb"
	"vpn-manager/pkg/logger"
	"vpn-manager/pkg/server"
	"vpn-manager/plans"
	"vpn-manager/servers"
	"vpn-manager/subscriptions"
	"vpn-manager/tasks"
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
	tasksService := tasks.NewService(tasks.NewStore(mongodb))
	stackStore := bot.NewStackScreens(mongodb)
	bot := bot.NewBot(b, *stackStore, logger, usersService, serversService, peersService, plansService, subscriptionsService, tasksService, cfg.ApiUrl)

	handler := api.NewHandler(peersService, plansService, paymentsService, subscriptionsService, tasksService, cfg.CloudPayments.SecretKey, bot, logger, cfg.ApiUrl, cfg.Apps)

	srv := server.NewServer(&server.HttpConfig{
		Port: cfg.Port,
	}, handler.RegisterRoutes())

	// Tasks ===============================
	var trialNudgeHandler = func(ctx context.Context, t tasks.Task) error {
		sub, err := subscriptionsService.GetByUserID(ctx, t.UserID)
		if err == nil && sub.ID != "" {
			return tasksService.MarkDone(ctx, t.ID)
		}

		if err := bot.SendTrialNudge(t.UserID); err != nil {
			return err
		}

		err = tasksService.Reschedule(ctx, t.ID, time.Now().UTC().Add(24*time.Hour))

		return err
	}

	var setupNudgeHandler = func(ctx context.Context, t tasks.Task) error {
		peer, err := peersService.GetPeerByUserID(ctx, t.UserID)
		if err != nil {
			return err
		}

		if peer.IsImported {
			return tasksService.MarkDone(ctx, t.ID)
		}

		if err := bot.SendSetupNudge(t.UserID, string(t.Payload)); err != nil {
			return err
		}

		err = tasksService.Reschedule(ctx, t.ID, time.Now().UTC().Add(24*time.Hour))

		return err
	}
	runner := tasks.NewRunner(tasksService, map[string]tasks.Handler{
		"trial_nudge": trialNudgeHandler,
		"setup_nudge": setupNudgeHandler,
	})

	go runner.Run(ctx, 1)
	// ===============================

	go jobs.RunDisableExpiredAccess(ctx, subscriptionsService, peersService, serversService, bot, logger)
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
