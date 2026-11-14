package opaqueuse

import (
	"example.com/consumer/opaque"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Wrap wraps a type of another package.
func Wrap() im.Opaque[opaque.Table] { return im.Seal(&opaque.Table{}) }

// Read reads a value of another package.
func Read(b *opaque.Box) int {
	table := b.Table
	return len(table.Value().Rows)
}

// Expression names the method of another package's value.
var Expression = im.Opaque[opaque.Table].Value
