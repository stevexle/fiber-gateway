package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/valyala/bytebufferpool"
)

// ansiRegex is used to strip terminal ANSI color codes before writing to a file logging sink.
var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// ansiStripper wraps an io.Writer and strips ANSI escape sequences.
type ansiStripper struct {
	w io.Writer
}

func (s *ansiStripper) Write(p []byte) (n int, err error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	last := 0
	indices := ansiRegex.FindAllIndex(p, -1)
	for _, match := range indices {
		buf.Write(p[last:match[0]])
		last = match[1]
	}
	buf.Write(p[last:])

	_, err = s.w.Write(buf.Bytes())
	return len(p), err
}

// RollingConfig sets up file rotation based on time, size and total size caps.
type RollingConfig struct {
	Filename   string // e.g. "logs/fiber-gateway.log"
	MaxSizeMB  int    // Maximum size in megabytes before rotation
	MaxAgeDays int    // Maximum number of days to retain old log files
	MaxBackups int    // Maximum number of old log files to retain
}

// NewRollingFile creates a rolling file that strips ANSI colors before dumping logs.
func NewRollingFile(config RollingConfig) io.Writer {
	if config.Filename == "" {
		config.Filename = "logs/fiber-gateway.log"
	}

	l := &nativeLogger{
		Config: config,
	}

	// ─── STARTUP ROTATION CHECK ───
	if info, err := os.Stat(config.Filename); err == nil {
		lastMod := info.ModTime()
		now := time.Now()
		if lastMod.Year() != now.Year() || lastMod.Month() != now.Month() || lastMod.Day() != now.Day() {
			_ = l.Rotate()
		}
	}
	l.archiveLogs() // Initial archive check

	// ─── DAILY ROTATION BACKGROUND PROCESS ───
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 1, 0, now.Location())
			time.Sleep(time.Until(next))

			if err := l.Rotate(); err != nil {
				slog.Error("Scheduled daily log rotation failed", slog.String("error", err.Error()))
			}
			l.archiveLogs()
		}
	}()

	// ─── JANITOR PROCESS ───
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			l.archiveLogs()
		}
	}()

	return &ansiStripper{w: l}
}

// nativeLogger implements a simple size-based log rotation logic.
type nativeLogger struct {
	Config RollingConfig
	mu     sync.Mutex
	file   *os.File
	size   int64
}

func (l *nativeLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		if err := l.openNew(); err != nil {
			return 0, err
		}
	}

	if l.size+int64(len(p)) > int64(l.Config.MaxSizeMB)*1024*1024 {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = l.file.Write(p)
	l.size += int64(n)
	return n, err
}

func (l *nativeLogger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rotate()
}

func (l *nativeLogger) openNew() error {
	dir := filepath.Dir(l.Config.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(l.Config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	l.file = f
	l.size = info.Size()
	return nil
}

func (l *nativeLogger) rotate() error {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	// Rename current log to timestamped name
	now := time.Now()
	timestamp := now.Format("2006-01-02T15-04-05.000")
	ext := filepath.Ext(l.Config.Filename)
	prefix := l.Config.Filename[:len(l.Config.Filename)-len(ext)]
	backupName := fmt.Sprintf("%s-%s%s", prefix, timestamp, ext)

	if err := os.Rename(l.Config.Filename, backupName); err != nil {
		return l.openNew() // Try to continue even if rename fails
	}

	// Compress in background
	go l.compressFile(backupName)

	return l.openNew()
}

func (l *nativeLogger) compressFile(src string) {
	dst := src + ".gz"
	f, err := os.Open(src)
	if err != nil {
		return
	}
	defer f.Close()

	gf, err := os.Create(dst)
	if err != nil {
		return
	}
	defer gf.Close()

	zw := gzip.NewWriter(gf)
	defer zw.Close()

	if _, err := io.Copy(zw, f); err != nil {
		return
	}

	zw.Close()
	gf.Close()
	f.Close()
	os.Remove(src)
}

func (l *nativeLogger) archiveLogs() {
	logDir := filepath.Dir(l.Config.Filename)
	base := filepath.Base(l.Config.Filename)
	ext := filepath.Ext(base)
	prefix := base[:len(base)-len(ext)]

	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-(\d{4}-\d{2}-\d{2})T.*\.log\.gz$`)

	var archivedFiles []string

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		matches := re.FindStringSubmatch(f.Name())
		if len(matches) < 2 {
			continue
		}

		dateDir := matches[1]
		archiveDir := filepath.Join(logDir, "archive", dateDir)
		_ = os.MkdirAll(archiveDir, 0755)

		oldPath := filepath.Join(logDir, f.Name())
		newPath := filepath.Join(archiveDir, f.Name())

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if err := os.Rename(oldPath, newPath); err == nil {
				archivedFiles = append(archivedFiles, newPath)
			}
		} else {
			archivedFiles = append(archivedFiles, newPath)
		}
	}

	l.cleanupArchives(logDir, prefix)
}

func (l *nativeLogger) cleanupArchives(logDir, prefix string) {
	// Find all compressed logs in archive subdirectories
	var allLogs []struct {
		path string
		time time.Time
	}

	archiveRoot := filepath.Join(logDir, "archive")
	_ = filepath.Walk(archiveRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), prefix+"-") && strings.HasSuffix(info.Name(), ".log.gz") {
			allLogs = append(allLogs, struct {
				path string
				time time.Time
			}{path, info.ModTime()})
		}
		return nil
	})

	// Sort logs by time (oldest first)
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].time.Before(allLogs[j].time)
	})

	// Cleanup by MaxBackups
	if l.Config.MaxBackups > 0 && len(allLogs) > l.Config.MaxBackups {
		toRemove := len(allLogs) - l.Config.MaxBackups
		for i := 0; i < toRemove; i++ {
			os.Remove(allLogs[i].path)
		}
		allLogs = allLogs[toRemove:]
	}

	// Cleanup by MaxAgeDays
	if l.Config.MaxAgeDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -l.Config.MaxAgeDays)
		for _, log := range allLogs {
			if log.time.Before(cutoff) {
				os.Remove(log.path)
			}
		}
	}
}
