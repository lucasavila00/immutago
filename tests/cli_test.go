// Package tests exercises the built analyzer against independent consumer modules.
package tests

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestCLI builds the shipped command and checks valid input, diagnostics, JSON, and go vet integration.
func TestCLI(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "immutago")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	runCommand(t, root, true, "go", "build", "-o", binary, "./cmd/immutago")
	t.Run("accepted", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		output := runCommand(t, dir, true, binary, "./...")
		if output != "" {
			t.Fatalf("unexpected diagnostics:\n%s", output)
		}
	})
	t.Run("ordinary_callback_library", func(t *testing.T) {
		dir := consumerModule(t, root, "rejected")
		output := runCommand(t, dir, true, binary, "./callback_api")
		if output != "" {
			t.Fatalf("unrelated callback library rejected:\n%s", output)
		}
	})
	t.Run("rejected", func(t *testing.T) {
		dir := consumerModule(t, root, "rejected")
		output := runCommand(t, dir, false, binary, "./...")
		checkDiagnostics(t, dir, output)
	})
	t.Run("vet", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		runCommand(t, dir, true, "go", "vet", "-vettool="+binary, "./...")
		dir = consumerModule(t, root, "rejected")
		output := runCommand(t, dir, false, "go", "vet", "-vettool="+binary, "./...")
		checkDiagnostics(t, dir, output)
	})
	t.Run("json", func(t *testing.T) {
		dir := consumerModule(t, root, "rejected")
		// The standard analysis driver exits successfully in JSON mode even with findings.
		output := runCommand(t, dir, true, binary, "-json", "./...")
		var result map[string]map[string]json.RawMessage
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, output)
		}
		if len(result) == 0 || !strings.Contains(output, "immutable field") {
			t.Fatalf("missing findings: %s", output)
		}
	})
	t.Run("invalid_go", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		writeFile(t, filepath.Join(dir, "broken.go"), "package broken\nvar Invalid = doesNotExist\n")
		output := runCommand(t, dir, false, binary, "./...")
		if !strings.Contains(output, "doesNotExist") {
			t.Fatalf("missing Go compiler error: %s", output)
		}
	})
}

// consumerModule copies fixtures into an independent module that resolves the library to this checkout.
func consumerModule(t *testing.T, root, fixture string) string {
	t.Helper()
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/consumer\n\ngo 1.27.1\n\n"+
		"require github.com/lucasavila00/immutago v0.0.0\n\n"+
		"replace github.com/lucasavila00/immutago => %s\n", strconv.Quote(filepath.ToSlash(root)))
	writeFile(t, filepath.Join(dir, "go.mod"), module)
	source := filepath.Join(root, "tests", "testdata", fixture)
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(dir, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// runCommand runs a child process with workspace mode disabled and checks its exit status.
func runCommand(t *testing.T, dir string, success bool, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if (err == nil) != success {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return string(output)
}

// checkDiagnostics checks each fixture's want expression at its exact file and line.
func checkDiagnostics(t *testing.T, dir, output string) {
	t.Helper()
	wants := 0
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		count, err := checkFileDiagnostics(t, path, output)
		wants += count
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if wants == 0 {
		t.Fatal("fixture has no diagnostic expectations")
	}
}

// checkFileDiagnostics checks expected findings in one source file and returns the expectation count.
func checkFileDiagnostics(t *testing.T, path, output string) (int, error) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	wants := 0
	for index, line := range strings.Split(string(source), "\n") {
		_, expression, found := strings.Cut(line, "// want ")
		if !found {
			continue
		}
		wants++
		pattern, err := strconv.Unquote(strings.TrimSpace(expression))
		if err != nil {
			return wants, fmt.Errorf("%s:%d: bad expectation: %w", path, index+1, err)
		}
		location := regexp.QuoteMeta(filepath.Base(path) + ":" + strconv.Itoa(index+1) + ":")
		matched, err := regexp.MatchString(location+`[0-9]+: `+pattern, output)
		if err != nil {
			return wants, err
		}
		if !matched {
			t.Errorf("missing %s:%d: %s", path, index+1, pattern)
		}
	}
	return wants, nil
}

// writeFile writes a generated consumer file and reports errors through the test.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
