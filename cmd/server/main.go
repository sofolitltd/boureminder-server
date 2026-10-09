// Command server runs the বউ রিমাইন্ডার API: HTTP, WebSocket, and River
// workers in a single binary (docs/BACKEND_GO.md §2), on top of Gin.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	appauth "github.com/sofolit/boureminder/api/internal/auth"
	"github.com/sofolit/boureminder/api/internal/config"
	"github.com/sofolit/boureminder/api/internal/features/me"
	"github.com/sofolit/boureminder/api/internal/features/pairing"
	"github.com/sofolit/boureminder/api/internal/httpx"
	"github.com/sofolit/boureminder/api/internal/jobs"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Pooled connection for ordinary request handling; PgBouncer-friendly.
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	riverClient, err := jobs.NewClient(pool)
	if err != nil {
		return fmt.Errorf("build river client: %w", err)
	}
	if err := riverClient.Start(ctx); err != nil {
		return fmt.Errorf("start river client: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = riverClient.Stop(stopCtx)
	}()

	fb, err := appauth.NewFirebase(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init firebase: %w", err)
	}

	meSvc := me.NewService(pool)
	meHandler := me.NewHandler(meSvc)
	pairingSvc := pairing.NewService(pool, riverClient)
	pairingHandler := pairing.NewHandler(pairingSvc)

	if cfg.Env == "prod" || cfg.Env == "staging" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(
		httpx.Recover(logger),
		httpx.Logging(logger),
		httpx.RequestIDMiddleware(),
		httpx.VersionGate(cfg.MinAppVersion),
	)

	router.GET("/healthz", healthzHandler(pool))

	v1 := router.Group("/v1", appauth.VerifyToken(fb))
	v1.POST("/me/bootstrap", meHandler.Bootstrap)

	authed := v1.Group("", appauth.RequireUser(meSvc))
	authed.GET("/me", meHandler.Get)
	authed.PATCH("/me", meHandler.Patch)
	authed.POST("/devices", meHandler.RegisterDevice)
	authed.DELETE("/devices/:token", meHandler.UnregisterDevice)
	authed.POST("/pair/codes", pairingHandler.CreateCode)
	authed.POST("/pair/join", pairingHandler.Join)
	authed.PUT("/couple/wedding-date", pairingHandler.SetWeddingDate)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func healthzHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			httpx.WriteErr(c, httpx.ErrCode(http.StatusServiceUnavailable, "db_unreachable"))
			return
		}
		httpx.WriteJSON(c, http.StatusOK, gin.H{"status": "ok"})
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
