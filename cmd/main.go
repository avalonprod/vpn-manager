package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vpn-manager/api"
	tgbot "vpn-manager/bot"
	"vpn-manager/core/config"
	"vpn-manager/notifier"
	"vpn-manager/peers"
	"vpn-manager/pkg/db/mongodb"
	"vpn-manager/pkg/logger"
	"vpn-manager/pkg/server"
	"vpn-manager/scheduler"
	"vpn-manager/servers"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"github.com/go-telegram/bot"
)

const timeout = 5 * time.Second

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []bot.Option{
		bot.WithWebhookSecretToken(cfg.TelegramWebhookToken),
	}

	bot, err := bot.New(cfg.TelegramAccessToken, opts...)
	if err != nil {
		panic(err)
	}

	mongodbClient, err := mongodb.NewConnection(cfg.MongoDB.URI, cfg.MongoDB.Username, cfg.MongoDB.Password)
	if err != nil {
		panic(err)
	}

	mongodb := mongodbClient.Database(cfg.MongoDB.Name)
	logger := logger.NewLogger()

	notifier := notifier.NewNotifier(bot)
	usersService := users.NewService(users.NewStore(mongodb))
	peersService := peers.NewService(peers.NewStore(mongodb))
	serversService := servers.NewService(servers.NewStore(mongodb), peersService, cfg.ApiUrl)
	subscriptionsService := subscriptions.NewService(subscriptions.NewStore(mongodb))
	scheduler := scheduler.NewScheduler(subscriptionsService, peersService, serversService, notifier, logger)

	handler := api.NewHandler(peersService, cfg.ApiUrl)

	tgBot := tgbot.NewTGBot(bot, logger, usersService, serversService, subscriptionsService)

	srv := server.NewServer(&server.HttpConfig{
		Port: cfg.Port,
	}, handler.RegisterRoutes())

	go runScheduler(ctx, scheduler)
	go srv.Run()
	go tgBot.Run(ctx, cfg.TelegramWebhookPort)

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
			scheduler.CheckExpiredSubscriptions(ctx)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
