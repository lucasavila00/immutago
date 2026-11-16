package generic_interface

import "github.com/lucasavila00/immutago/pkg/im"

type Settings struct {
	im.Frozen
	Limit int
}
