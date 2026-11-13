// Command immutago checks marked Go structs and immutable collection types.
package main

import (
	"github.com/lucasavila00/immutago/pkg/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// main runs the analyzer with the standard Go analysis command interface.
func main() {
	singlechecker.Main(analyzer.Analyzer)
}
