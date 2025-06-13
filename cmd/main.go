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

func main() {
	cfg := config.MustLoad()

	bot, err := bot.New(cfg.TelegramAccessToken)
	if err != nil {
		panic(err)
	}

	mongodbClient, err := mongodb.NewConnection(cfg.MongoDB.URI, cfg.MongoDB.Username, cfg.MongoDB.Password)
	if err != nil {
		panic(err)
	}

	notifier := notifier.NewNotifier(bot)

	logger := logger.NewLogger()

	mongodb := mongodbClient.Database(cfg.MongoDB.Name)

	usersStore := users.NewStore(mongodb)
	usersService := users.NewService(usersStore)

	peersStore := peers.NewStore(mongodb)
	peersService := peers.NewService(peersStore)

	serversStore := servers.NewStore(mongodb)
	serversService := servers.NewService(serversStore, peersService, cfg.ApiUrl)

	subscriptionsStore := subscriptions.NewStore(mongodb)
	subscriptionsService := subscriptions.NewService(subscriptionsStore)

	scheduler := scheduler.NewScheduler(subscriptionsService, peersService, serversService, notifier, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(3 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				scheduler.CheckExpiredSubscriptions(context.Background())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	handler := api.NewHandler(peersService, cfg.ApiUrl)

	tgBot := tgbot.NewTGBot(bot, logger, usersService, serversService, subscriptionsService)

	srv := server.NewServer(&server.HttpConfig{
		Port: cfg.Port,
	}, handler.RegisterRoutes())

	go func() {
		srv.Run()
	}()

	go func() {
		tgBot.Run(context.Background())
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	const timeout = 5 * time.Second

	ctx, shutdown := context.WithTimeout(context.Background(), timeout)
	defer shutdown()

	if err := srv.Stop(ctx); err != nil {
		log.Printf("error stopping server: %v", err)
	}

	if err := mongodbClient.Disconnect(ctx); err != nil {
		log.Printf("error disconnect to mongodbClient. err: %v", err)
	}
}
