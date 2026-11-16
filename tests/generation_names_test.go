package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/projectiongen"
)

// TestGenerationNames checks accepted source names and nearby rejected forms through both drivers.
func TestGenerationNames(t *testing.T) {
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
		t.Run(mode, func(t *testing.T) {
			dir := consumerModule(t, root, "generation_names")
			for _, name := range []string{"accepted", "local_rejections", "record_isnil", "record_unsafe",
				"interface_isnil", "private_value", "private_pointer", "private_array", "private_pointer_array"} {
				t.Run(name, func(t *testing.T) {
					output := runBoundary(t, dir, binary, mode, name == "accepted", "./"+name)
					checkFixtureSnapshots(t, root, "generation_names/"+name, output)
				})
			}
		})
	}
}

// TestGenerationRemoval rejects stale output and its dependents until the reported repair is applied.
func TestGenerationRemoval(t *testing.T) {
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
		t.Run(mode, func(t *testing.T) {
			dir := consumerModule(t, root, "generation_removal")
			runCommand(t, dir, true, "go", "get", "-tool", "github.com/lucasavila00/immutago/cmd/immutago-gen")
			runBoundary(t, dir, binary, mode, true, "./...")
			path := filepath.Join(dir, "schema", "schema.go")
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(source), "im.Frozen", "im.Local", 1)
			changed = strings.Replace(changed, projectiongen.Directive, "", 1)
			writeFile(t, path, changed)
			runCommand(t, dir, true, "go", "build", "./...")
			for _, name := range []string{"schema", "consumer"} {
				output := runBoundary(t, dir, binary, mode, false, "./"+name)
				checkSnapshot(t, filepath.Join(root, "tests/testdata/generation_removal", name, "output.txt"),
					normalizeOutput(output))
			}
			runCommand(t, filepath.Dir(path), true, "go", "run",
				"github.com/lucasavila00/immutago/cmd/immutago-gen")
			runBoundary(t, dir, binary, mode, true, "./...")
			if err := os.Remove(filepath.Join(filepath.Dir(path), projectiongen.Filename)); err != nil {
				t.Fatal(err)
			}
			runBoundary(t, dir, binary, mode, true, "./...")
		})
	}
}
