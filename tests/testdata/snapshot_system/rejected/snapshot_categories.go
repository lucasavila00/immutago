package rejected

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Retained struct {
	im.Snapshot
	Text string
}
type Local struct {
	im.Local
	Text im.StringView
}
type BadLocal struct {
	im.Local
	Text   string
	Frozen im.FrozenString
}
type BadSnapshot struct {
	im.Snapshot
	Result Local
}
type BadNative struct{ Saved im.View[string] }
type Alias = Local

var saved Local
var savedAlias Alias
var savedZero [0]im.View[string]
var badCell im.Cell[Local]
var badView im.View[Local]
var badFrozen im.FrozenSlice[Local]
var badMap im.FrozenMap[im.StringView, int]

type Service struct {
	supervise.Service[supervise.NoArgs]
}

func (*Service) Raw() string               { return "text" }
func (*Service) Frozen() im.FrozenString   { return im.FreezeString("text") }
func (*Service) Retained() Retained        { return Retained{} }
func (*Service) Container() []im.View[int] { return nil }
func (*Service) LocalPointer() *Local      { return &Local{} }

func copyRetained(view im.View[Retained], text im.View[im.FrozenString], values im.SliceView[string], entries im.MapView[string, int]) {
	_ = im.Copy(view)
	_ = im.Copy(text)
	_ = values.Clone()
	_ = entries.Clone()
}

func localStorage(value Local, text im.StringView) {
	saved = value
	_ = []Local{value}
	_ = map[int]Local{1: value}
	_ = im.ViewOf(value)
	_ = im.Trust(&value)
	_ = any(text)
	go consume(text)
	defer consume(text)
	_ = func() int { return text.Len() }
}

func consume(im.StringView) {}

func mutableLocal(value Local)                { value.Text = im.StringView{} }
func escapingIterator(view im.SliceView[int]) { all := view.All(); _ = all }
