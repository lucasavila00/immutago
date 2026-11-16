package storage

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Local struct {
	im.Local
	Text im.StringProjection
}

type Alias = Local

type Direct struct {
	supervise.TerminalService[Alias]
}
type Array struct {
	supervise.TerminalService[[2]Local]
}
type Pointer struct {
	supervise.TerminalService[*Local]
}
type Callback struct {
	supervise.TerminalService[func(Local)]
}
type Nested struct {
	supervise.TerminalService[struct{ Value [2]Local }]
}
