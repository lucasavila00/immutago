package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScopeVariants keeps ordinary acceptance separate from checked failure measurements.
func TestScopeVariants(t *testing.T) {
	for _, library := range []string{"", "/checkout/immutago"} {
		dir := t.TempDir()
		if err := writeModule(dir, 1, library); err != nil {
			t.Fatal(err)
		}
		source, err := os.ReadFile(filepath.Join(dir, "p0_0", "p.go"))
		if err != nil {
			t.Fatal(err)
		}
		checked := strings.Contains(string(source), `"github.com/lucasavila00/immutago/pkg/im"`)
		if checked != (library != "") {
			t.Fatalf("library=%q: checked=%t", library, checked)
		}
		if got := strings.Count(string(source), "_ = unsafe.Pointer(&n)"); got != 50 {
			t.Fatalf("conversion count=%d, want 50", got)
		}
		_, err = os.Stat(filepath.Join(dir, "consumer", "consumer.go"))
		if library != "" && err != nil || library == "" && !os.IsNotExist(err) {
			t.Fatalf("library=%q: consumer status=%v", library, err)
		}
	}
}
