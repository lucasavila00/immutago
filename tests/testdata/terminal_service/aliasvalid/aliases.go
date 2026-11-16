package aliasvalid

import "example.com/consumer/aliasapi"

var retryWrapped aliasapi.RetryWrapped[int]
var terminalWrapped aliasapi.TerminalWrapped[int]
var retryReordered aliasapi.RetryReordered[int, aliasapi.Point]
var terminalReordered aliasapi.TerminalReordered[int, aliasapi.Point]
var retryUnused aliasapi.RetryUnused[aliasapi.Point]
var terminalUnused aliasapi.TerminalUnused[aliasapi.Point]
