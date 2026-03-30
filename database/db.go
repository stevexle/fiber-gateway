package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	cfg := config.AppConfig
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s search_path=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode, cfg.DBSchema)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true, // Optimize repeated queries
		SkipDefaultTransaction: true, // Faster writes
	})

	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return err
	}

	// Connection Pool Optimization
	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)                  // Keep 10 idle connections
	sqlDB.SetMaxOpenConns(100)                 // Capacity for 100 concurrent connections
	sqlDB.SetConnMaxLifetime(time.Hour)        // Refresh connections every hour

	slog.Info("connected to database", "pool_size", 100)

	// Auto Migration
	return DB.AutoMigrate(&models.User{}, &models.Client{}, &models.AuthorizeCode{}, &models.RefreshToken{})
}

func Close() {
	if DB != nil {
		if sqlDB, err := DB.DB(); err == nil {
			slog.Info("closing database connection")
			sqlDB.Close()
		}
	}
}
