package pathutil

import (
	"path/filepath"
	"runtime"
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

// ResolveOutputDirectory returns the directory that contains a component's
// build artifacts. Older project files sometimes stored the component root
// (or an empty value) here. Treating that as the output would package source
// files, so those legacy values use the conventional build directory.
func ResolveOutputDirectory(base, configured string) string {
	basePath := filepath.Clean(base)
	resolved := Resolve(basePath, configured)
	baseAbs, baseErr := filepath.Abs(basePath)
	resolvedAbs, resolvedErr := filepath.Abs(resolved)
	if strings.TrimSpace(configured) == "" || (baseErr == nil && resolvedErr == nil && samePath(baseAbs, resolvedAbs)) {
		return filepath.Join(basePath, "build")
	}
	return resolved
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
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
