package ui

import (
	"fmt"
	"sync"
	"time"
)

type LogLevel int

const (
	INFO LogLevel = iota
	DEBUG
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

type LogBuffer struct {
	entries []LogEntry
	mu      sync.RWMutex
	maxSize int
}

var Logger *LogBuffer

func InitLogger(maxLines int) {
	Logger = &LogBuffer{
		entries: make([]LogEntry, 0, maxLines),
		maxSize: maxLines,
	}
}

func (l *LogBuffer) Printf(level LogLevel, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, v...)

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
	}

	if len(l.entries) >= l.maxSize {

		l.entries = l.entries[1:]
	}
	l.entries = append(l.entries, entry)
}

func Info(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf(INFO, format, v...)
	} else {
		fmt.Printf(format+"\n", v...)
	}
}

func Debug(format string, v ...interface{}) {
	if Logger != nil {
		Logger.Printf(DEBUG, format, v...)
	} else {

	}
}

func (l *LogBuffer) GetLines(showDebug bool, limit int) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var lines []string
	count := 0

	for _, entry := range l.entries {
		if entry.Level == DEBUG && !showDebug {
			continue
		}

		timeStr := entry.Timestamp.Format("15:04:05")
		prefix := "[INFO] "
		if entry.Level == DEBUG {
			prefix = "[DEBUG] "
		}

		lines = append(lines, fmt.Sprintf("%s %s%s", timeStr, prefix, entry.Message))
		count++
	}

	if limit > 0 && len(lines) > limit {
		return lines[len(lines)-limit:]
	}

	return lines
}
