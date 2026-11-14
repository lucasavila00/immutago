package tests

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCgoDiagnostics checks real compiler wrappers and dependency facts in both drivers.
func TestCgoDiagnostics(t *testing.T) {
	output, err := exec.Command("go", "env", "CGO_ENABLED").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(output)) != "1" {
		t.Skip("cgo is disabled")
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	dir := consumerModule(t, root, "cgo")
	runCommand(t, dir, true, "go", "build", "./...")
	for _, mode := range []string{"cli", "vet"} {
		t.Run(mode, func(t *testing.T) {
			for _, pkg := range []string{"cgolib", "consumer"} {
				t.Run(pkg, func(t *testing.T) {
					command, arguments := binary, []string{"./" + pkg}
					if mode == "vet" {
						command, arguments = "go", []string{"vet", "-vettool=" + binary, "./" + pkg}
					}
					output := runCommand(t, dir, false, command, arguments...)
					checkFixtureSnapshots(t, root, "cgo/"+pkg, output)
				})
			}
		})
	}
}
