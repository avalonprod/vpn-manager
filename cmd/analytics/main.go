package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vpn-manager/analytics"
	"vpn-manager/analytics/sheets"
	"vpn-manager/core/config"
	"vpn-manager/payments"
	"vpn-manager/pkg/db/mongodb"
	"vpn-manager/pkg/logger"
	"vpn-manager/subscriptions"
	"vpn-manager/users"
)

const timeout = 5 * time.Second

const (
	usersSheetName     = "Users"
	analyticsSheetName = "Analytics"
)

func main() {
	cfg := config.MustLoad()
	logger := logger.NewLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongodbClient, err := mongodb.NewConnection(cfg.MongoDB.URI, cfg.MongoDB.Username, cfg.MongoDB.Password)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	mongodb := mongodbClient.Database(cfg.MongoDB.Name)

	exporterWriter, err := sheets.NewWriter(ctx, cfg.AnalyticsSpreadsheetID, usersSheetName)
	if err != nil {
		log.Fatalf("failed to init Sheets writer: %v", err)
	}
	analyticsWriter, err := sheets.NewWriter(ctx, cfg.AnalyticsSpreadsheetID, analyticsSheetName)
	if err != nil {
		log.Fatalf("failed to init Sheets writer: %v", err)
	}

	usersStore := users.NewStore(mongodb)
	subscriptionsStore := subscriptions.NewStore(mongodb)
	paymentsStore := payments.NewStore(mongodb)

	exporter := analytics.NewExporter(exporterWriter, usersStore, subscriptionsStore, paymentsStore)
	analytics := analytics.NewAnalytics(analyticsWriter, usersStore, subscriptionsStore, paymentsStore, logger)

	go runDataExporter(ctx, exporter)
	go runAnalyticsUpdater(ctx, analytics)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	shutdownCtx, shutdown := context.WithTimeout(context.Background(), timeout)
	defer shutdown()

	if err := mongodbClient.Disconnect(shutdownCtx); err != nil {
		log.Printf("error disconnect to mongodbClient. err: %v", err)
	}
}

func runDataExporter(ctx context.Context, exporter *analytics.Exporter) {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("recovered in data export: %v\n", r)
					}
				}()

				if err := exporter.ExportOverview(ctx); err != nil {
					log.Printf("failed to export overview: %v\n", err)
				} else {
					log.Println("overview exported successfully")
				}

			}()
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func runAnalyticsUpdater(ctx context.Context, analytics *analytics.Analytics) {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("recovered in analytics update: %v\n", r)
					}
				}()

				if err := analytics.UpdateAnalyticsData(ctx); err != nil {
					log.Printf("failed to update analytics data: %v\n", err)
				} else {
					log.Println("analytics data updated successfully")
				}
			}()
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
