package consumer

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

func genericReads(levels *api.Levels, reply api.LevelReply) (api.Duration, bool) {
	_ = levels.Meter.Read()
	_ = levels.Meters[0].Read()
	_ = levels.Window.Width()
	_ = reply.Meter.Read()
	_ = reply.Meters[0].Read()
	_ = reply.Window.Width()
	_ = api.Duration.Read(reply.Meter)
	_ = api.Range[api.Seconds].Width(reply.Window)
	_ = levels.Meter.Add(reply.Meter, reply.Meters[0])
	return reply.Meter.Pair()
}

func genericCopies(levels *api.Levels, reply api.LevelReply) (api.Duration, api.Duration) {
	read := reply.Meter.Next
	width := reply.Window.Width
	_ = width()
	return levels.Meter.Next(), read()
}

func genericOrdinaryCopy(reply api.LevelReply) int {
	window := reply.Window
	window.Grow()
	return window.Width()
}

func genericBuilderRead() int {
	meters := []api.Duration{4}
	read := meters[0].Read
	_ = im.FreezeSlice(meters)
	windows := map[int]api.Range[api.Seconds]{0: {Low: 1, High: 3}}
	_ = windows[0].Width()
	_ = im.FreezeMap(windows)
	return read()
}
