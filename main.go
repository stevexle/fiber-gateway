package main

import (
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/database"
	"github.com/fiber-gateway/pkg/logger"
	"github.com/fiber-gateway/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// ── Load Config ───────────────────────────────────────────────────────────
	cfg := config.Load()

	// ── Logger ────────────────────────────────────────────────────────────────
	logLevel := logger.ParseLevel(cfg.LogLevel)

	// Initialize log file matched to Logback equivalent rolling policy config
	logFile := logger.NewRollingFile(logger.RollingConfig{
		Filename:   cfg.LogFilename,
		MaxSizeMB:  300,
		MaxAgeDays: 60,
		MaxBackups: 68, // 20GB max total cap (20480 MB / 300 = 68 backups)
	})

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger.GlobalWriter = multiWriter
	log := logger.NewWithWriter(cfg.ServiceName, logLevel, multiWriter)
	slog.SetDefault(log)

	log.Info("starting", slog.String("service", cfg.ServiceName), slog.String("level", logLevel.String()))

	// ── Fiber app ─────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:               cfg.ServiceName,
		DisableStartupMessage: true,
		// Multi-core performance: spawn child processes
		Prefork:               cfg.Environment == "production", 
		// Performance optimizations
		CaseSensitive:         true,
		StrictRouting:         true,
		ServerHeader:          "Fiber Gateway",
		// Timeouts
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		// Resource protection
		BodyLimit: cfg.BodyLimit * 1024 * 1024,
		ProxyHeader: "X-Forwarded-For",
		// Buffer optimizations for high throughput
		ReadBufferSize:  8 * 1024,
		WriteBufferSize: 8 * 1024,
		Concurrency:     256 * 1024,
	})

	// Middlewares
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// ── Start ─────────────────────────────────────────────────────────────────
	log.Info("listening", slog.String("port", cfg.Port))

	err := database.Connect()
	if err != nil {
		log.Error("Failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ── Routes ────────────────────────────────────────────────────────────────
	router.SetupRoutes(app)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("gracefully shutting down server...")
		database.Close()
		_ = app.ShutdownWithTimeout(5 * time.Second)
	}()

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Error("server exited", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
