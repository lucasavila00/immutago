package relay

import (
	"example.com/consumer/deep"
	"example.com/consumer/state"
)

// Relay calls a function of another checked package that changes a package map.
func Relay(key string) { state.Record(key) }

// NewTracker returns a value whose method changes a package map of another package.
func NewTracker() deep.Tracker { return deep.Tracker{} }

// Echo uses a marked type, so the checker checks this package.
func Echo(note deep.Note) string { return note.Text }
