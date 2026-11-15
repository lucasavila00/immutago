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
	_ = projection.Read()
	_ = projection.Read().Name
	_ = projection.Read().Name[:]
	_ = identity(projection.Read().Name)
	leaked = projection.Read().Name
	_ = im.ProjectionOf(projection.Read().Next)
	_ = im.ProjectionOf(&projection.Read().Names)
	_ = &projection.Read().Next
	projection.Read().Name = "changed"
	raw := projection.Read
	_ = raw
	_ = im.Projection[*Data].Read(projection)
	_ = scalar.Read()
}

func identity(value string) string { return value }

func cellStatements() {
	go integerCell.LoadShared()
	defer stringCell.LoadShared()
	go identity(stringCell.LoadShared())
	defer identity(stringCell.LoadShared())
	go integerCell.Load()
	defer stringCell.Load()
	go stringCell.Store("text")
	defer stringCell.Store("text")
}
