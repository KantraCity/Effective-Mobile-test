package main

import (
	"Testwork/internal/config"
	"Testwork/internal/handler"
	"Testwork/internal/repository"
	"Testwork/internal/service"
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "Testwork/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Subscription Service API
// @version 1.0
// @description API для управления подписками пользователей
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
func main() {
	cfg := config.Load()

	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("app", "subscription-service").
		Logger()

	logger.Info().
		Str("port", cfg.Port).
		Str("log_level", cfg.LogLevel).
		Int("max_idle_conns", cfg.MaxIdleConns).
		Msg("configuration loaded")

	if cfg.DBURL == "" {
		logger.Fatal().Msg("DB_URL is not set")
	}

	logger.Info().Str("version", "1.0.0").Msg("application starting")

	logger.Info().Str("db_url", maskDBURL(cfg.DBURL)).Msg("connecting to database")

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to open database connection")
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close database connection")
		}
	}()

	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to ping database")
	}
	logger.Info().Msg("database connection established")

	runMigrations(cfg.DBURL, logger)

	repo := repository.NewSubscriptionRepository(db, logger)
	logger.Debug().Msg("repository layer initialized")

	svc := service.NewSubscriptionService(repo, logger)
	logger.Debug().Msg("service layer initialized")

	h := handler.NewHandler(svc, logger)
	logger.Debug().Msg("handler layer initialized")

	gin.SetMode(ginMode(cfg.LogLevel))
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLoggerMiddleware(logger))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	api := r.Group("/api/v1")
	{
		subs := api.Group("/subscriptions")
		{
			subs.POST("/", h.CreateSubscription)
			subs.GET("/", h.ListSubscriptions)
			subs.GET("/total", h.GetTotalCost)
			subs.GET("/:id", h.GetSubscription)
			subs.PUT("/:id", h.UpdateSubscription)
			subs.DELETE("/:id", h.DeleteSubscription)
		}
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		sig := <-sigint

		logger.Warn().Str("signal", sig.String()).Msg("shutdown signal received, starting graceful shutdown")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("error during graceful shutdown")
		}
		close(idleConnsClosed)
	}()

	logger.Info().Str("port", cfg.Port).Msg("server starting")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal().Err(err).Msg("server failed to start")
	}

	<-idleConnsClosed
	logger.Info().Msg("server stopped gracefully")
}

func runMigrations(dbURL string, logger zerolog.Logger) {
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init migrations")
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			logger.Error().Err(srcErr).Err(dbErr).Msg("failed to close migrator")
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}

	logger.Info().Msg("migrations applied successfully")
}

func ginMode(logLevel string) string {
	if logLevel == "debug" {
		return gin.DebugMode
	}
	return gin.ReleaseMode
}

func maskDBURL(dsn string) string {
	if dsn == "" {
		return ""
	}
	const sep = "://"
	sepIdx := len(sep)
	start := -1
	end := -1
	for i := 0; i < len(dsn)-len(sep); i++ {
		if dsn[i:i+len(sep)] == sep {
			start = i
			sepIdx = i + len(sep)
			break
		}
	}
	if start != -1 {
		for j := sepIdx; j < len(dsn); j++ {
			if dsn[j] == '@' {
				end = j
				break
			}
		}
	}
	if start != -1 && end != -1 {
		return dsn[:sepIdx] + "***" + dsn[end:]
	}
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-10:]
	}
	return "***"
}

func requestLoggerMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		if path == "/health" {
			logger.Debug().Str("method", method).Str("path", path).Msg("health check")
			c.Next()
			return
		}

		logger.Debug().
			Str("method", method).
			Str("path", path).
			Str("ip", c.ClientIP()).
			Msg("request started")

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		event := logger.Info()
		if status >= 400 && status < 500 {
			event = logger.Warn()
		} else if status >= 500 {
			event = logger.Error()
		}

		event.
			Int("status", status).
			Str("method", method).
			Str("path", path).
			Dur("latency_ms", latency).
			Int("body_size", c.Writer.Size()).
			Msg("request completed")
	}
}
