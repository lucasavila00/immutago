package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func (c Counter) Read() int { return int(c) }

type ScalarReply struct {
	im.Local
	Counter Counter
	Values  [2]Counter
}

type Text string

func (t Text) Read() string { return string(t) }

type TextOwner struct {
	im.Frozen
	Text Text
}

func scalarPointer(snapshot *Counted, reply ScalarReply) {
	snapshot.Counter.Inc()
	reply.Counter.Inc()
	reply.Values[0].Inc()
	bound := snapshot.Counter.Inc
	bound()
	(&reply.Counter).Inc()
	snapshot.Counter++
}

func localScalarStatements(reply ScalarReply, output chan int) {
	go reply.Counter.Read()
	defer reply.Counter.Read()
	_ = func() int { return reply.Counter.Read() }
	select {
	case output <- reply.Counter.Read():
	}
	first, second := reply.Counter.Read(), 1
	_ = first
	_ = second
}

func nonCopiedScalar(owner TextOwner) {
	_ = owner.Text.Read()
}

func scalarBuilderPointer() {
	values := []Counter{1}
	values[0].Inc()
	_ = im.FreezeSlice(values)
}

func scalarBuilderAfterFreeze() {
	values := []Counter{1}
	_ = im.FreezeSlice(values)
	_ = values[0].Read()
}

func scalarBuilderAlias() {
	values := []Counter{1}
	alias := values
	_ = alias[0].Read()
	_ = im.FreezeSlice(values)
}

func scalarBuilderAsync() {
	values := []Counter{1}
	go values[0].Read()
	defer values[0].Read()
	_ = im.FreezeSlice(values)
}
