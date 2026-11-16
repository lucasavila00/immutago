package aliasrejected

import (
	"example.com/consumer/aliasapi"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

var retryWrapped aliasapi.RetryWrapped[aliasapi.Point]
var terminalWrapped aliasapi.TerminalWrapped[aliasapi.Point]
var directRetry supervise.Service[[]aliasapi.Point]
var directTerminal supervise.TerminalService[[]aliasapi.Point]

type Local struct{ im.Local }

var retryReordered aliasapi.RetryReordered[int, Local]
var terminalReordered aliasapi.TerminalReordered[int, Local]
var retryUnused aliasapi.RetryUnused[Local]
var terminalUnused aliasapi.TerminalUnused[Local]
