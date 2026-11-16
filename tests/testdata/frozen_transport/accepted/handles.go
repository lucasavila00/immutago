package accepted

import (
	"context"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Text = im.FrozenString
type Handles struct {
	Title  Text
	Names  [2]Text
	Points im.FrozenSlice[*api.Count]
	Index  im.FrozenMap[string, *api.Count]
	Reader api.Reader
	Calls  int
}
type HandleAlias = Handles
type NestedHandles struct{ Items [2]HandleAlias }
type HandleBox[T any] struct{ Value T }
type EmbeddedText struct{ Text }
type EmbeddedTextPointer struct{ *Text }
type EmbeddedPoints struct{ im.FrozenSlice[*api.Count] }

func (h *Handles) Rename(title Text) { h.Title = title; h.Calls++ }
func (h Handles) With(title Text) Handles {
	h.Title = title
	return h
}
func (b *HandleBox[T]) Set(value T)          { b.Value = value }
func handlePair(h Handles) (Handles, Text)   { return h, h.Title }
func handleReaders(values ...api.Reader) int { return len(values) }

func HandleSlots() int {
	text := im.FreezeString("old")
	points := im.FreezeSlice([]*api.Count{{N: 7}})
	index := im.FreezeMap(map[string]*api.Count{"current": points.At(0)})
	h := Handles{Title: text, Points: points, Index: index, Reader: api.Count{N: 2}}
	copy := h
	h.Rename(im.FreezeString("new"))
	slot := &h.Title
	*slot = im.FreezeString("next")
	h.Names[0] = h.Title
	nested := NestedHandles{}
	nested.Items[0] = h
	box := HandleBox[Text]{Value: text}
	box.Set(h.Title)
	other, title := handlePair(h)
	other.Title, h.Title = h.Title, title
	values := []Text{text}
	alias := values
	alias[0] = h.Title
	values = append(values, alias...)
	var i int
	var value Text
	for i, value = range values {
		_ = i
		_ = value.Len()
	}
	entries := map[Text]Handles{text: h}
	entries[text] = other
	ready := make(chan api.Reader, 1)
	ready <- api.Count{N: 3}
	reader, ok := <-ready
	if !ok {
		return -1
	}
	select {
	case ready <- reader:
	default:
	}
	select {
	case <-ready:
	default:
	}
	ready <- reader
	<-ready
	close(ready)
	for item := range ready {
		_ = item.Number()
	}
	read := func() int { return copy.Title.Len() }
	length := EmbeddedText{Text: text}.Len
	_ = EmbeddedTextPointer{Text: &text}.Len()
	_ = EmbeddedPoints{FrozenSlice: points}.Len()
	_ = identity(&h)
	_ = identity(values)
	_ = handleReaders(h.Reader, api.Count{})
	return read() + length() + box.Value.Len() + h.Points.At(0).N + reader.Number()
}

type HandleWorker struct {
	supervise.Service[Handles]
	State Handles
}

func (w *HandleWorker) Run(ctx context.Context) { w.State = w.Args(); <-ctx.Done() }
func (w *HandleWorker) NameLen() int            { return w.State.Title.Len() }
func (w *HandleWorker) Rename(text Text)        { w.State.Rename(text) }

func startHandleWorker(root *supervise.Group, args Handles) (supervise.Ref[HandleWorker], error) {
	return supervise.Start[HandleWorker](root, args)
}

func copiedRangeIndex(values []Text, local im.StringProjection) {
	index := local.Len()
	var indexes [1]int
	var value Text
	for indexes[index], value = range values {
	}
	_ = value
}
