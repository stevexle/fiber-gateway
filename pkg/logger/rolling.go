package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
// Logback configuration mapping:
// <maxFileSize>300mb</maxFileSize>    -> MaxSizeMB: 300
// <totalSizeCap>20GB</totalSizeCap>   -> MaxBackups: 68  (20GB/300MB = 68 backups)
// <maxHistory>60</maxHistory>         -> MaxAgeDays: 60
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
		Compress:   true, // Automatically gzip archive files
	}

	// ─── DAILY ROTATION BACKGROUND PROCESS ───
	// Triggers at midnight to move the current log to a dated archive folder.
	// Pattern: ${LOG_FOLDER}/archive/YYYY-MM-DD/service.YYYY-MM-DD.i.log
	go func() {
		for {
			now := time.Now()
			// Calculate next midnight (00:00:01)
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 1, 0, now.Location())
			t := time.NewTimer(next.Sub(now))

			<-t.C // Wait for midnight

			// 1. Prepare Archive Path
			// We are archiving yesterday's logs
			yesterday := time.Now().Add(-24 * time.Hour)
			dateStr := yesterday.Format("2006-01-02")
			logDir := filepath.Dir(config.Filename)
			archiveDir := filepath.Join(logDir, "archive", dateStr)
			_ = os.MkdirAll(archiveDir, 0755)

			ext := filepath.Ext(config.Filename)
			base := filepath.Base(config.Filename)
			nameOnly := strings.TrimSuffix(base, ext)

			// 2. Find next available index (%i)
			i := 0
			var targetPath string
			for {
				targetPath = filepath.Join(archiveDir, fmt.Sprintf("%s.%s.%d%s", nameOnly, dateStr, i, ext))
				if _, err := os.Stat(targetPath); os.IsNotExist(err) {
					break
				}
				i++
			}

			// 3. Perform the move
			// Normal OS rename is safe on Mac/Linux even with open handles.
			// lumberjack will automatically create a new file on the next Write.
			if err := os.Rename(config.Filename, targetPath); err != nil {
				slog.Error("Daily log archive failed", slog.String("error", err.Error()), slog.String("path", targetPath))
			} else {
				// Also trigger lumberjack's internal rotation to clear older backups
				_ = roll.Rotate()
			}
		}
	}()

	return &ansiStripper{w: roll}
}
