package accepted

import (
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
)

type Count int
type CountAlias = Count
type Flag bool

func (c Count) Read() int           { return int(c) }
func (c Count) Next() Count         { c++; return c }
func (c Count) Pair() (Count, bool) { return c, true }
func (c Count) Add(values ...Count) Count {
	for _, value := range values {
		c += value
	}
	return c
}
func (c *Count) Inc()                         { *c++ }
func (f Flag) Enabled() bool                  { return bool(f) }
func (c Count) Map[T any](read func(int) T) T { return read(int(c)) }

type Score struct {
	im.Copied
	Count Count
}

func (s Score) Read() int { return s.Count.Read() }

type ScalarSnapshot struct {
	im.Frozen
	Count   Count
	Alias   CountAlias
	Counts  [2][2]Count
	Score   Score
	Flag    Flag
	Timeout time.Duration
}

type ScalarReply struct {
	im.Local
	Count   Count
	Counts  [2][2]Count
	Score   Score
	Flag    Flag
	Timeout time.Duration
}

func scalarIdentity(value int) int { return value }

func scalarReads(snapshot ScalarSnapshot, reply *ScalarReply) (Count, bool) {
	_ = snapshot.Count.Read()
	_ = snapshot.Counts[0][1].Read()
	_ = snapshot.Score.Count.Read()
	_ = reply.Count.Read()
	_ = reply.Counts[0][1].Read()
	_ = reply.Score.Count.Read()
	_ = snapshot.Flag.Enabled()
	_ = reply.Flag.Enabled()
	_ = ((snapshot.Alias).Read)()
	_ = Count.Read(snapshot.Count)
	_ = Count.Read(reply.Count)
	_ = snapshot.Count.Add(snapshot.Alias, snapshot.Counts[0][1])
	_ = snapshot.Count.Map(scalarIdentity)
	_ = snapshot.Count.Map[int](scalarIdentity)
	_ = snapshot.Timeout.Seconds()
	_ = reply.Timeout.Seconds()
	var count, found = snapshot.Count.Pair()
	_ = count
	_ = found
	return reply.Count.Pair()
}

func scalarCopies(snapshot *ScalarSnapshot, reply ScalarReply) (Count, Count) {
	read := reply.Count.Next
	readScore := reply.Score.Read
	_ = readScore()
	return snapshot.Count.Next(), read()
}

func scalarZero() int {
	var snapshot ScalarSnapshot
	_ = ScalarSnapshot{}.Count.Read()
	return snapshot.Count.Read()
}

func scalarOrdinaryCopy(reply ScalarReply) func() int {
	count := reply.Count
	count.Inc()
	return count.Read
}

func scalarFrozenAsync(snapshot ScalarSnapshot) {
	go snapshot.Count.Read()
	defer snapshot.Count.Read()
}

func scalarBuilderReads() float64 {
	timeouts := []time.Duration{time.Second}
	_ = timeouts[0].Seconds()
	read := timeouts[0].Seconds
	frozen := im.FreezeSlice(timeouts)
	_ = frozen.At(0).Seconds()
	var empty map[int]Count
	_ = empty[0].Read()
	_ = im.FreezeMap(empty)
	counts := map[int]Count{0: 1}
	_ = counts[0].Read()
	_ = Count.Read(counts[0])
	_ = im.FreezeMap(counts)
	return read()
}
