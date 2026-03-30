package middleware

import (
	"strings"
	"sync"
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

		var clientID string

		// 1. Identification
		clientID = c.Query("client_id")
		if clientID == "" {
			type basicReq struct { ClientID string `json:"client_id"` }
			var br basicReq
			_ = c.BodyParser(&br)
			clientID = br.ClientID
		}
		if clientID == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if claims, err := utils.ParseToken(token); err == nil {
					clientID = claims.ClientID
				}
			}
		}

		// 2. Cache Lookup & Validation
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
					c.Set("Access-Control-Allow-Origin", origin)
					c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
					c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
					c.Set("Access-Control-Allow-Credentials", "true")
					c.Set("Access-Control-Max-Age", "3600")
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
