package localviews

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Text = im.FrozenByValueString

type Reply struct {
	im.ByValue
	Text   Text
	Counts im.FrozenByValueSlice[int]
	Scores im.FrozenByValueMap[im.FrozenString, int]
}

// TextValue returns a local view through a concrete typed result.
func TextValue(text string) Text { return im.FreezeByValueString(text) }

// Pass preserves the local rule on the complete containing record.
func Pass(reply Reply) Reply { return reply }

// First reads one nested local view without making a shared reference.
func First(values im.FrozenByValueSlice[Text]) Text { return values.At(0) }

// Forms checks local bindings, fixed reads, direct ranges, and independent clones.
func Forms(native string) {
	var text Text
	text = Text{}
	text = TextValue(native)
	other := text
	_ = other == text
	_ = text.Len()
	_ = text.At(0)
	copy := text.Clone()
	_ = copy[:1]
	_ = native[:1]
	pair := [2]Text{text, other}
	_ = pair[0].Clone()

	counts := im.FreezeSlice([]int{1, 2})
	view := counts.ByValue()
	values := view.Clone()
	values[0] = 3
	next := im.FreezeByValueSlice(values)
	for index, value := range next.All() {
		_ = index
		_ = value
	}
	var emptySlice im.FrozenByValueSlice[int]
	var emptyMap im.FrozenByValueMap[int, int]
	_ = emptySlice.Len()
	_ = emptyMap.Len()

	key := im.FreezeString("score")
	scores := im.FreezeMap(map[im.FrozenString]int{key: 5})
	scoreView := scores.ByValue()
	score, found := scoreView.Get(im.FreezeString(native))
	_ = score
	_ = found
	copiedMap := scoreView.Clone()
	independent := im.FreezeMap(copiedMap)
	for savedKey, value := range independent.All() {
		_ = savedKey.Clone()
		_ = value
	}
	scalarMap := im.FreezeByValueMap(map[int]int{1: 2})
	for key, value := range scalarMap.All() {
		_ = key
		_ = value
	}

	builder := []Text{text}
	builder[0] = other
	nested := im.FreezeByValueSlice(builder)
	item := First(nested)
	_ = item.Clone()
	result := Pass(Reply{Text: text, Counts: next, Scores: scoreView})
	_ = result.Text.Clone()
}

type State struct {
	im.ByValue
	Counts im.FrozenSlice[int]
}

// The internal state may retain its own shared data.
var state im.Owner[State]

type Service struct{ supervise.Service[string] }

func (*Service) Run(context.Context) {}
func (*Service) Text() Text          { return TextValue("text") }
func (*Service) Reply() Reply        { return Reply{} }
func (*Service) Pair() [2]Text       { return [2]Text{TextValue("text")} }

// References remain a separate explicit service reference type.
func References(ref supervise.Ref[Service]) {
	values := im.FreezeByValueSlice([]supervise.Ref[Service]{ref})
	result := values.At(0)
	_ = result.Current().Text().Clone()
}
