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
	_ = im.ProjectionOf(&projection.Unchecked().Names)
	_ = &projection.Unchecked().Next
	projection.Unchecked().Name = "changed"
	raw := projection.Unchecked
	_ = raw
	_ = im.Projection[*Data].Unchecked(projection)
}

func identity(value string) string { return value }

func cellStatements() {
	go integerCell.Load().Unchecked()
	defer stringCell.Load().Unchecked()
	go identity(stringCell.Load().Unchecked())
	defer identity(stringCell.Load().Unchecked())
	go integerCell.Load()
	defer stringCell.Load()
	go stringCell.Store("text")
	defer stringCell.Store("text")
}
