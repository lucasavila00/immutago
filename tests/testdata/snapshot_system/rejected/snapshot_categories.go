package rejected

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Retained struct {
	im.Frozen
	Text string
}
type Local struct {
	im.Local
	Text im.StringProjection
}
type BadLocal struct {
	im.Local
	Text   string
	Frozen im.FrozenString
}
type BadSnapshot struct {
	im.Frozen
	Result Local
}
type BadNative struct{ Saved im.Projection[string] }
type Alias = Local

var saved Local
var savedAlias Alias
var savedZero [0]im.Projection[string]
var badCell im.Cell[Local]
var badProjection im.Projection[Local]
var badFrozen im.FrozenSlice[Local]
var badMap im.FrozenMap[im.StringProjection, int]

type Service struct {
	supervise.Service[supervise.NoArgs]
}

func (*Service) Raw() string                     { return "text" }
func (*Service) Frozen() im.FrozenString         { return im.FreezeString("text") }
func (*Service) Retained() Retained              { return Retained{} }
func (*Service) Container() []im.Projection[int] { return nil }
func (*Service) LocalPointer() *Local            { return &Local{} }

func copyRetained(projection im.Projection[Retained], text im.Projection[im.FrozenString], values im.SliceProjection[string], entries im.MapProjection[string, int]) {
	_ = im.Copy(projection)
	_ = im.Copy(text)
	_ = values.Clone()
	_ = entries.Clone()
}

func localStorage(value Local, text im.StringProjection) {
	saved = value
	_ = []Local{value}
	_ = map[int]Local{1: value}
	_ = im.ProjectionOf(value)
	_ = im.Trust(&value)
	_ = any(text)
	go consume(text)
	defer consume(text)
	_ = func() int { return text.Len() }
}

func consume(im.StringProjection) {}

func mutableLocal(value Local)                            { value.Text = im.StringProjection{} }
func escapingIterator(projection im.SliceProjection[int]) { all := projection.All(); _ = all }

func freezeLocal(value im.StringProjection) {
	values := []im.StringProjection{value}
	frozen := im.FreezeSlice(values)
	_ = frozen
}

func returnLocalBuilder(value im.StringProjection) []im.StringProjection {
	values := []im.StringProjection{value}
	return values
}
