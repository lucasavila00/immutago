//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package privateconsumer

import (
	"example.com/consumer/privateapi"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Model struct {
	im.Frozen
	Reader privateapi.Public
}
