// Package readcost compares generated field reads with native pointer reads.
package readcost

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

// Record8KiB puts one copied field after a large array.
type Record8KiB struct {
	im.Frozen
	Padding [8192]byte
	Count   int
}

// Record64KiB checks whether read cost changes with the root size.
type Record64KiB struct {
	im.Frozen
	Padding [65536]byte
	Count   int
}

var record8 Record8KiBCell
var record64 Record64KiBCell
