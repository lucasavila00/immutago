package tests

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// packageOutputs assigns complete messages by source path, including generated files.
func packageOutputs(directory, module, output string) (map[string]string, error) {
	files, err := fixtureSources(directory, module)
	if err != nil {
		return nil, err
	}
	messages := make(map[string][]string)
	for _, pkg := range files {
		messages[pkg] = nil
	}
	location := regexp.MustCompile(`^(.+?\.go):[0-9]+:[0-9]+: `)
	for _, record := range splitDiagnosticRecords(output, false) {
		match := location.FindStringSubmatch(record)
		if match == nil {
			return nil, fmt.Errorf("unmatched tool output: %s", record)
		}
		var owner string
		longest := -1
		for path, pkg := range files {
			if (match[1] == path || strings.HasSuffix(match[1], "/"+path)) && len(path) > longest {
				owner, longest = pkg, len(path)
			}
		}
		if owner == "" {
			return nil, fmt.Errorf("unmatched tool output: %s", record)
		}
		messages[owner] = append(messages[owner], strings.TrimSuffix(normalizeOutput(record), "\n"))
	}
	result := make(map[string]string, len(messages))
	for pkg, records := range messages {
		sort.Strings(records)
		result[pkg] = strings.Join(records, "\n")
		if len(records) > 0 {
			result[pkg] += "\n"
		}
	}
	return result, nil
}

// fixtureSources indexes each source by its path within the consumer module.
func fixtureSources(directory, module string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		relative, err := filepath.Rel(module, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = filepath.Dir(path)
		return nil
	})
	return files, err
}

// TestPackageOutputs distinguishes repeated generated filenames and retains their errors.
func TestPackageOutputs(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first", "second", "empty"} {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "zz_immutago.gen.go"), "package fixture\n")
	}
	output := "/tmp/consumer/second/zz_immutago.gen.go:5:1: second error\n  fix: second fix\n" +
		"C:\\consumer\\first\\zz_immutago.gen.go:3:1: first error\r\n"
	packages, err := packageOutputs(root, root, output)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"first":  "zz_immutago.gen.go:3:1: first error\n",
		"second": "zz_immutago.gen.go:5:1: second error\n  fix: second fix\n",
		"empty":  "",
	} {
		if got := packages[filepath.Join(root, name)]; got != want {
			t.Errorf("%s output = %q; want %q", name, got, want)
		}
	}
	if _, err := packageOutputs(root, root, "unexpected tool output\n"); err == nil {
		t.Fatal("unlocated output was discarded")
	}
}
