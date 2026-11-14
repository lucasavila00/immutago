// Command dependency-stress writes the layered package graph from issues 20 and 28.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// main selects the graph size and an optional relevant consumer.
func main() {
	scale := flag.Int("scale", 1, "layers and width are each four times this value")
	out := flag.String("out", "dependency-stress", "output module directory")
	library := flag.String("library", "", "immutago checkout; check generated packages and add a consumer when set")
	flag.Parse()
	if err := writeModule(*out, *scale, *library); err != nil {
		log.Fatal(err)
	}
}

// writeModule creates a dense graph with 50 unsafe pointer conversions per package.
func writeModule(out string, scale int, library string) error {
	module := "module dependency-stress.example\n\ngo 1.27.1\n"
	if library != "" {
		module += "\nrequire github.com/lucasavila00/immutago v0.0.0\n" +
			"replace github.com/lucasavila00/immutago => " + strconv.Quote(filepath.ToSlash(library)) + "\n"
	}
	if err := writeFile(out, "go.mod", module); err != nil {
		return err
	}
	size := 4 * scale
	for layer := range size {
		for column := range size {
			name := fmt.Sprintf("p%d_%d", layer, column)
			if err := writeFile(out, name+"/p.go", packageSource(name, layer, size, library != "")); err != nil {
				return err
			}
		}
	}
	if library != "" {
		return writeConsumer(out, size)
	}
	return nil
}

// packageSource imports every package in the previous layer and uses 50 unsafe pointers.
func packageSource(name string, layer, width int, checked bool) string {
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\nimport (\n\t\"unsafe\"\n", name)
	if checked {
		source.WriteString("\t_ \"github.com/lucasavila00/immutago/pkg/im\"\n")
	}
	if layer > 0 {
		for column := range width {
			fmt.Fprintf(&source, "\tp%d_%d \"dependency-stress.example/p%d_%d\"\n", layer-1, column, layer-1, column)
		}
	}
	source.WriteString(")\n\nfunc Size() int {\n\tn := int(unsafe.Sizeof(0))\n")
	for range 50 {
		source.WriteString("\t_ = unsafe.Pointer(&n)\n")
	}
	if layer > 0 {
		for column := range width {
			fmt.Fprintf(&source, "\tn += p%d_%d.Size()\n", layer-1, column)
		}
	}
	source.WriteString("\treturn n\n}\n")
	return source.String()
}

// writeConsumer adds a silent wrapper and a consumer that must receive an original failure.
func writeConsumer(out string, size int) error {
	var source strings.Builder
	source.WriteString("package wrapper\n\nimport (\n")
	for column := range size {
		fmt.Fprintf(&source, "\t_ \"dependency-stress.example/p%d_%d\"\n", size-1, column)
	}
	source.WriteString(")\n")
	if err := writeFile(out, "wrapper/wrapper.go", source.String()); err != nil {
		return err
	}
	return writeFile(out, "consumer/consumer.go", "package consumer\n\nimport (\n"+
		"\t_ \"dependency-stress.example/wrapper\"\n"+
		"\t_ \"github.com/lucasavila00/immutago/pkg/im\"\n)\n")
}

// writeFile writes one generated module file.
func writeFile(out, name, content string) error {
	path := filepath.Join(out, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0600)
}
