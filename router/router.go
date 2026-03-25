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
)

// SetupRoutes registers all application routes.
func SetupRoutes(app *fiber.App) {
	// 1. Global Middleware
	registerGlobalMiddleware(app)

	api := app.Group("/api/v1")

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
}

func registerInternalAuthRoutes(api fiber.Router) {
	authGroup := api.Group("/auth")
	authGroup.Get("/login", handler.Login)
	authGroup.Get("/refresh", handler.Refresh)
	authGroup.Get("/logout", handler.Logout)
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

	// 4. Final Proxy Handler
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
