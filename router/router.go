package router

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/handler"
	"github.com/fiber-gateway/middleware"
	"github.com/fiber-gateway/models"
	"github.com/fiber-gateway/pkg/balancer"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/contrib/circuitbreaker"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

// SetupRoutes registers all application routes.
func SetupRoutes(app *fiber.App) {
	// 1. Global Middleware
	registerGlobalMiddleware(app)

	api := app.Group("/api/v1")
	// Apply Smart Dynamic CORS to all API routes
	api.Use(middleware.DynamicCORS())

	// 2. Health Check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// 3. Internal Auth Routes
	registerInternalAuthRoutes(api)

	// 4. Dynamic Proxy Routes
	registerDynamicProxyRoutes(api, config.AppConfig.Proxy)
}

func registerGlobalMiddleware(app *fiber.App) {
	app.Use(middleware.HTTPLogger())

	cfg := config.AppConfig
	if cfg.RateLimitGlobalMax > 0 {
		expiration := 1 * time.Minute
		if cfg.RateLimitGlobalExpiration != "" {
			if d, err := time.ParseDuration(cfg.RateLimitGlobalExpiration); err == nil {
				expiration = d
			}
		}
		app.Use(middleware.RateLimiter(cfg.RateLimitGlobalMax, expiration))
		slog.Info("Global rate limiter enabled", "max", cfg.RateLimitGlobalMax, "expiration", expiration)
	}
	// Enable compression if configured
	if cfg.GzipEnabled {
		app.Use(compress.New(compress.Config{
			Level: compress.LevelBestSpeed, // Nginx default-ish
		}))
		slog.Info("Global Gzip compression enabled")
	}
}

func registerInternalAuthRoutes(api fiber.Router) {
	authGroup := api.Group("/auth")
	authGroup.Post("/register", handler.RegisterUser)
	authGroup.Post("/login", handler.Login)
	authGroup.Post("/authorize", handler.Authorize)
	authGroup.Post("/token", handler.ExchangeToken)
	authGroup.Post("/refresh", handler.Refresh)
	authGroup.Post("/logout", handler.Logout)

	clientGroup := api.Group("/client")
	clientGroup.Post("/register", handler.RegisterClient)
}

func registerDynamicProxyRoutes(api fiber.Router, routes []config.RouteConfig) {
	for _, rc := range routes {
		lb := createBalancer(rc)
		handlers := buildHandlersChain(rc, lb)

		method := strings.ToUpper(rc.Method)
		slog.Info("Registering dynamic route",
			"method", method,
			"path", rc.Path,
			"targets", len(rc.Targets),
			"strategy", rc.Strategy,
		)

		switch method {
		case "GET":
			api.Get(rc.Path, handlers...)
		case "POST":
			api.Post(rc.Path, handlers...)
		case "PUT":
			api.Put(rc.Path, handlers...)
		case "DELETE":
			api.Delete(rc.Path, handlers...)
		case "PATCH":
			api.Patch(rc.Path, handlers...)
		default:
			api.All(rc.Path, handlers...)
		}
	}
}

func buildHandlersChain(rc config.RouteConfig, lb balancer.Balancer) []fiber.Handler {
	var handlers []fiber.Handler

	// 1. Authentication
	if rc.Protected {
		handlers = append(handlers, middleware.JWTProtected())
	}

	// 2. Per-Route Rate Limiting
	if rc.RateLimitMax > 0 {
		expiration := 1 * time.Minute
		if rc.RateLimitExpiration != "" {
			if d, err := time.ParseDuration(rc.RateLimitExpiration); err == nil {
				expiration = d
			}
		}
		handlers = append(handlers, middleware.RateLimiter(rc.RateLimitMax, expiration))
	}

	// 3. Role Authorization
	if len(rc.Roles) > 0 {
		var modelsRoles []models.Role
		for _, r := range rc.Roles {
			modelsRoles = append(modelsRoles, models.Role(r))
		}
		handlers = append(handlers, middleware.RequireRole(modelsRoles...))
	}

	// 4. Per-route Cache
	if rc.Cache {
		exp := 5 * time.Minute
		if rc.CacheExpiration != "" {
			if d, err := time.ParseDuration(rc.CacheExpiration); err == nil {
				exp = d
			}
		}
		handlers = append(handlers, cache.New(cache.Config{
			Expiration: exp,
			CacheHeader: "X-Cache",
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.Method() + "|" + c.Path()
			},
			Next: func(c *fiber.Ctx) bool {
				// Only cache GET and HEAD requests (Nginx default)
				return c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead
			},
		}))
		slog.Info("Cache enabled for route", "path", rc.Path, "exp", exp)
	}

	// 5. Per-route Compression (if not global)
	if rc.Compress && !config.AppConfig.GzipEnabled {
		handlers = append(handlers, compress.New())
	}

	// 6. Circuit Breaker
	if rc.CircuitBreaker {
		maxFail := config.AppConfig.CBMaxFailures
		if rc.CBMaxFailures > 0 {
			maxFail = rc.CBMaxFailures
		}
		exp := time.Duration(config.AppConfig.CBExpirationSeconds) * time.Second
		if rc.CBExpirationSeconds > 0 {
			exp = time.Duration(rc.CBExpirationSeconds) * time.Second
		}
		cb := circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: maxFail,
			Timeout:          exp,
		})
		handlers = append(handlers, circuitbreaker.Middleware(cb))
		slog.Info("Circuit breaker enabled for route", "path", rc.Path, "threshold", maxFail, "expiration", exp)
	}

	// 7. Final Proxy Handler
	handlers = append(handlers, middleware.ReverseProxy(lb))

	return handlers
}

func createBalancer(rc config.RouteConfig) balancer.Balancer {
	targets := rc.Targets
	switch strings.ToLower(rc.Strategy) {
	case "random":
		return balancer.NewRandom(targets)
	case "least_conn":
		return balancer.NewLeastConn(targets)
	default:
		return balancer.NewRoundRobin(targets)
	}
}
