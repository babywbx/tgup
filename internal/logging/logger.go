package logging

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Logger writes leveled operational logs.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// SimpleLogger is a minimal logger with redaction support.
type SimpleLogger struct {
	mu       sync.Mutex
	out      io.Writer
	redactor Redactor
}

// NewSimpleLogger creates a logger writing to out.
func NewSimpleLogger(out io.Writer, redactor Redactor) *SimpleLogger {
	return &SimpleLogger{
		out:      out,
		redactor: redactor,
	}
}

// Infof logs an info-level message.
func (l *SimpleLogger) Infof(format string, args ...any) {
	l.logf("info", format, args...)
}

// Warnf logs a warn-level message.
func (l *SimpleLogger) Warnf(format string, args ...any) {
	l.logf("warn", format, args...)
}

// Errorf logs an error-level message.
func (l *SimpleLogger) Errorf(format string, args ...any) {
	l.logf("error", format, args...)
}

func (l *SimpleLogger) logf(level string, format string, args ...any) {
	if l == nil || l.out == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = l.redactor.Redact(msg)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().UTC().Format(time.RFC3339), strings.ToUpper(level), msg)

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.out, line)
}
