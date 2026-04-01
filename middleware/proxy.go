package middleware

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/fiber-gateway/pkg/balancer"
	"github.com/fiber-gateway/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

var (
	headerUserID    = []byte("X-User-ID")
	headerRole      = []byte("X-Role")
	headerRealIP    = []byte("X-Real-IP")
	headerProto     = []byte("X-Forwarded-Proto")
	headerHost      = []byte("X-Forwarded-Host")
	headerRequestID = []byte("X-Request-ID")
)

func ReverseProxy(lb balancer.Balancer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Performance: Cache common locals
		userID := c.Locals("user_id")
		role := c.Locals("role")

		// Performance: Fast path for headers without reflection using byte slices
		if userID != nil {
			var userIDBytes []byte
			switch v := userID.(type) {
			case uint:
				userIDBytes = fasthttp.AppendUint(nil, int(v))
			case int:
				userIDBytes = fasthttp.AppendUint(nil, v)
			default:
				// Fallback only for unknown types, still using a buffer to avoid string allocation
				userIDBytes = fmt.Appendf(nil, "%v", v)
			}
			c.Request().Header.SetBytesKV(headerUserID, userIDBytes)
		}

		if role != nil {
			if r, ok := role.(string); ok {
				c.Request().Header.SetBytesKV(headerRole, []byte(r))
			} else {
				c.Request().Header.SetBytesKV(headerRole, fmt.Appendf(nil, "%v", role))
			}
		}

		// Standard Proxy Headers (Nginx style) using pre-allocated byte keys
		c.Request().Header.SetBytesKV(headerRealIP, []byte(c.IP()))
		c.Request().Header.SetBytesKV(headerProto, []byte(c.Protocol()))
		c.Request().Header.SetBytesKV(headerHost, []byte(c.Hostname()))

		// Performance: Safe path extraction
		originalPath := c.Path()
		trimmedPath := originalPath
		if strings.HasPrefix(originalPath, "/api/v1/") {
			trimmedPath = originalPath[7:]
		} else if originalPath == "/api/v1" {
			trimmedPath = "/"
		}

		// Observability: Add a Request-ID if not present
		reqID := c.Get(fiber.HeaderXRequestID)
		if reqID == "" {
			reqID, _ = utils.GenerateRandomString(16)
			c.Set(fiber.HeaderXRequestID, reqID)
		}
		c.Request().Header.SetBytesKV(headerRequestID, []byte(reqID))

		var lastTarget string
		var lastErr error
		maxRetries := 3
		allTargets := lb.Targets()

		for i := range maxRetries {
			target := lb.Next()

			// Diagnostics: what did we start with on retry?
			if i > 0 {
				slog.Info("Balancer picked target", "attempt", i+1, "picked", target, "last_failed", lastTarget)
			}

			// Avoid repeating the same failing target if we have alternatives
			if i > 0 && target == lastTarget && len(allTargets) > 1 {
				for idx, t := range allTargets {
					if t == lastTarget {
						newTarget := allTargets[(idx+1)%len(allTargets)]
						slog.Info("Jumping to next available target", "from", target, "to", newTarget)
						target = newTarget
						break
					}
				}
			}

			if target == "" {
				return c.Status(fiber.StatusServiceUnavailable).SendString("No upstream targets available")
			}

			lastTarget = target

			// Pre-allocate buffer for target URL construction to avoid multiple string allocations
			buf := bytebufferpool.Get()
			buf.Reset()
			buf.WriteString(target)
			buf.WriteString(trimmedPath)
			targetURL := buf.String()
			bytebufferpool.Put(buf)

			// Track active connection
			lb.Update(target, 1)

			// Ensure the target path is correctly pointed at for this specific backend
			c.Request().URI().SetPath(trimmedPath)

			if i > 0 {
				slog.Warn("Retrying proxy request", "attempt", i+1, "target", target)
			} else {
				// Base log for initial attempt
				slog.Info("Proxying request", "target", target, "path", originalPath)
			}

			err := proxy.Do(c, targetURL)
			lb.Update(target, -1)

			if err == nil {
				return nil
			}

			lastErr = err
			if i < maxRetries-1 {
				slog.Warn("Proxy attempt failed, retrying next target", "error", err, "target", target, "attempt", i+1)
			} else {
				slog.Error("Final proxy attempt failed", "error", err, "target", target, "attempt", i+1)
			}
		}

		return lastErr
	}
}
