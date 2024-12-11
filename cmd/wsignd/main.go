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
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	displayhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http"
	displaypg "github.com/wrale/wrale-signage/internal/wsignd/display/postgres"
	"github.com/wrale/wrale-signage/internal/wsignd/display/service"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Initialize structured logging with JSON format for easier parsing
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

	// Establish database connection with proper connection pooling and run migrations
	db, err := database.SetupDatabase(connStr, 5, time.Second)
	if err != nil {
		logger.Error("failed to setup database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create HTTP server with timeouts and configuration
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      setupRouter(cfg, db, logger),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start the server in a goroutine to allow for graceful shutdown
	go func() {
		logger.Info("starting server",
			"host", cfg.Server.Host,
			"port", cfg.Server.Port,
		)

		var err error
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

	// Set up graceful shutdown on interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-shutdown
	logger.Info("shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

// setupRouter creates and configures the HTTP router with all application routes
func setupRouter(cfg *config.Config, db *sql.DB, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Set up display service dependencies
	displayRepo := displaypg.NewRepository(db, logger)
	displayPublisher := &noopEventPublisher{} // TODO: Implement real event publisher
	displayService := service.New(displayRepo, displayPublisher, logger)

	// Set up activation service
	activationRepo := activation.NewRepository(db, logger)
	activationService := activation.NewService(activationRepo)

	// Create and mount display handlers
	displayHandler := displayhttp.NewHandler(displayService, activationService, logger)
	r.Mount("/api/v1alpha1/displays", displayHandler.Router())

	// Set up content service dependencies
	contentRepo := contentpg.NewRepository(db)
	contentProcessor := &noopEventProcessor{}  // TODO: Implement real event processor
	contentMetrics := &noopMetricsAggregator{} // TODO: Implement real metrics
	contentMonitor := &noopHealthMonitor{}     // TODO: Implement real monitor
	contentService := content.NewService(contentRepo, contentProcessor, contentMetrics, contentMonitor)

	// Create and mount content handlers
	contentHandler := contenthttp.NewHandler(contentService)
	r.Mount("/api/v1alpha1/content", contentHandler.Router())

	return r
}

// noopEventPublisher is a temporary implementation of display.EventPublisher
type noopEventPublisher struct{}

func (p *noopEventPublisher) Publish(ctx context.Context, event display.Event) error {
	return nil
}

// Temporary no-op implementations for content infrastructure
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
