package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Data struct {
	im.Frozen
	Name  string
	Names [2]string
	Next  *Data
}

var leaked string
var integerCell im.Cell[int]
var stringCell im.Cell[string]

func projection(projection im.Projection[*Data], scalar im.Projection[int]) {
	_ = im.ProjectionOf(&projection.Unsafe().Names)
	_ = &projection.Unsafe().Next
	projection.Unsafe().Name = "changed"
	raw := projection.Unsafe
	_ = raw
	_ = im.Projection[*Data].Unsafe(projection)
}

func identity(value string) string { return value }

func cellStatements() {
	go integerCell.Load().Unsafe()
	defer stringCell.Load().Unsafe()
	go identity(stringCell.Load().Unsafe())
	defer identity(stringCell.Load().Unsafe())
	go integerCell.Load()
	defer stringCell.Load()
	go stringCell.Store("text")
	defer stringCell.Store("text")
}
