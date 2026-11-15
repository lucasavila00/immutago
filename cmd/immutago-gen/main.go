// Command immutago-gen generates typed views and cells for a package's Frozen records.
package main

import (
	"fmt"
	"os"

	"github.com/lucasavila00/immutago/internal/viewgen"
)

// main writes the generated source in the current package.
func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// generate loads the current package and replaces only its generator-owned output.
func generate() error {
	if len(os.Args) != 1 {
		return fmt.Errorf("run immutago-gen without arguments in the target package")
	}
	content, err := viewgen.Generate(".")
	if err != nil {
		return err
	}
	return os.WriteFile(viewgen.Filename, content, 0644)
}
