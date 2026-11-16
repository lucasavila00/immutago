package api

import "github.com/lucasavila00/immutago/pkg/im"

type Level int
type LevelAlias = Level

func (l Level) Urgent() bool { return l > 2 }
func (l *Level) Raise()      { *l++ }

type Levels struct {
	im.Frozen
	Current LevelAlias
	Values  [2]Level
	Meter   Meter[Seconds]
	Meters  [2]Meter[Seconds]
	Window  Range[Seconds]
}

type LevelReply struct {
	im.Local
	Current Level
	Values  [2]Level
	Meter   Meter[Seconds]
	Meters  [2]Meter[Seconds]
	Window  Range[Seconds]
}
