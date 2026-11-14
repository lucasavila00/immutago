package foreign

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Direct reads a foreign payload.
func Direct(value im.Opaque[api.Table]) *api.Table { return im.Unseal(value) }

// Seal wraps a foreign payload.
func Seal() im.Opaque[api.Table] { return im.Seal(&api.Table{}) }

// Alias retains the foreign type identity.
type Alias = api.Alias

// ReadAlias cannot change the declaring package through an alias.
func ReadAlias(value im.Opaque[Alias]) *Alias { return (im.Unseal[Alias])(value) }

// GenericAlias preserves the argument's original declaring package.
type GenericAlias[T any] = im.Opaque[T]

// ReadGenericAlias infers the foreign payload through a generic alias.
func ReadGenericAlias(value GenericAlias[api.Table]) *api.Table { return im.Unseal(value) }

// Reader names the function without calling it.
var Reader = im.Unseal[api.Table]

// Sealer names the other boundary function.
var Sealer = im.Seal[api.Table]

// ReadGeneric cannot extract a pointer through a generic helper.
func ReadGeneric[T any](value im.Opaque[T]) *T { return im.Unseal(value) }

// ReadInstance cannot extract a foreign named instance.
func ReadInstance(value im.Opaque[api.Box[int]]) *api.Box[int] { return im.Unseal(value) }

// Wrapper embeds a handle without acquiring its declaring package's access.
type Wrapper struct{ im.Opaque[api.Table] }

// ReadEmbedded still uses the same foreign payload type.
func ReadEmbedded(value Wrapper) *api.Table { return im.Unseal(value.Opaque) }

// Defined copies the handle representation but cannot read its private field.
type Defined im.Opaque[api.Table]

// ReadDefined restores the original handle type before attempting extraction.
func ReadDefined(value Defined) *api.Table { return im.Unseal(im.Opaque[api.Table](value)) }

// InferredReader infers the same foreign type argument from its declaration.
var InferredReader func(im.Opaque[api.Table]) *api.Table = im.Unseal
