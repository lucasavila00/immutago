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
	return len(im.Unseal(table).Rows)
}

// Reader names the read function for another package's value.
var Reader = im.Unseal[opaque.Table]
