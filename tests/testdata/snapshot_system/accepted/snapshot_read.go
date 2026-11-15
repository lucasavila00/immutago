package accepted

import (
	"fmt"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Alias = im.Projection[*api.Model]
type Label string

var storedPoint api.Point
var storedString im.FrozenString
var typedCell api.ModelCell

func build(text string) {
	name := im.FreezeString(text)
	_ = text
	names := im.FreezeSlice([]im.FrozenString{name})
	entries := im.FreezeMap(map[string]api.Entry{"name": {Name: name, Score: 2}})
	byName := im.FreezeMap(map[im.FrozenString]int{name: 2})
	api.Publish(api.Model{Name: name, Native: text, Names: names, Entries: entries, ByName: byName})
	storedString = name
}

func read(projection Alias, index int) api.Result {
	model := api.NewModelProjection(projection)
	name := model.Name()
	native := model.Native()
	_ = native.Clone()
	point := model.Points().At(index)
	storedPoint = api.Point{X: point.X(), Y: point.Y()}
	child := model.Child()
	if !child.IsNil() {
		_ = child.Name().Len()
	}
	names := model.Names()
	for _, value := range names.All() {
		_ = im.ProjectedFrozenString(value).Clone()
	}
	entries := model.Entries()
	entry, ok := entries.Get("name")
	score := 0
	if ok {
		score = api.NewEntryProjection(im.CopyRecord(entry)).Score()
	}
	for key, value := range entries.All() {
		_ = im.ProjectedString(key).Len()
		_ = api.NewEntryProjection(im.CopyRecord(value)).Name().At(0)
	}
	byName := model.ByName()
	for key, value := range byName.All() {
		_ = im.ProjectedFrozenString(key).Clone()
		_ = im.Copy(value)
	}
	return api.Result{Name: name, Score: score}
}

func ordinaryCopies(slice im.SliceProjection[int], mapping im.MapProjection[int, int]) {
	values := slice.Clone()
	values = append(values, 1)
	frozenValues := im.FreezeSlice(values)
	_ = frozenValues
	entries := mapping.Clone()
	entries[1] = 2
	frozenEntries := im.FreezeMap(entries)
	_ = frozenEntries
	label := im.ProjectString(Label("label"))
	_ = label.Clone()
	owned := api.LoadShared()
	_ = owned.Native[:1]
}

func readZero() {
	var projection im.Projection[*api.Model]
	var reply api.Result
	var text im.StringProjection
	var local [0]im.Projection[string]
	_ = read(projection, 0)
	_ = reply
	_ = text.Len()
	_ = local
}

func typedWrappers(h *api.Holder) {
	h.Current.Store(api.Model{})
	projection := h.Current.Load()
	_ = projection.Name().EqualString("name")
	typedCell.Store(api.Model{})
	_ = typedCell.Load().Name().Len()
	_ = fmt.Sprintf("%s", projection.Name().Unsafe())
	_ = fmt.Sprintf("%s", storedString.Unsafe())
}

func localBuilders(value im.StringProjection) {
	values := []im.StringProjection{value}
	values = append(values, value)
	for _, item := range values {
		_ = item.Len()
	}
	byName := map[string]im.StringProjection{"name": value}
	for name, item := range byName {
		_ = name
		_ = item.Len()
	}
	byProjection := map[im.StringProjection]int{value: 1}
	for key, count := range byProjection {
		_ = key.Len()
		_ = count
	}
}

func recordEntries(values im.MapProjection[string, api.Entry], pointers im.MapProjection[*api.Entry, *api.Entry],
	sequence im.SliceProjection[api.Entry]) int {
	value, ok := values.Get("name")
	if ok {
		_ = api.NewEntryProjection(im.CopyRecord(value)).Name().Len()
	}
	for key, entry := range pointers.All() {
		_ = api.NewEntryProjection(key).Score()
		_ = api.NewEntryProjection(entry).Name().EqualString("name")
	}
	return api.NewEntryProjection(sequence.AtPointer(0)).Score()
}
