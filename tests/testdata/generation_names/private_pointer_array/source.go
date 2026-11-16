package private_pointer_array

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"example.com/consumer/privateapi"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Record struct {
	im.Frozen
	Field [2]*privateapi.Public
}
