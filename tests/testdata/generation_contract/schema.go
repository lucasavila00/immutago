//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package contract

import "github.com/lucasavila00/immutago/pkg/im"

type Data struct {
	im.Frozen
	Number int
	Values [2]int
	Text   string
}
