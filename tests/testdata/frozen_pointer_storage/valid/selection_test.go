package valid

import (
	"reflect"
	"testing"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var childPointer *api.Child

func TestSelectedChildren(t *testing.T) {
	cell := new(api.SelectionCell)
	if !cell.Load().IsNil() || cell.Load().Unsafe() != nil {
		t.Fatal("empty projection changed nil")
	}
	items := im.FreezeSlice([]api.Child{{Name: "list", N: 4}})
	table := im.FreezeMap(map[string]api.Child{"key": {Name: "map", N: 5}})
	child := &api.Child{Name: "pointer", N: 2}
	cell.Store(api.Selection{
		Child: api.Child{Name: "field", N: 1}, Pointer: child,
		Array: [2]api.Child{{Name: "array", N: 3}}, List: items, Table: table,
	})
	selected := cell.Load()
	root := selected.Unsafe()
	field := selected.Child().Unsafe()
	if reflect.ValueOf(field).Pointer() != reflect.ValueOf(root).Elem().FieldByName("Child").Addr().Pointer() {
		t.Fatal("field read copied the child")
	}
	if selected.Pointer().Unsafe() != child || selected.Array().At(0).Unsafe().N != 3 {
		t.Fatal("child read changed the selection")
	}
	list := selected.List()
	first := api.NewChildProjection(list.AtPointer(0)).Unsafe()
	if first != api.NewChildProjection(list.AtPointer(0)).Unsafe() || first.N != 4 {
		t.Fatal("slice read copied or changed its element")
	}
	value, found := selected.Table().Get("key")
	if !found {
		t.Fatal("missing map value")
	}
	copy := api.NewChildProjection(im.CopyRecord(value)).Unsafe()
	other := api.NewChildProjection(im.CopyRecord(value)).Unsafe()
	if copy == other || copy.N != 5 || copy.Name != value.Unsafe().Name {
		t.Fatal("map root copy changed its value or reused the root")
	}
	cell.Store(api.Selection{})
	if selected.Unsafe() != root || field.N != 1 || root == cell.Load().Unsafe() {
		t.Fatal("escape loaded another Cell version")
	}
	if got := testing.AllocsPerRun(100, func() { childPointer = root.Pointer }); got != 0 {
		t.Fatalf("pointer field allocated %g times", got)
	}
	if got := testing.AllocsPerRun(100, func() {
		childPointer = cell.Load().Child().Unsafe()
	}); got != 0 {
		t.Fatalf("inline child escape allocated %g times", got)
	}
}

func TestRawCellPayloads(t *testing.T) {
	cells := new(api.Payloads)
	if im.Value(cells.Number.Load()) != 0 || cells.Text.Load().Unsafe() != "" || cells.Pointer.Load().Unsafe() != nil {
		t.Fatal("empty raw Cell changed its zero value")
	}
	if cells.Record.Load().Unsafe().N != 0 || cells.Array.Load().Unsafe() != [2]int{} || cells.Copied.Load().Unsafe().N != 0 {
		t.Fatal("empty value Cell changed its zero value")
	}
	if cells.String.Load().Unsafe().Len() != 0 || cells.Slice.Load().Unsafe().Len() != 0 || cells.Map.Load().Unsafe().Len() != 0 {
		t.Fatal("empty collection Cell changed its zero value")
	}
	cells.Number.Store(1)
	cells.Text.Store("text")
	cells.String.Store(im.FreezeString("frozen"))
	slice := im.FreezeSlice([]int{2})
	cells.Slice.Store(slice)
	mapping := im.FreezeMap(map[int]int{3: 4})
	cells.Map.Store(mapping)
	child := &api.Child{N: 5}
	cells.Pointer.Store(child)
	cells.Record.Store(api.Child{N: 6})
	cells.Array.Store([2]int{7, 8})
	cells.Copied.Store(api.Score{N: 9})
	if im.Value(cells.Number.Load()) != 1 || cells.Text.Load().Unsafe() != "text" || cells.Pointer.Load().Unsafe() != child {
		t.Fatal("raw Cell changed its stored value")
	}
	value, found := cells.Map.Load().Unsafe().Get(3)
	if !found || value != 4 || cells.Slice.Load().Unsafe().At(0) != 2 || cells.String.Load().Unsafe().Unsafe() != "frozen" {
		t.Fatal("raw collection Cell changed its entries")
	}
	if cells.Record.Load().Unsafe().N != 6 || im.Value(cells.Array.Load()) != [2]int{7, 8} || im.Value(cells.Copied.Load()).N != 9 {
		t.Fatal("raw value Cell changed its fields")
	}
}
