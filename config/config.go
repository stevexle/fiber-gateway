package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type RouteConfig struct {
	Path                string   `json:"path"`
	Method              string   `json:"method"` // ALL, GET, POST, etc
	Roles               []string `json:"roles"`
	Targets             []string `json:"targets"` // For load balancing
	Strategy            string   `json:"strategy"`
	RateLimitMax        int      `json:"rate_limit_max"`
	RateLimitExpiration string   `json:"rate_limit_expiration"` // e.g. "1m", "5s"
	Protected           bool     `json:"protected"`
	Cache               bool     `json:"cache"`       // Toggle caching
	CacheExpiration     string   `json:"cache_exp"`   // Cache TTL e.g. "5m"
	Compress            bool     `json:"compress"`    // Toggle compression
	CircuitBreaker      bool     `json:"circuit_breaker"`
	CBMaxFailures       int      `json:"cb_max_failures"`
	CBExpirationSeconds int      `json:"cb_exp_seconds"`
}

type LoggingConfig struct {
	SkipPaths []string `json:"skip_paths"`
}

type RoutesJSON struct {
	Logging LoggingConfig `json:"logging"`
	Proxy   []RouteConfig `json:"proxy"`
}

type Config struct {
	Environment      string
	ServiceName      string
	Port             string
	LogLevel         string
	LogFilename      string
	CORSAllowOrigins string

	RateLimitGlobalMax        int
	RateLimitGlobalExpiration string

	GzipEnabled bool // Global gzip
	BodyLimit   int  // Max body limit in MB

	// Global Circuit Breaker defaults
	CBMaxFailures       int
	CBExpirationSeconds int

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBSchema   string

	JWTSecret             []byte
	JWTAccessExpMinutes   time.Duration
	JWTRefreshExpDays     time.Duration
	JWTSsoSessionDays     time.Duration
	JWTAuthCodeExpMinutes time.Duration

	Logging LoggingConfig
	Proxy   []RouteConfig
}

var (
	AppConfig *Config
	once      sync.Once
)

func getEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return fallback
}

func Load() *Config {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			slog.Warn("no .env file found (using system environment variables)")
		}

		globalMax, _ := strconv.Atoi(getEnv("RATE_LIMIT_GLOBAL_MAX", "0"))
		bodyLimit, _ := strconv.Atoi(getEnv("BODY_LIMIT_MB", "4"))
		AppConfig = &Config{
			ServiceName:               getEnv("SERVICE_NAME", "fiber-gateway"),
			Port:                      getEnv("PORT", "8080"),
			LogLevel:                  getEnv("LOG_LEVEL", "debug"),
			LogFilename:               getEnv("LOG_FILENAME", "logs/fiber-gateway.log"),
			CORSAllowOrigins:          getEnv("CORS_ALLOW_ORIGINS", "*"),
			RateLimitGlobalMax:        globalMax,
			RateLimitGlobalExpiration: getEnv("RATE_LIMIT_GLOBAL_EXPIRATION", "1m"),
			GzipEnabled:               getEnv("GZIP_ENABLED", "true") == "true",
			BodyLimit:                 bodyLimit,
			CBMaxFailures:             5,
			CBExpirationSeconds:       30,
			DBHost:                    getEnv("DB_HOST", "localhost"),
			DBPort:           getEnv("DB_PORT", "5432"),
			DBUser:           getEnv("DB_USER", "postgres"),
			DBPassword:       getEnv("DB_PASSWORD", "postgres"),
			DBName:           getEnv("DB_NAME", "postgres"),
			DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
			DBSchema:         getEnv("DB_SCHEMA", "public"),
			JWTSecret:        []byte(os.Getenv("JWT_SECRET")),
		}

		// Load routes from JSON (like nginx config)
		routesFile := getEnv("ROUTES_CONFIG", "routes.json")
		if data, err := os.ReadFile(routesFile); err == nil {
			var rwrap RoutesJSON
			if err := json.Unmarshal(data, &rwrap); err == nil {
				AppConfig.Logging = rwrap.Logging
				AppConfig.Proxy = rwrap.Proxy
			} else {
				slog.Error("Failed to parse routes.json", "error", err)
			}
		}

		if len(AppConfig.JWTSecret) == 0 {
			slog.Warn("JWT_SECRET is not set! Using a default insecure key for development.")
			AppConfig.JWTSecret = []byte("my-insecure-development-secret-change-me")
		}

		AppConfig.JWTAccessExpMinutes = 15 * time.Minute
		if val := os.Getenv("JWT_ACCESS_EXPIRATION_MINUTES"); val != "" {
			if min, err := strconv.Atoi(val); err == nil && min > 0 {
				AppConfig.JWTAccessExpMinutes = time.Duration(min) * time.Minute
			}
		}

		AppConfig.JWTRefreshExpDays = 7 * 24 * time.Hour
		if val := os.Getenv("JWT_REFRESH_EXPIRATION_DAYS"); val != "" {
			if days, err := strconv.Atoi(val); err == nil && days > 0 {
				AppConfig.JWTRefreshExpDays = time.Duration(days) * 24 * time.Hour
			}
		}

		AppConfig.Environment = "development"
		if val := os.Getenv("ENV"); val != "" {
			AppConfig.Environment = val
		}

		AppConfig.JWTSsoSessionDays = 30 * 24 * time.Hour
		if val := os.Getenv("JWT_SSO_SESSION_DAYS"); val != "" {
			if days, err := strconv.Atoi(val); err == nil && days > 0 {
				AppConfig.JWTSsoSessionDays = time.Duration(days) * 24 * time.Hour
			}
		}

		AppConfig.JWTAuthCodeExpMinutes = 5 * time.Minute
		if val := os.Getenv("JWT_AUTH_CODE_EXP_MINUTES"); val != "" {
			if min, err := strconv.Atoi(val); err == nil && min > 0 {
				AppConfig.JWTAuthCodeExpMinutes = time.Duration(min) * time.Minute
			}
		}
	})
	return AppConfig
}
