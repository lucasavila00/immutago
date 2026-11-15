// Command immutago-gen generates projections for Frozen records and interfaces.
package main

import (
	"fmt"
	"os"

	"github.com/lucasavila00/immutago/internal/projectiongen"
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
	content, err := projectiongen.Generate(".")
	if err != nil {
		return err
	}
	if len(content) == 0 {
		err := os.Remove(projectiongen.Filename)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(projectiongen.Filename, content, 0644)
}
