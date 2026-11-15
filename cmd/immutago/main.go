// Command immutago checks immutable data forms and service method results.
package main

import (
	"github.com/lucasavila00/immutago/pkg/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// main runs the analyzer with the standard Go analysis command interface.
func main() {
	singlechecker.Main(analyzer.Analyzer)
}
