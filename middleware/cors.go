package middleware

import (
	"log/slog"
	"strings"
	"sync"
	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/repository"
	"github.com/fiber-gateway/utils"
	"github.com/gofiber/fiber/v2"
)

// Internal cache to store web_origins (avoiding DB hits on every request)
// Key: clientID, Value: []string (allowedOrigins)
var originCache sync.Map

// DynamicCORS validates the Origin header against registered web_origins.
func DynamicCORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")
		if origin == "" {
			return c.Next()
		}

		// helper to set standard headers
		setHeaders := func(ctx *fiber.Ctx, org string) {
			ctx.Set("Access-Control-Allow-Origin", org)
			ctx.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			ctx.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			ctx.Set("Access-Control-Allow-Credentials", "true")
			ctx.Set("Access-Control-Max-Age", "3600")
		}

		// 1. Check Global Fallback FIRST (from config .env)
		globalAllowed := strings.Split(config.AppConfig.CORSAllowOrigins, ",")
		for _, g := range globalAllowed {
			trimmed := strings.TrimSpace(g)
			if trimmed == origin || trimmed == "*" {
				// Mirror the specific origin to satisfy the 'Allow-Credentials' requirement
				setHeaders(c, origin)
				if c.Method() == "OPTIONS" {
					return c.SendStatus(204)
				}
				return c.Next()
			}
		}
		
		// Log if we didn't match the fallback
		slog.Debug("CORS fallback check missed", "origin", origin, "config", config.AppConfig.CORSAllowOrigins)

		var clientID string

		// 2. Identification from Request
		clientID = c.Query("client_id")
		if clientID == "" {
			type basicReq struct {
				ClientID string `json:"client_id"`
			}
			var br basicReq
			_ = c.BodyParser(&br)
			clientID = br.ClientID
		}
		if clientID == "" {
			authHeader := c.Get("Authorization")
			if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
				token := after
				if claims, err := utils.ParseToken(token); err == nil {
					clientID = claims.ClientID
				}
			}
		}

		// 3. Cache Lookup & Validation
		if clientID != "" {
			var allowedOrigins []string
			if val, ok := originCache.Load(clientID); ok {
				allowedOrigins = val.([]string)
			} else {
				// Cache Miss: Query DB once
				client, err := repository.FindClientByID(clientID)
				if err == nil && client.WebOrigins != "" {
					allowedOrigins = strings.Split(client.WebOrigins, ",")
					// Cache the split result
					originCache.Store(clientID, allowedOrigins)
				}
			}

			// Validate
			for _, allowed := range allowedOrigins {
				if strings.TrimSpace(allowed) == origin {
					setHeaders(c, origin)
					break
				}
			}
		}

		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}

		return c.Next()
	}
}

// InvalidateOriginCache clears the cache when a client is updated or registered
func InvalidateOriginCache(clientID string) {
	originCache.Delete(clientID)
}
