// Package logger provides a [slog.Handler] that formats log records in
// Logback's classic pattern layout:
//
//	2006-01-02 15:04:05.000 INFO  [goroutine-1] fiber-gateway: message key=value
//
// The format mirrors the following Logback pattern:
//
//	%d{yyyy-MM-dd HH:mm:ss.SSS} %-5level [%thread] %logger{36}: %msg%n
//
// # Quick start
//
//	log := logger.New("my-service", slog.LevelInfo)
//	log.Info("server started", "port", 3000)
//
// A package-level [Default] logger and convenience wrappers ([Info], [Warn],
// [Error], [Debug]) are provided for services that need a single global logger.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

// GlobalWriter is the default writer used for non-slog output (e.g. pretty console logs).
// It can be overridden in main.go to point to a multi-writer (stdout + file).
var GlobalWriter io.Writer = os.Stdout

var (
	pid        = os.Getpid()
	pidStr     = fmt.Sprintf("pid-%d/goroutine-", pid)
)

// ─── Goroutine ID ────────────────────────────────────────────────────────────

// goroutineID extracts the current goroutine ID from the runtime stack.
// Optimized for zero-allocations and speed.
func goroutineID() int64 {
	var buf [32]byte
	n := runtime.Stack(buf[:], false)
	if n < 11 { // Length of "goroutine " is 10
		return 0
	}
	// The stack starts with "goroutine 123456 ..."
	// We parse the digits directly from the byte slice to avoid string copies.
	var id int64
	for i := 10; i < n; i++ {
		c := buf[i]
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + int64(c-'0')
	}
	return id
}

// ─── Level formatting ─────────────────────────────────────────────────────────

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// ─── Logger name ─────────────────────────────────────────────────────────────

// truncateName truncates s to at most n characters from the right,
// prepending "..." when truncated. Mirrors logback's %logger{36}.
func truncateName(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-(n-3):]
}

// ─── Handler ─────────────────────────────────────────────────────────────────

// LogbackHandler is a [slog.Handler] that formats log records using Logback's
// classic pattern layout. Each record is written as a single line:
//
//	2006-01-02 15:04:05.000 INFO  [goroutine-1] fiber-gateway: message key=value
//
// Use [New] or [NewWithWriter] to construct a [*slog.Logger] backed by this handler.
type LogbackHandler struct {
	name  string // %logger{36}
	level slog.Level
	out   io.Writer
	attrs []slog.Attr // inherited attributes (WithAttrs)
}

// New returns a [*slog.Logger] that writes Logback-formatted records to stdout.
//
// name is the logger name shown in the %logger field (truncated to 36 chars).
// level is the minimum level that will be emitted.
//
// Example:
//
//	log := logger.New("fiber-gateway", slog.LevelInfo)
//	log.Info("listening", "port", 3000)
//	// 2026-03-23 11:00:00.000 INFO  [goroutine-1] fiber-gateway: listening port=3000
func New(name string, level slog.Level) *slog.Logger {
	return slog.New(&LogbackHandler{
		name:  name,
		level: level,
		out:   os.Stdout,
	})
}

// NewWithWriter is like [New] but writes records to w instead of stdout.
// Useful for directing output to a file, buffer, or [os.Stderr].
//
// Example:
//
//	f, _ := os.Create("app.log")
//	log := logger.NewWithWriter("fiber-gateway", slog.LevelDebug, f)
func NewWithWriter(name string, level slog.Level, w io.Writer) *slog.Logger {
	return slog.New(&LogbackHandler{
		name:  name,
		level: level,
		out:   w,
	})
}

// Enabled reports whether the handler handles records at the given level.
// It implements [slog.Handler].
func (h *LogbackHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats r as a single Logback-style log line and writes it to the
// handler's output writer. It implements [slog.Handler].
//
// The output format is:
//
//	<timestamp> <LEVEL> [goroutine-<id>] <name>: <message> [key=value ...]
// Handle formats r as a single Logback-style log line and writes it to the
// handler's output writer. It implements [slog.Handler].
func (h *LogbackHandler) Handle(_ context.Context, r slog.Record) error {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	// 1. Timestamp (Zero-copy append)
	buf.WriteString(ColorGray)
	buf.B = r.Time.AppendFormat(buf.B, "2006-01-02 15:04:05.000")
	buf.WriteString(ColorReset)
	buf.WriteByte(' ')

	// 2. Level
	h.appendLevel(buf, r.Level)
	buf.WriteByte(' ')

	// 3. PID/Goroutine ID (Optimized append)
	buf.WriteString(ColorPurple)
	buf.WriteString(pidStr)
	buf.B = fasthttp.AppendUint(buf.B, int(goroutineID()))
	buf.WriteString(ColorReset)
	buf.WriteByte(' ')

	// 4. Logger Name (Source File)
	var srcFile string
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			srcFile = filepath.Base(f.File)
		}
	}
	if srcFile == "" {
		srcFile = h.name
	}
	buf.WriteString(ColorGreen)
	buf.WriteString(truncateName(srcFile, 36))
	buf.WriteString(ColorReset)
	buf.WriteString(": ")

	// 5. Message
	buf.WriteString(r.Message)

	// 6. Inherited Attrs
	for _, a := range h.attrs {
		buf.WriteByte(' ')
		buf.WriteString(a.Key)
		buf.WriteByte('=')
		appendValue(buf, a.Value)
	}

	// 7. Record Attrs
	r.Attrs(func(a slog.Attr) bool {
		buf.WriteByte(' ')
		buf.WriteString(a.Key)
		buf.WriteByte('=')
		appendValue(buf, a.Value)
		return true
	})

	// 8. Newline
	buf.WriteByte('\n')

	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *LogbackHandler) appendLevel(buf *bytebufferpool.ByteBuffer, l slog.Level) {
	switch {
	case l >= slog.LevelError:
		buf.WriteString(ColorRed)
		buf.WriteString("ERROR")
	case l >= slog.LevelWarn:
		buf.WriteString(ColorYellow)
		buf.WriteString("WARN ")
	case l >= slog.LevelInfo:
		buf.WriteString(ColorCyan)
		buf.WriteString("INFO ")
	default:
		buf.WriteString(ColorPurple)
		buf.WriteString("DEBUG")
	}
	buf.WriteString(ColorReset)
}

func appendValue(buf *bytebufferpool.ByteBuffer, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		buf.WriteString(v.String())
	case slog.KindInt64:
		buf.B = fasthttp.AppendUint(buf.B, int(v.Int64()))
	case slog.KindBool:
		if v.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case slog.KindDuration:
		buf.WriteString(v.Duration().String())
	case slog.KindTime:
		buf.B = v.Time().AppendFormat(buf.B, time.RFC3339)
	default:
		buf.B = fmt.Appendf(buf.B, "%v", v.Any())
	}
}

// WithAttrs returns a new handler whose attributes consist of both the
// receiver's attributes and attrs. It implements [slog.Handler].
func (h *LogbackHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(merged, h.attrs)
	copy(merged[len(h.attrs):], attrs)
	return &LogbackHandler{name: h.name, level: h.level, out: h.out, attrs: merged}
}

// WithGroup returns a new handler with the given group name appended to the
// logger name using dot notation (e.g. "fiber-gateway.db"), mirroring
// Logback's logger hierarchy. It implements [slog.Handler].
func (h *LogbackHandler) WithGroup(name string) slog.Handler {
	return &LogbackHandler{
		name:  h.name + "." + name,
		level: h.level,
		out:   h.out,
		attrs: h.attrs,
	}
}

// ─── Convenience wrappers ─────────────────────────────────────────────────────

// Default is the package-level logger for the fiber-gateway service,
// writing INFO and above records to stdout.
var Default = New("fiber-gateway", slog.LevelInfo)

// Info logs a message at [slog.LevelInfo] using the [Default] logger.
func Info(msg string, args ...any) { Default.Info(msg, args...) }

// Warn logs a message at [slog.LevelWarn] using the [Default] logger.
func Warn(msg string, args ...any) { Default.Warn(msg, args...) }

// Error logs a message at [slog.LevelError] using the [Default] logger.
func Error(msg string, args ...any) { Default.Error(msg, args...) }

// Debug logs a message at [slog.LevelDebug] using the [Default] logger.
func Debug(msg string, args ...any) { Default.Debug(msg, args...) }

// FiberTimeFormat is the timestamp layout used by Fiber's logger middleware,
// kept consistent with [LogbackHandler]'s timestamp format.
const FiberTimeFormat = "2006-01-02 15:04:05.000"

// FiberLogFormat returns a format string for Fiber's logger middleware that
// produces output visually consistent with [LogbackHandler].
//
// Pass the result to [fiblogger.Config.Format] and [FiberTimeFormat] to
// [fiblogger.Config.TimeFormat]:
//
//	app.Use(fiblogger.New(fiblogger.Config{
//	    Format:     logger.FiberLogFormat("fiber-gateway"),
//	    TimeFormat: logger.FiberTimeFormat,
//	}))
//
func FiberLogFormat(serviceName string) string {
	name := truncateName(serviceName, 36)
	pid := os.Getpid()
	return fmt.Sprintf(
		"%s${time}%s %sINFO %s %s[pid-%d]%s %s%s%s: ${method} ${path} ${status} ${latency}\n",
		ColorGray, ColorReset,
		ColorCyan, ColorReset,
		ColorPurple, pid, ColorReset,
		ColorGreen, name, ColorReset,
	)
}

// ParseLevel converts a log level string to a [slog.Level].
// The comparison is case-insensitive. Unrecognised values default to [slog.LevelInfo].
//
//	"debug"           → slog.LevelDebug
//	"info"            → slog.LevelInfo
//	"warn"/"warning"  → slog.LevelWarn
//	"error"           → slog.LevelError
//
// Example:
//
//	level := logger.ParseLevel(os.Getenv("LOG_LEVEL"))
//	log := logger.New("my-service", level)
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Since returns the elapsed time since start formatted as milliseconds with
// three decimal places (e.g. "1.234ms"). Useful for timing log entries.
//
// Example:
//
//	start := time.Now()
//	rows, err := db.Query(ctx, sql)
//	logger.Info("query done", "elapsed", logger.Since(start))
func Since(start time.Time) string {
	return fmt.Sprintf("%.3fms", float64(time.Since(start).Microseconds())/1000)
}
