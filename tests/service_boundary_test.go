package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestServiceBoundary checks service-only packages and imported Ref contracts through both drivers.
func TestServiceBoundary(t *testing.T) {
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
		for _, target := range []string{"valid", "direct", "capture", "contract", "dependency", "invalidresults", "strings"} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "service_boundary")
				output := runBoundary(t, dir, binary, mode, target != "invalidresults" && target != "strings", "./"+target)
				checkFixtureSnapshots(t, root, "service_boundary/"+target, output)
			})
		}
	}
}
