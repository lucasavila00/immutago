package foreign

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Direct reads a foreign payload.
func Direct(value im.UncheckedValue[api.Table]) *api.Table { return im.Unwrap(value) }

// Trust wraps a foreign payload.
func Trust() im.UncheckedValue[api.Table] { return im.Trust(&api.Table{}) }

// Alias retains the foreign type identity.
type Alias = api.Alias

// ReadAlias cannot change the declaring package through an alias.
func ReadAlias(value im.UncheckedValue[Alias]) *Alias { return (im.Unwrap[Alias])(value) }

// GenericAlias preserves the argument's original declaring package.
type GenericAlias[T any] = im.UncheckedValue[T]

// ReadGenericAlias infers the foreign payload through a generic alias.
func ReadGenericAlias(value GenericAlias[api.Table]) *api.Table { return im.Unwrap(value) }

// Reader names the function without calling it.
var Reader = im.Unwrap[api.Table]

// Truster names the other boundary function.
var Truster = im.Trust[api.Table]

// ReadGeneric cannot extract a pointer through a generic helper.
func ReadGeneric[T any](value im.UncheckedValue[T]) *T { return im.Unwrap(value) }

// ReadInstance cannot extract a foreign named instance.
func ReadInstance(value im.UncheckedValue[api.Box[int]]) *api.Box[int] { return im.Unwrap(value) }

// Wrapper embeds a handle without acquiring its declaring package's access.
type Wrapper struct{ im.UncheckedValue[api.Table] }

// ReadEmbedded still uses the same foreign payload type.
func ReadEmbedded(value Wrapper) *api.Table { return im.Unwrap(value.UncheckedValue) }

// Defined copies the handle representation but cannot read its private field.
type Defined im.UncheckedValue[api.Table]

// ReadDefined restores the original handle type before attempting extraction.
func ReadDefined(value Defined) *api.Table { return im.Unwrap(im.UncheckedValue[api.Table](value)) }

// InferredReader infers the same foreign type argument from its declaration.
var InferredReader func(im.UncheckedValue[api.Table]) *api.Table = im.Unwrap
