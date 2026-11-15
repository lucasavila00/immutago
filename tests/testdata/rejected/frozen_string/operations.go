package frozenstring

import (
	"reflect"

	"github.com/lucasavila00/immutago/pkg/im"
)

func Box(text im.FrozenString) any { return text }

func MethodValue(text im.FrozenString) {
	clone := text.Clone
	_ = clone()
}

func MethodExpression(text im.FrozenString) { _ = im.FrozenString.Clone(text) }

var Constructor = im.FreezeString

func Capture(text im.FrozenString) { _ = func() string { return text.Clone() } }

func Delayed(text im.FrozenString) {
	go text.Clone()
	defer text.Clone()
}

func Multiple() {
	first, second := im.FreezeString("a"), im.FreezeString("b")
	_ = first
	_ = second
}

type Record struct {
	im.Shared
	Text im.FrozenString
}

func Write(record Record) { record.Text = im.FreezeString("replacement") }

type Other im.FrozenString

func Convert(text im.FrozenString) Other { return Other(text) }

func Pointer() { _ = new(im.FrozenString) }

func Generic[T any](value T) T { return value }
func Instantiate()             { _ = Generic(im.FreezeString("text")) }

func Reflect(text im.FrozenString) { _ = reflect.ValueOf(text) }

func Variadic(text ...im.FrozenString) {}
