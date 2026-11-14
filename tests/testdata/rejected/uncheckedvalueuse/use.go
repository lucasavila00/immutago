package uncheckedvalueuse

import (
	"example.com/consumer/uncheckedvalue"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Wrap wraps a type of another package.
func Wrap() im.UncheckedValue[uncheckedvalue.Table] { return im.Trust(&uncheckedvalue.Table{}) }

// Read reads a value of another package.
func Read(b *uncheckedvalue.Box) int {
	table := b.Table
	return len(im.Unwrap(table).Rows)
}

// Reader names the read function for another package's value.
var Reader = im.Unwrap[uncheckedvalue.Table]
