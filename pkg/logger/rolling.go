package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ansiRegex is used to strip terminal ANSI color codes before writing to a file logging sink.
var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// ansiStripper wraps an io.Writer and strips ANSI escape sequences.
type ansiStripper struct {
	w io.Writer
}

func (s *ansiStripper) Write(p []byte) (n int, err error) {
	clean := ansiRegex.ReplaceAll(p, []byte(""))
	_, err = s.w.Write(clean)
	// We return len(p) so that io.MultiWriter and callers believe the entire exact byte slice was written.
	return len(p), err
}

// RollingConfig sets up file rotation based on time, size and total size caps.
// Matches the behavior of Logback SizeAndTimeBasedRollingPolicy.
type RollingConfig struct {
	Filename   string // e.g. "logs/fiber-gateway.log"
	MaxSizeMB  int    // Corresponds to <maxFileSize>
	MaxAgeDays int    // Corresponds to <maxHistory>
	MaxBackups int    // Corresponds to <totalSizeCap> limit. (MaxBackups = totalSizeCapMB / MaxSizeMB)
}

// NewRollingFile creates a rolling file that strips ANSI colors before dumping logs.
func NewRollingFile(config RollingConfig) io.Writer {
	if config.Filename == "" {
		config.Filename = "logs/fiber-gateway.log"
	}
	// Automatically ensure that log folder exists
	_ = os.MkdirAll(filepath.Dir(config.Filename), 0755)

	roll := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSizeMB,
		MaxAge:     config.MaxAgeDays,
		MaxBackups: config.MaxBackups,
		Compress:   true,
	}

	// ─── STARTUP ROTATION CHECK ───
	// If the application starts and the existing log file is from a previous day,
	// rotate it immediately to maintain daily consistency.
	if info, err := os.Stat(config.Filename); err == nil {
		lastMod := info.ModTime()
		now := time.Now()
		if lastMod.Year() != now.Year() || lastMod.Month() != now.Month() || lastMod.Day() != now.Day() {
			_ = roll.Rotate()
		}
	}

	// ─── DAILY ROTATION BACKGROUND PROCESS ───
	go func() {
		for {
			now := time.Now()
			// Calculate next midnight (00:00:01)
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 1, 0, now.Location())
			time.Sleep(time.Until(next))

			if err := roll.Rotate(); err != nil {
				slog.Error("Scheduled daily log rotation failed", slog.String("error", err.Error()))
			}
		}
	}()

	return &ansiStripper{w: roll}
}
