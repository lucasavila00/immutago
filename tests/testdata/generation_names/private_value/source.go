package private_value

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"example.com/consumer/privateapi"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Record struct {
	im.Frozen
	Field privateapi.Public
}
