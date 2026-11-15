package frozenstring

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Text = im.FrozenString

type Record struct {
	im.Shared
	Text Text
}

type Reply struct {
	im.ByValue
	Text im.FrozenByValueString
}

// New keeps a native substring as a shareable string.
func New(text string) Record { return Record{Text: im.FreezeString(text[1:])} }

// TextValue returns a concrete frozen value across a package boundary.
func TextValue() Text { return im.FreezeString("sample") }

// Native returns independent bytes that follow ordinary Go string rules.
func Native(value Text) string { return value.Clone() }

// Forms exercises the listed frozen string forms and common value statements.
func Forms(native string) {
	var zero Text
	zero = Text{}
	text := im.FreezeString(native)
	_ = native[:1]
	copy := text
	copy = im.FreezeString("other")
	_ = text == copy
	_ = text != zero
	_ = [2]Text{text, copy}
	_ = text.Len()
	_ = text.At(0)
	plain := text.Clone()
	_ = plain[:1]
	_ = Native(text)
	switch text {
	case copy:
		_ = text
	}
	values := []Text{text, copy}
	frozen := im.FreezeSlice(values)
	for _, value := range frozen.All() {
		_ = value.Clone()
	}
	mapping := map[Text]int{text: 1}
	frozenMap := im.FreezeMap(mapping)
	value, found := frozenMap.Get(text)
	_ = value
	_ = found
}

type Catalog struct {
	supervise.Service[string]
	text string
}

func (*Catalog) Run(context.Context) {}

// Text returns a local read view of the service string.
func (c *Catalog) Text() im.FrozenByValueString { return im.FreezeByValueString(c.text) }

// Reply puts frozen text inside a marked result value.
func (c *Catalog) Reply() Reply { return Reply{Text: c.Text()} }

// Array returns a fixed array of frozen strings.
func (c *Catalog) Array() [2]im.FrozenByValueString {
	return [2]im.FrozenByValueString{c.Text()}
}

// Set keeps ordinary method input and implementation strings unrestricted.
func (c *Catalog) Set(text string) { c.text = text[:1] }
