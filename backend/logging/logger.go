package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger writes human-readable application events to a local file.
type Logger struct {
	path string
	mu   sync.Mutex
}

func New(path string) *Logger { return &Logger{path: path} }

func (l *Logger) Info(message string)  { l.write("INFO", message) }
func (l *Logger) Error(message string) { l.write("ERROR", message) }

func (l *Logger) write(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s [%s] %s\n", time.Now().Format(time.RFC3339), level, message)
}
