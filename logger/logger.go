package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const maxLogSize = 5 * 1024 * 1024 // 5 MB

var out io.Writer = os.Stderr // fallback if file not init'd

// Init opens (or creates) the log file in %APPDATA%\RebornPluginAutoinstaller\.
// Call once at startup. Safe to call multiple times.
func Init() {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(appData, "RebornPluginAutoinstaller")
	os.MkdirAll(dir, 0755)

	logPath := filepath.Join(dir, "debug.log")

	// Rotate if too large
	if info, err := os.Stat(logPath); err == nil && info.Size() > maxLogSize {
		os.Rename(logPath, logPath+".old")
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("logger: cannot open log file: %v", err)
		return
	}

	out = io.MultiWriter(os.Stderr, f)
	Info("=== App started ===")
}

// Info logs an informational message.
func Info(format string, args ...interface{}) { write("INFO", format, args...) }

// Warn logs a warning message.
func Warn(format string, args ...interface{}) { write("WARN", format, args...) }

// Error logs an error message.
func Error(format string, args ...interface{}) { write("ERROR", format, args...) }

// Debug logs a debug message.
func Debug(format string, args ...interface{}) { write("DEBUG", format, args...) }

func write(level, format string, args ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] [%s] %s\n", ts, level, msg)
	fmt.Fprint(out, line)
}
