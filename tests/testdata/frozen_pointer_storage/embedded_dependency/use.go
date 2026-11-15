package embedded_dependency

import (
	"example.com/consumer/embedded"
	"github.com/lucasavila00/immutago/pkg/im"
)

var Cell im.Cell[embedded.Numeric]

func Read() *embedded.Numeric { return Cell.LoadPointer().Unsafe() }
