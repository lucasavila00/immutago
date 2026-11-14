package valid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Model keeps a trusted handle in immutable storage.
type Model struct {
	im.Shared
	Table im.UncheckedValue[api.Table]
}

// Wrapper embeds an ordinary handle without exposing the payload.
type Wrapper struct{ im.UncheckedValue[api.Table] }

// Copy keeps ordinary handle transport available outside the declaring package.
func Copy() (Model, any) {
	value := api.New()
	return Model{Table: value}, Wrapper{UncheckedValue: value}
}

// Read reaches the payload through its declaring package.
func Read(model *Model) int { return api.Read(model.Table) }

// FreezeSlice stores an unchecked value without inspecting its payload.
func FreezeSlice() im.FrozenSlice[im.UncheckedValue[api.Table]] {
	values := []im.UncheckedValue[api.Table]{api.New()}
	frozen := im.FreezeSlice(values)
	return frozen
}

// FreezeMap permits the same handle as a frozen map element.
func FreezeMap() im.FrozenMap[string, im.UncheckedValue[api.Table]] {
	values := map[string]im.UncheckedValue[api.Table]{"table": api.New()}
	frozen := im.FreezeMap(values)
	return frozen
}
