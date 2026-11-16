package imported

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"example.com/consumer/alias"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Entry struct {
	im.Frozen
	Item alias.Entry
}
