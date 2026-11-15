package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestSnapshotSystem checks category and projection forms through the shipped drivers.
func TestSnapshotSystem(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	for _, mode := range []string{"cli", "vet"} {
		for _, target := range []string{"api", "accepted", "rejected", "badapi", "dependency"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "snapshot_system")
				accepted := target == "api" || target == "accepted"
				output := runBoundary(t, dir, binary, mode, accepted, "./"+target)
				checkFixtureSnapshots(t, root, "snapshot_system/"+target, output)
			})
		}
	}
}
