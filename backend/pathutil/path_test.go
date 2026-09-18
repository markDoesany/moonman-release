package pathutil

import (
	"path/filepath"
	"testing"
)

func TestResolveOutputDirectoryUsesBuildForLegacyRootValues(t *testing.T) {
	root := t.TempDir()
	for _, configured := range []string{"", ".", root} {
		got := ResolveOutputDirectory(root, configured)
		want := filepath.Join(root, "build")
		if got != want {
			t.Fatalf("ResolveOutputDirectory(%q) = %q, want %q", configured, got, want)
		}
	}
}

func TestResolveOutputDirectoryPreservesExplicitFolder(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "dist")
	if got := ResolveOutputDirectory(root, "dist"); got != want {
		t.Fatalf("ResolveOutputDirectory(dist) = %q, want %q", got, want)
	}
}
