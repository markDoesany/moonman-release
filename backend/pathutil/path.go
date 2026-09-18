package pathutil

import (
	"path/filepath"
	"strings"
)

// Resolve returns configured as an absolute path when it is already absolute,
// otherwise it resolves it relative to base. Both forms are accepted in YAML.
func Resolve(base, configured string) string {
	configured = strings.TrimSpace(configured)
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}
	if configured == "" {
		return filepath.Clean(base)
	}
	return filepath.Clean(filepath.Join(base, configured))
}

// RelativeIfInside stores a selected directory relative to base when possible.
// Paths outside base remain absolute.
func RelativeIfInside(base, selected string) string {
	baseAbs, baseErr := filepath.Abs(filepath.Clean(base))
	selectedAbs, selectedErr := filepath.Abs(filepath.Clean(selected))
	if baseErr != nil || selectedErr != nil {
		return filepath.Clean(selected)
	}
	relative, err := filepath.Rel(baseAbs, selectedAbs)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return selectedAbs
	}
	if relative == "." {
		return "."
	}
	return filepath.Clean(relative)
}
