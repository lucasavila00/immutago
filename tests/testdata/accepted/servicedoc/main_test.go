package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestFileCacheRefresh checks setup and a later background file refresh.
func TestFileCacheRefresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.txt")
	if err := os.WriteFile(path, []byte("initial\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root := supervisetest.NewRoot(t)
	cache, err := supervise.Start[FileCache](root, FileArgs{Path: im.FreezeString(path), Every: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if got := cache.Current().Text().Clone(); got != "initial\n" {
		t.Fatalf("initial text = %q", got)
	}
	if err := os.WriteFile(path, []byte("updated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for cache.Current().Text().Clone() != "updated\n" {
		if time.Now().After(deadline) {
			t.Fatal("background refresh did not publish the new text")
		}
		time.Sleep(time.Millisecond)
	}
}
