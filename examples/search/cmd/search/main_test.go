package main

import (
	"context"
	"testing"
	"time"
)

// TestRunStopsOnCancellation checks HTTP and service shutdown together.
func TestRunStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, t.TempDir(), "127.0.0.1:0", time.Second); err != nil {
		t.Fatal(err)
	}
}

// TestRunInitialFailure checks that the command returns the first scan error.
func TestRunInitialFailure(t *testing.T) {
	if err := run(context.Background(), t.TempDir()+"/missing", "127.0.0.1:0", time.Second); err == nil {
		t.Fatal("command accepted a missing document directory")
	}
}
