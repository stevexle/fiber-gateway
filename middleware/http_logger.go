package middleware

import (
	"slices"
	"fmt"
	"log/slog"
	"time"

	"github.com/fiber-gateway/config"
	"github.com/fiber-gateway/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/bytebufferpool"
)

func isSkipped(path string) bool {
	return slices.Contains(config.AppConfig.Logging.SkipPaths, path)
}

func getJSONString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return string(b)
}

func prettyHeaders(headers map[string][]string, buf *bytebufferpool.ByteBuffer) {
	if len(headers) == 0 {
		return
	}
	for k, v := range headers {
		buf.WriteString("  ")
		buf.WriteString(logger.ColorGray)
		buf.WriteString(k)
		buf.WriteString(":")
		buf.WriteString(logger.ColorReset)
		buf.WriteString(" ")
		fmt.Fprintf(buf, "%v", v)
		buf.WriteString("\n")
	}
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
		reqBody := getJSONString(c.Body())

		buf := bytebufferpool.Get()
		buf.Reset()
		fmt.Fprintf(buf, "%s[HTTP-REQ]%s %s %s %s", logger.ColorCyan, logger.ColorReset, ip, method, originalURL)
		
		if len(reqHeaders) > 0 {
			buf.WriteByte('\n')
			prettyHeaders(reqHeaders, buf)
		}
		
		if reqBody != "" {
			buf.WriteByte('\n')
			buf.WriteString(logger.ColorGray)
			buf.WriteString("Body:")
			buf.WriteString(logger.ColorReset)
			buf.WriteByte('\n')
			buf.WriteString(reqBody)
		}

		slog.Info(buf.String(),
			"ip", ip,
			"method", method,
			"path", path,
		)
		bytebufferpool.Put(buf)

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
		resBody := getJSONString(c.Response().Body())

		// Status Color
		statusColor := logger.ColorGreen
		if status >= 400 {
			statusColor = logger.ColorYellow
		}
		if status >= 500 {
			statusColor = logger.ColorRed
		}

		bufRes := bytebufferpool.Get()
		bufRes.Reset()
		fmt.Fprintf(bufRes, "%s[HTTP-RES]%s %s %s %s -> %s%d%s (%s)",
			logger.ColorCyan, logger.ColorReset, ip, method, originalURL, statusColor, status, logger.ColorReset, time.Since(start))

		if len(resHeaders) > 0 {
			bufRes.WriteByte('\n')
			prettyHeaders(resHeaders, bufRes)
		}
		
		if resBody != "" {
			bufRes.WriteByte('\n')
			bufRes.WriteString(logger.ColorGray)
			bufRes.WriteString("Body:")
			bufRes.WriteString(logger.ColorReset)
			bufRes.WriteByte('\n')
			bufRes.WriteString(resBody)
		}

		slog.Info(bufRes.String())
		bytebufferpool.Put(bufRes)

		// Return nil because the error was already handled by the global error handler
		// during status capture (i.e. response was written). 
		// If err was nil, we continue normally.
		return nil
	}
}
