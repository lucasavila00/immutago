package analyzer

import (
	"strings"
	"testing"
)

// TestBodylessSignatures checks the native implementation boundary by resolved type.
func TestBodylessSignatures(t *testing.T) {
	const prefix = `type Value struct { im.Frozen; N int }
type Local struct { im.Local; N int }
type Plain struct { N int }
type Alias = *Value
`
	for _, source := range []string{
		`func cpuID() uint64`,
		`func checksum([]byte) uint64`,
		`func update(*Plain, map[string]int) error`,
		`func (*Plain) Read() int`,
		`type Count = uint64; func read() Count`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, prefix+source); got != "" {
				t.Fatalf("ordinary signature rejected: %s", got)
			}
		})
	}
	for _, source := range []string{
		`func read(*Value) int`,
		`func read(Alias) int`,
		`func makeValue() Value`,
		`func (*Value) Read() int`,
		`func read(*Local) int`,
		`func read(im.FrozenSlice[int]) int`,
		`func read(im.Cell[Local]) int`,
		`func read(callback func(*Value))`,
		`type Contract[T interface{ ~int | im.FrozenSlice[int] }] = int; func read(Contract[int])`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, prefix+source); !strings.Contains(got, "unsupported function without a Go body") {
				t.Fatalf("protected signature accepted: %s", got)
			}
		})
	}
}
