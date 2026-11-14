package valid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Model keeps a trusted handle in immutable storage.
type Model struct {
	im.Shared
	Table im.Opaque[api.Table]
}

// Wrapper embeds an ordinary handle without exposing the payload.
type Wrapper struct{ im.Opaque[api.Table] }

// Copy keeps ordinary handle transport available outside the declaring package.
func Copy() (Model, any) {
	value := api.New()
	return Model{Table: value}, Wrapper{Opaque: value}
}

// Read reaches the payload through its declaring package.
func Read(model *Model) int { return api.Read(model.Table) }
