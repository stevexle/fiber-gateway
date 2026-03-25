package middleware

import (
	"slices"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

func isSkipped(path string) bool {
	return slices.Contains(config.AppConfig.Logging.SkipPaths, path)
}

func prettyJSON(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err == nil {
		return out.String()
	}
	return string(b)
}

func prettyHeaders(headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}
	var b strings.Builder
	// Approximate size to reduce reallocations
	b.Grow(len(headers) * 64)
	for k, v := range headers {
		b.WriteString("  ")
		b.WriteString(logger.ColorGray)
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(logger.ColorReset)
		b.WriteString(" ")
		fmt.Fprintf(&b, "%v", v)
		b.WriteString("\n")
	}
	return b.String()
}

// HTTPLogger logs everything with respect to the HTTP request and response (headers, body, etc)
func HTTPLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fast path if logger is disabled at Info level
		if !slog.Default().Enabled(c.Context(), slog.LevelInfo) {
			return c.Next()
		}

		start := time.Now()
		method := c.Method()
		path := c.Path()
		originalURL := c.OriginalURL()
		ip := c.IP()

		// Skip logging for whitelisted paths (health checks, etc)
		if isSkipped(path) {
			return c.Next()
		}

		// --- Request Logging ---
		reqHeaders := c.GetReqHeaders()
		reqBody := prettyJSON(c.Body())

		var reqSummary strings.Builder
		reqSummary.Grow(256 + len(reqBody)) // Base size + body
		fmt.Fprintf(&reqSummary, "%s[HTTP-REQ]%s %s %s %s", logger.ColorCyan, logger.ColorReset, ip, method, originalURL)
		if h := prettyHeaders(reqHeaders); h != "" {
			reqSummary.WriteByte('\n')
			reqSummary.WriteString(h)
		}
		if reqBody != "" {
			reqSummary.WriteByte('\n')
			reqSummary.WriteString(logger.ColorGray)
			reqSummary.WriteString("Body:")
			reqSummary.WriteString(logger.ColorReset)
			reqSummary.WriteByte('\n')
			reqSummary.WriteString(reqBody)
		}

		slog.Info(reqSummary.String(),
			"ip", ip,
			"method", method,
			"path", path,
		)

		err := c.Next()

		// If an error occurred during the request chain, handle it with the global
		// error handler first so we can record the correct (e.g. 500) status code.
		if err != nil {
			if hErr := c.App().ErrorHandler(c, err); hErr != nil {
				// If the error handler itself fails, we must return its error
				return hErr
			}
		}

		// --- Response Logging ---
		status := c.Response().StatusCode()
		resHeaders := c.GetRespHeaders()
		resBody := prettyJSON(c.Response().Body())

		// Status Color
		statusColor := logger.ColorGreen
		if status >= 400 {
			statusColor = logger.ColorYellow
		}
		if status >= 500 {
			statusColor = logger.ColorRed
		}

		var resSummary strings.Builder
		resSummary.Grow(256 + len(resBody))
		fmt.Fprintf(&resSummary, "%s[HTTP-RES]%s %s %s %s -> %s%d%s (%s)",
			logger.ColorCyan, logger.ColorReset, ip, method, originalURL, statusColor, status, logger.ColorReset, time.Since(start))

		if h := prettyHeaders(resHeaders); h != "" {
			resSummary.WriteByte('\n')
			resSummary.WriteString(h)
		}
		if resBody != "" {
			resSummary.WriteByte('\n')
			resSummary.WriteString(logger.ColorGray)
			resSummary.WriteString("Body:")
			resSummary.WriteString(logger.ColorReset)
			resSummary.WriteByte('\n')
			resSummary.WriteString(resBody)
		}

		slog.Info(resSummary.String())

		// Return nil because the error was already handled by the global error handler
		// during status capture (i.e. response was written). 
		// If err was nil, we continue normally.
		return nil
	}
}
