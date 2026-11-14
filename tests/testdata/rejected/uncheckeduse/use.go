package uncheckeduse

import (
	"example.com/consumer/unchecked"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Wrap wraps a type of another package.
func Wrap() im.Unchecked[unchecked.Table] { return im.Trust(&unchecked.Table{}) }

// Read reads a value of another package.
func Read(b *unchecked.Box) int {
	table := b.Table
	return len(table.Value().Rows)
}

// Expression names the method of another package's value.
var Expression = im.Unchecked[unchecked.Table].Value
