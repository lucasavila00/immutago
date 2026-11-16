package mixed

import "github.com/lucasavila00/immutago/pkg/supervise"

type Ambiguous struct {
	supervise.Service[supervise.NoArgs]
	supervise.TerminalService[supervise.NoArgs]
}

var ambiguous supervise.Ref[Ambiguous]

type RetryBase struct {
	supervise.Service[supervise.NoArgs]
}
type TerminalBase struct {
	supervise.TerminalService[supervise.NoArgs]
}
type Nested struct {
	RetryBase
	TerminalBase
}

var nested supervise.Ref[Nested]

type DefinedRetry supervise.Service[supervise.NoArgs]
type DefinedTerminal supervise.TerminalService[supervise.NoArgs]

var retry supervise.Ref[DefinedRetry]
var terminal supervise.Ref[DefinedTerminal]

type Named struct {
	Base supervise.TerminalService[supervise.NoArgs]
}

var named supervise.Ref[Named]
