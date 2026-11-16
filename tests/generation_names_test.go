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
			for _, name := range []string{"api", "privateapi", "accepted", "local_rejections", "record_isnil",
				"record_unchecked", "interface_isnil", "private_value", "private_pointer",
				"private_array", "private_pointer_array"} {
				t.Run(name, func(t *testing.T) {
					valid := name == "api" || name == "privateapi" || name == "accepted"
					output := runBoundary(t, dir, binary, mode, valid, "./"+name)
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
			output := runBoundary(t, dir, binary, mode, false, "./schema")
			checkFixtureSnapshots(t, root, "generation_removal/schema", output)
			output = runBoundary(t, dir, binary, mode, false, "./consumer")
			fixture := filepath.Join(root, "tests/testdata/generation_removal")
			packages, err := packageOutputs(fixture, fixture, output)
			if err != nil {
				t.Fatal(err)
			}
			checkSnapshot(t, filepath.Join(fixture, "consumer/output.txt"), packages[filepath.Join(fixture, "consumer")])
			// Vet can also print the dependency's diagnostic when it runs that analysis again.
			if source := packages[filepath.Join(fixture, "schema")]; source != "" {
				checkSnapshot(t, filepath.Join(fixture, "schema/output.txt"), source)
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

// TestGenerationSchemas keeps each invalid schema independent of operation fixtures.
func TestGenerationSchemas(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	entries, err := os.ReadDir(filepath.Join(root, "tests/testdata/generation_rejected"))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"cli", "vet"} {
		dir := consumerModule(t, root, "generation_rejected")
		for _, entry := range entries {
			t.Run(mode+"/"+entry.Name(), func(t *testing.T) {
				output := runBoundary(t, dir, binary, mode, false, "./"+entry.Name())
				checkFixtureSnapshots(t, root, "generation_rejected/"+entry.Name(), output)
			})
		}
	}
}
