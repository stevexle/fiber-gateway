package middleware

import (
	"fmt"
	"time"

	"github.com/fiber-gateway/schemas/response"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimiter returns a middleware that limits requests per IP or User ID.
// max: maximum number of requests allowed within the duration.
// expiration: the time window for the rate limit (e.g. 1 * time.Minute).
func RateLimiter(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// If user is authenticated, limit by User ID
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("user_%v", userID)
			}
			// Fallback to IP address
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.SendError(c, fiber.StatusTooManyRequests, "Too many requests, please try again later.")
		},
	})
}
