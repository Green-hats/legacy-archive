package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ani-rss/internal/config"
	"ani-rss/internal/model"
)

var (
	mu       sync.RWMutex
	entries  []model.Log
	logFile  *os.File
	logsDir  string
	maxSize  = 128
)

// Init sets up the log file under the config dir.
func Init() error {
	logsDir = config.ConfigDirFile("logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(logsDir, "ani-rss.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	logFile = f
	return nil
}

// SetMaxSize updates the ring buffer size from config.
func SetMaxSize(n int) {
	mu.Lock()
	defer mu.Unlock()
	if n > 0 {
		maxSize = n
	}
	if len(entries) > maxSize {
		entries = entries[len(entries)-maxSize:]
	}
}

// Write appends a log entry.
func Write(level, loggerName, msg string) {
	if msg == "" {
		return
	}
	e := model.Log{Message: msg, Level: level, LoggerName: loggerName, ThreadName: "main"}
	mu.Lock()
	entries = append(entries, e)
	if len(entries) > maxSize {
		entries = entries[len(entries)-maxSize:]
	}
	mu.Unlock()
	if logFile != nil {
		ts := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(logFile, "%s %-5s [%s] %s\n", ts, level, loggerName, msg)
	}
}

// Info writes an info log.
func Info(loggerName, msg string) { Write("INFO", loggerName, msg) }

// Warn writes a warn log.
func Warn(loggerName, msg string) { Write("WARN", loggerName, msg) }

// Warnf writes a formatted warn log.
func Warnf(loggerName, format string, args ...interface{}) {
	Warn(loggerName, fmt.Sprintf(format, args...))
}

// Error writes an error log.
func Error(loggerName, msg string) { Write("ERROR", loggerName, msg) }

// Debug writes a debug log (only shown when debug enabled).
func Debug(loggerName, msg string) {
	if config.Get().Debug {
		Write("DEBUG", loggerName, msg)
	}
}

// Debugf writes a formatted debug log.
func Debugf(loggerName, format string, args ...interface{}) {
	Debug(loggerName, fmt.Sprintf(format, args...))
}

// InfoF writes a formatted info log.
func Infof(loggerName, format string, args ...interface{}) {
	Info(loggerName, fmt.Sprintf(format, args...))
}

// ErrorF writes a formatted error log.
func Errorf(loggerName, format string, args ...interface{}) {
	Error(loggerName, fmt.Sprintf(format, args...))
}

// List returns a copy of the ring buffer entries.
func List() []model.Log {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]model.Log, len(entries))
	copy(out, entries)
	return out
}

// Clear empties the in-memory buffer and log file.
func Clear() {
	mu.Lock()
	entries = nil
	mu.Unlock()
	if logFile != nil {
		logFile.Truncate(0)
		logFile.Seek(0, 0)
	}
}

// LogsDir returns the logs directory path.
func LogsDir() string { return logsDir }

// Close flushes the log file.
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}