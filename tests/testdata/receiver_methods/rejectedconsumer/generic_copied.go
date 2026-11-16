package rejectedconsumer

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

func genericPointers(levels *api.Levels, reply api.LevelReply) {
	levels.Meter.Inc()
	levels.Meters[0].Inc()
	reply.Window.Grow()
	bound := reply.Meter.Inc
	bound()
	(&reply.Meter).Inc()
	levels.Meter++
}

func genericLocalStatements(reply api.LevelReply) {
	go reply.Meter.Read()
	defer reply.Window.Width()
	_ = func() int { return reply.Meter.Read() }
	first, second := reply.Meter.Read(), 1
	_ = first
	_ = second
}

func genericBuilderPointer() {
	meters := []api.Duration{1}
	meters[0].Inc()
	_ = im.FreezeSlice(meters)
}

func genericBuilderAfterFreeze() {
	meters := []api.Duration{1}
	_ = im.FreezeSlice(meters)
	_ = meters[0].Read()
}
