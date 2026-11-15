package localviews

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Text = im.FrozenByValueString
type Reply struct {
	im.ByValue
	Text Text
}
type Shared struct {
	im.Shared
	Count int
}

var zero Text
var saved = im.FreezeByValueString("text")
var record Reply
var empty [0]Text
var slot im.Owner[Reply]
var unchecked im.UncheckedValue[Reply]

type Stored struct {
	im.Shared
	Reply Reply
}
type Mutable struct{ Text Text }
type Service struct{ supervise.Service[Reply] }

func (*Service) Run(context.Context)   {}
func (*Service) Text() im.FrozenString { return im.FreezeString("text") }

type Retained struct {
	im.ByValue
	Text im.FrozenString
}

func (*Service) Retained() Retained { return Retained{} }

type PointerResult struct {
	im.ByValue
	Pointer *Shared
}

func (*Service) Pointer() PointerResult { return PointerResult{} }

// Store cannot retain a whole record or a value selected from it.
func Store(reply Reply) {
	record = reply
	saved = reply.Text
	empty = [0]Text{}
}

// Escape checks conversions, generic calls, callbacks, and delayed calls.
func Escape(text Text) {
	var boxed any = text
	_ = boxed
	_ = identity(text)
	callback := func() int { return text.Len() }
	_ = callback
	go text.Len()
	defer text.Len()
	method := text.Clone
	_ = method
}
func identity[T any](value T) T { return value }

// FreezeShared cannot put local views in persistent storage.
func FreezeShared(text Text) {
	values := []Text{text}
	_ = im.FreezeSlice(values)
}

// Consume applies the same builder rules to local freezing.
func Consume() {
	values := []int{1}
	_ = im.FreezeByValueSlice(values)
	_ = values[0]
}

// Alias does not grant shared ownership to a local freeze.
func Alias() {
	values := map[int]int{1: 2}
	alias := values
	_ = im.FreezeByValueMap(values)
	_ = alias
}
