package tests

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestCopiedRecords checks value records, generated getters, and dependency failures in both drivers.
func TestCopiedRecords(t *testing.T) {
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
		for _, target := range []string{
			"api", "accepted", "bodyless", "frozen", "fields", "writes", "transport", "badapi", "dependency",
		} {
			t.Run(mode+"/"+target, func(t *testing.T) {
				dir := consumerModule(t, root, "copied_records")
				accepted := target == "api" || target == "accepted" || target == "frozen" ||
					target == "transport" || target == "bodyless"
				output := runBoundary(t, dir, binary, mode, accepted, "./"+target)
				checkFixtureSnapshots(t, root, "copied_records/"+target, output)
			})
		}
	}
	t.Run("runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "copied_records")
		runCommand(t, dir, true, "go", "test", "-race", "-count=1", "./accepted")
	})
}
