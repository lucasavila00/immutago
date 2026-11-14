// Package tests exercises the built analyzer against independent consumer modules.
package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var updateSnapshots = flag.Bool("update-snapshots", false, "update compiler and analyzer diagnostic snapshots")

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
	for _, group := range []string{"review_convergence", "documentation", "owner", "process", "uncheckedvalue", "uncheckedvalueuse"} {
		t.Run(group, func(t *testing.T) {
			for _, fixture := range []string{"accepted", "rejected"} {
				t.Run(fixture, func(t *testing.T) {
					dir := consumerModule(t, root, fixture)
					output := runCommand(t, dir, fixture == "accepted", binary, "./"+group)
					checkFixtureSnapshots(t, root, fixture+"/"+group, output)
				})
			}
		})
	}
	t.Run("owner_cross_package", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		output := runCommand(t, dir, true, binary, "./ownercross")
		checkFixtureSnapshots(t, root, "accepted/ownercross", output)
	})
	t.Run("process_cross_package", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		output := runCommand(t, dir, true, binary, "./processcross")
		checkFixtureSnapshots(t, root, "accepted/processcross", output)
	})
	t.Run("accepted", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		output := runCommand(t, dir, true, binary, "./...")
		checkFixtureSnapshots(t, root, "accepted", output)
	})
	t.Run("freeze_runtime", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		output := runCommand(t, dir, true, "go", "run", "./freezeruntime")
		checkSnapshot(t, filepath.Join(root, "tests/testdata/accepted/freezeruntime/stdout.txt"), output)
	})
	t.Run("invalid_dependencies", func(t *testing.T) {
		checkInvalidDependencies(t, root, binary)
	})
	t.Run("ordinary_callback_library", func(t *testing.T) {
		dir := consumerModule(t, root, "rejected")
		output := runCommand(t, dir, true, binary, "./callback_api")
		checkFixtureSnapshots(t, root, "rejected/callback_api", output)
	})
	t.Run("rejected", func(t *testing.T) {
		dir := consumerModule(t, root, "rejected")
		output := runCommand(t, dir, false, binary, "./...")
		checkFixtureSnapshots(t, root, "rejected", output)
	})
	t.Run("vet", func(t *testing.T) {
		dir := consumerModule(t, root, "accepted")
		output := runCommand(t, dir, true, "go", "vet", "-vettool="+binary, "./...")
		checkFixtureSnapshots(t, root, "accepted", output)
		dir = consumerModule(t, root, "rejected")
		output = runCommand(t, dir, false, "go", "vet", "-vettool="+binary, "./...")
		checkFixtureSnapshots(t, root, "rejected", output)
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
		source, err := os.ReadFile(filepath.Join(root, "tests/testdata/compiler_error/broken.go"))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "broken.go"), string(source))
		output := runCommand(t, dir, false, binary, "./...")
		checkSnapshot(t, filepath.Join(root, "tests/testdata/compiler_error/broken.txt"), normalizeOutput(output))
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

// checkFixtureSnapshots checks each source against its sibling text snapshot.
func checkFixtureSnapshots(t *testing.T, root, fixture, output string) {
	t.Helper()
	directory := filepath.Join(root, "tests", "testdata", fixture)
	lines := diagnosticRecords(output)
	used := make(map[int]bool)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		var diagnostics []string
		for index, line := range lines {
			if strings.HasPrefix(line, filepath.Base(path)+":") {
				diagnostics = append(diagnostics, line)
				used[index] = true
			}
		}
		actual := ""
		if len(diagnostics) > 0 {
			actual = strings.Join(diagnostics, "\n") + "\n"
		}
		checkSnapshot(t, strings.TrimSuffix(path, ".go")+".txt", actual)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for index, line := range lines {
		if line != "" && !used[index] {
			t.Errorf("unmatched tool output: %s", line)
		}
	}
}

// checkSnapshot compares exact tool output with a checked-in sibling diagnostic snapshot.
func checkSnapshot(t *testing.T, path, actual string) {
	t.Helper()
	if *updateSnapshots {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, path, actual)
		return
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v; use -update-snapshots to create it", err)
	}
	saved := strings.ReplaceAll(string(expected), "\r\n", "\n")
	if actual != saved {
		t.Errorf("snapshot %s differs; inspect output before using -update-snapshots\nSAVED:\n%s\nACTUAL:\n%s",
			path, saved, actual)
	}
}

// normalizeOutput joins stable records without changing the order of their context lines.
func normalizeOutput(output string) string {
	records := diagnosticRecords(output)
	if len(records) == 0 {
		return ""
	}
	return strings.Join(records, "\n") + "\n"
}

// diagnosticRecords removes machine paths and sorts complete messages with their indented context.
func diagnosticRecords(output string) []string {
	location := regexp.MustCompile(`^.*?\.go:[0-9]+:[0-9]+: `)
	output = strings.ReplaceAll(output, "\\", "/")
	output = strings.ReplaceAll(output, "\r\n", "\n")
	var records []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" || strings.HasPrefix(line, "# ") {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmed)]
		prefix := location.FindString(trimmed)
		related := prefix != "" && strings.HasPrefix(strings.TrimPrefix(trimmed, prefix), "\t")
		if indent == "" && prefix != "" {
			line = filepath.Base(prefix) + strings.TrimPrefix(trimmed, prefix)
		}
		if (indent != "" || related) && len(records) > 0 {
			records[len(records)-1] += "\n" + line
		} else {
			records = append(records, line)
		}
	}
	sort.Strings(records)
	return records
}

// writeFile writes a generated consumer file and reports errors through the test.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// checkInvalidDependencies selects only consumers so driver-hidden dependency errors must propagate through facts.
func checkInvalidDependencies(t *testing.T, root, binary string) {
	t.Helper()
	dir := consumerModule(t, root, "dependency_invalid")
	output := runCommand(t, dir, false, binary, "./...")
	checkFixtureSnapshots(t, root, "dependency_invalid", output)
	cases := []string{
		"shape_consumer", "unsafe_consumer", "borrow_consumer",
		"wrapped_shape_consumer", "wrapped_borrow_consumer",
		"freeze_consumer", "wrapped_freeze_consumer",
		"noref_consumer", "wrapped_noref_consumer",
		"diamond_consumer", "diamond_direct_consumer",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			commands := [][]string{
				{binary, "./" + name},
				{"go", "vet", "-vettool=" + binary, "./" + name},
			}
			for _, command := range commands {
				output := runCommand(t, dir, false, command[0], command[1:]...)
				checkFixtureSnapshots(t, root, "dependency_invalid/"+name, output)
			}
		})
	}
}

// TestSnapshotLineEndings accepts CRLF snapshots from a Windows checkout without changing diagnostics.
func TestSnapshotLineEndings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diagnostic.txt")
	writeFile(t, path, "source.go:3:4: diagnostic\r\n  fix: use a copy\r\n")
	checkSnapshot(t, path, "source.go:3:4: diagnostic\n  fix: use a copy\n")
}
