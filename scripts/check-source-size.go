// Command check-source-size checks the size of tracked Go source files.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const maxSourceLines = 700

// main checks tracked source files and exits with failure if a file exceeds the size limit.
func main() {
	files, err := exec.Command("git", "ls-files", "-z", "--", "*.go").Output()
	if err == nil {
		err = checkSourceFiles(strings.Split(string(files), "\x00"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// checkSourceFiles counts source lines and returns each file that exceeds the limit.
func checkSourceFiles(paths []string) error {
	var failures []error
	for _, path := range paths {
		if path == "" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := bytes.Count(content, []byte{'\n'})
		if len(content) > 0 && content[len(content)-1] != '\n' {
			lines++
		}
		if lines > maxSourceLines {
			failures = append(failures, fmt.Errorf("%s: %d lines; maximum is %d", path, lines, maxSourceLines))
		}
	}
	return errors.Join(failures...)
}
