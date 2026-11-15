package accepted

import (
	"fmt"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Alias = im.View[api.Model]
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

func read(view Alias, index int) api.Result {
	name := im.String(im.Select(view, func(m api.Model) im.FrozenString { return m.Name }))
	native := im.Text(im.Select(view, func(m api.Model) string { return m.Native }))
	_ = native.Clone()
	point := im.Copy(im.Select(view, func(m api.Model) api.Point { return m.Points[index] }))
	storedPoint = point
	child := im.Select(view, func(m api.Model) *api.Entry { return m.Child })
	if !im.IsNil(child) {
		_ = im.String(im.Select(child, func(e *api.Entry) im.FrozenString { return e.Name })).Len()
	}
	names := im.Slice(im.Select(view, func(m api.Model) im.FrozenSlice[im.FrozenString] { return m.Names }))
	for _, value := range names.All() {
		_ = im.String(value).Clone()
	}
	entries := im.Map(im.Select(view, func(m api.Model) im.FrozenMap[string, api.Entry] { return m.Entries }))
	entry, ok := entries.Get("name")
	score := 0
	if ok {
		score = im.Copy(im.Select(entry, func(e api.Entry) int { return e.Score }))
	}
	for key, value := range entries.All() {
		_ = im.Text(key).Len()
		_ = im.String(im.Select(value, func(e api.Entry) im.FrozenString { return e.Name })).At(0)
	}
	byName := im.Map(im.Select(view, func(m api.Model) im.FrozenMap[im.FrozenString, int] { return m.ByName }))
	for key, value := range byName.All() {
		_ = im.String(key).Clone()
		_ = im.Copy(value)
	}
	return api.Result{Name: name, Score: score}
}

func ordinaryCopies(slice im.SliceView[int], mapping im.MapView[int, int]) {
	values := slice.Clone()
	values = append(values, 1)
	frozenValues := im.FreezeSlice(values)
	_ = frozenValues
	entries := mapping.Clone()
	entries[1] = 2
	frozenEntries := im.FreezeMap(entries)
	_ = frozenEntries
	label := im.Text(im.ViewOf(Label("label")))
	_ = label.Clone()
	owned := api.Shared()
	_ = owned.Native[:1]
}

func readZero() {
	var view im.View[api.Model]
	var reply api.Result
	var text im.StringView
	var local [0]im.View[string]
	_ = read(view, 0)
	_ = reply
	_ = text.Len()
	_ = local
}

func typedWrappers(h *api.Holder) {
	h.Current.Store(api.Model{})
	view := h.Current.Load()
	_ = view.Name().EqualString("name")
	typedCell.Store(api.Model{})
	_ = typedCell.Load().Name().Len()
	_ = fmt.Sprintf("%s", view.Name().Unsafe())
	_ = fmt.Sprintf("%s", storedString.Unsafe())
}

func localBuilders(value im.StringView) {
	values := []im.StringView{value}
	values = append(values, value)
	for _, item := range values {
		_ = item.Len()
	}
	byName := map[string]im.StringView{"name": value}
	for name, item := range byName {
		_ = name
		_ = item.Len()
	}
	byView := map[im.StringView]int{value: 1}
	for key, count := range byView {
		_ = key.Len()
		_ = count
	}
}
