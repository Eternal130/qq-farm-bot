package bot

import (
	"fmt"
	"sync"
	"time"

	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

// Logger provides structured logging for a bot instance.
// Logs are stored in SQLite and broadcast to WebSocket subscribers.
type Logger struct {
	accountID   int64
	store       *store.Store
	subscribers map[chan *model.LogEntry]struct{}
	mu          sync.RWMutex
}

func NewLogger(accountID int64, s *store.Store) *Logger {
	return &Logger{
		accountID:   accountID,
		store:       s,
		subscribers: make(map[chan *model.LogEntry]struct{}),
	}
}

func (l *Logger) Info(tag, msg string) {
	l.emit("info", tag, msg)
}

func (l *Logger) Infof(tag, format string, args ...interface{}) {
	l.emit("info", tag, fmt.Sprintf(format, args...))
}

func (l *Logger) Warn(tag, msg string) {
	l.emit("warn", tag, msg)
}

func (l *Logger) Warnf(tag, format string, args ...interface{}) {
	l.emit("warn", tag, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(tag, format string, args ...interface{}) {
	l.emit("error", tag, fmt.Sprintf(format, args...))
}

func (l *Logger) emit(level, tag, msg string) {
	entry := &model.LogEntry{
		AccountID: l.accountID,
		Tag:       tag,
		Message:   msg,
		Level:     level,
		CreatedAt: time.Now(),
	}

	// Store in database (fire-and-forget)
	if l.store != nil {
		_ = l.store.AddLog(entry)
	}

	// Broadcast to subscribers
	l.mu.RLock()
	for ch := range l.subscribers {
		select {
		case ch <- entry:
		default: // drop if channel full
		}
	}
	l.mu.RUnlock()

	// Also print to stdout
	fmt.Printf("[%s] [账号#%d] [%s] %s\n", time.Now().Format("15:04:05"), l.accountID, tag, msg)
}

// Subscribe returns a channel that receives log entries. Call Unsubscribe to stop.
func (l *Logger) Subscribe() chan *model.LogEntry {
	ch := make(chan *model.LogEntry, 100)
	l.mu.Lock()
	l.subscribers[ch] = struct{}{}
	l.mu.Unlock()
	return ch
}

func (l *Logger) Unsubscribe(ch chan *model.LogEntry) {
	l.mu.Lock()
	delete(l.subscribers, ch)
	l.mu.Unlock()
	close(ch)
}
