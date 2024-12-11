// The wsignd command implements the Wrale Signage server
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"

	"github.com/wrale/wrale-signage/internal/wsignd/config"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
	contenthttp "github.com/wrale/wrale-signage/internal/wsignd/content/http"
	contentpg "github.com/wrale/wrale-signage/internal/wsignd/content/postgres"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	activationpg "github.com/wrale/wrale-signage/internal/wsignd/display/activation/postgres"
	displayhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http"
	displaypg "github.com/wrale/wrale-signage/internal/wsignd/display/postgres"
	"github.com/wrale/wrale-signage/internal/wsignd/display/service"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadFile(*configPath)
		if err != nil {
			logger.Error("failed to load config file", "error", err)
			os.Exit(1)
		}
	} else {
		cfg, err = config.Load()
		if err != nil {
			logger.Error("failed to load configuration", "error", err)
			os.Exit(1)
		}
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Setup database and run migrations
	db, err := database.SetupDatabase(connStr, 5, time.Second)
	if err != nil {
		logger.Error("failed to setup database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      setupRouter(cfg, db, logger),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	go func() {
		logger.Info("starting server",
			"host", cfg.Server.Host,
			"port", cfg.Server.Port,
		)

		if cfg.Server.TLSCert != "" && cfg.Server.TLSKey != "" {
			err = server.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Handle shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

func setupRouter(cfg *config.Config, db *sql.DB, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Set up display service
	displayRepo := displaypg.NewRepository(db, logger)
	displayPublisher := service.NewNoopEventPublisher()
	displayService := service.New(displayRepo, displayPublisher, logger)

	// Set up activation service
	activationRepo := activationpg.NewRepository(db, logger)
	activationService := activation.NewService(activationRepo)

	// Create display handler
	displayHandler := displayhttp.NewHandler(displayService, activationService, logger)
	r.Mount("/", displayHandler.Router())

	// Set up content service
	contentRepo := contentpg.NewRepository(db)
	contentProcessor := &noopEventProcessor{}
	contentMetrics := &noopMetricsAggregator{}
	contentMonitor := &noopHealthMonitor{}
	contentService := content.NewService(contentRepo, contentProcessor, contentMetrics, contentMonitor)

	// Create content handler
	contentHandler := contenthttp.NewHandler(contentService, logger.With().Str("component", "content").Logger())
	r.Mount("/api/v1alpha1/content", contentHandler.Router())

	return r
}

// Content no-op implementations
type noopEventProcessor struct{}

func (p *noopEventProcessor) ProcessEvents(ctx context.Context, batch content.EventBatch) error {
	return nil
}

type noopMetricsAggregator struct{}

func (m *noopMetricsAggregator) RecordMetrics(ctx context.Context, event content.Event) error {
	return nil
}
func (m *noopMetricsAggregator) GetURLMetrics(ctx context.Context, url string) (*content.URLMetrics, error) {
	return &content.URLMetrics{URL: url}, nil
}

type noopHealthMonitor struct{}

func (h *noopHealthMonitor) CheckHealth(ctx context.Context, url string) (*content.HealthStatus, error) {
	return &content.HealthStatus{URL: url, Healthy: true}, nil
}
func (h *noopHealthMonitor) GetHealthHistory(ctx context.Context, url string) ([]content.HealthStatus, error) {
	return []content.HealthStatus{}, nil
}
