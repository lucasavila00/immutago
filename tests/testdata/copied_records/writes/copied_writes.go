package writes

import (
	"encoding/json"

	"example.com/consumer/api"
	"example.com/consumer/frozen"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Result struct {
	im.Local
	Point api.Alias
	Rows  [2]api.Summary
}

func Writes(point api.Point, stored frozen.Snapshot, local Result) {
	stored.Point = point
	stored.Point.X++
	stored.Points[0].Y = 2
	stored.Summary.Points[0][0].X = 3
	local.Point.X = 3
	local.Rows[0].Points[0][0].Y++
}

func Addresses(stored *frozen.Snapshot, local *Result) {
	_ = &stored.Point
	_ = &stored.Point.X
	_ = &stored.Points[0]
	_ = &stored.Summary.Points[0][0]
	_ = &local.Point
	_ = &local.Rows[0].Points[0][0].Y
}

func ImplicitAddresses(stored *frozen.Snapshot, local *Result) {
	stored.Point.Set(2)
	stored.Summary.Points[0][0].Set(3)
	local.Point.Set(2)
	local.Rows[0].Points[0][0].Set(3)
	method := stored.Point.Set
	method(2)
	(*api.Point).Set(&stored.Point, 3)
	_ = json.Unmarshal([]byte(`{"X":2}`), &stored.Point)
}

func SelectedPointer(stored *frozen.Snapshot) *api.Alias {
	return &stored.Point
}

func BuilderAliases() {
	values := []api.Point{{}}
	alias := values
	_ = alias
	_ = im.FreezeSlice(values)
}

func BuilderMapAlias() {
	values := map[int]api.Point{0: {}}
	alias := values
	_ = alias
	_ = im.FreezeMap(values)
}

func BuilderAddresses() {
	values := []api.Point{{}}
	pointer := &values[0]
	pointer.Set(2)
	_ = im.FreezeSlice(values)
}

func BuilderImplicitAddress() {
	values := []api.Point{{}}
	values[0].Set(2)
	_ = im.FreezeSlice(values)
}

func Consumed() {
	values := []api.Point{{}}
	_ = im.FreezeSlice(values)
	values[0].X++
}

func ProjectionPointers(value im.Projection[api.Point], values im.SliceProjection[api.Point]) {
	_ = im.CopyRecord(value)
	_ = values.AtPointer(0)
	_ = value.Read()
	_ = values.At(0).Read()
}

var savedField *int

func saveField(pointer *int) { savedField = pointer }

func SaveFrozenField(stored *frozen.Snapshot) {
	saveField(&stored.Point.X)
	*savedField = 9
}

func SaveLocalField(local *Result) {
	savedField = &local.Rows[0].Points[0][0].Y
	*savedField = 9
}

func SaveBuilderField() {
	values := []api.Point{{X: 1}}
	saveField(&values[0].X)
	frozen := im.FreezeSlice(values)
	*savedField = 9
	_ = frozen.At(0)
}

func saveBuilderField(values []api.Point) { savedField = &values[0].X }

func PassBuilderToPointerSaver() {
	values := []api.Point{{X: 1}}
	saveBuilderField(values)
	frozen := im.FreezeSlice(values)
	*savedField = 9
	_ = frozen.At(0)
}

func SaveNestedBuilderField() {
	values := []api.Summary{{}}
	savedField = &values[0].Points[1][0].X
	frozen := im.FreezeSlice(values)
	*savedField = 9
	_ = frozen.At(0)
}
