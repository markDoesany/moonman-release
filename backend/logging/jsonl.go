package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JSONLWriter appends one JSON value per line to a local file.
type JSONLWriter struct {
	path string
	mu   sync.Mutex
}

func NewJSONLWriter(path string) *JSONLWriter { return &JSONLWriter{path: path} }

func (w *JSONLWriter) Append(value any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return fmt.Errorf("create structured log directory: %w", err)
	}
	contents, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal structured log entry: %w", err)
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open structured log: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(contents, '\n')); err != nil {
		return fmt.Errorf("write structured log: %w", err)
	}
	return nil
}
