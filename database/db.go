package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/fiber-gateway/config"
	"log/slog"
)

var DB *gorm.DB

func Connect() error {
	var err error
	cfg := config.AppConfig

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Ho_Chi_Minh search_path=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
		cfg.DBSchema,
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return err
	}

	// Optimize connection pool for high-concurrency and parallel prefork processing
	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	slog.Info("connected to database")
	return nil
}

// Close gracefully closes the database connection.
func Close() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			_ = sqlDB.Close()
			slog.Info("database connection closed")
		}
	}
}
