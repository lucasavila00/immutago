package consumer

import (
	"testing"

	"example.com/consumer/api"
)

func TestGenericCopiedReceivers(t *testing.T) {
	levels := &api.Levels{Meter: 4, Window: api.Range[api.Seconds]{Low: 1, High: 3}}
	reply := api.LevelReply{Meter: 6, Window: api.Range[api.Seconds]{Low: 2, High: 5}}
	first, second := genericCopies(levels, reply)
	if first != 5 || second != 7 || levels.Meter != 4 || reply.Meter != 6 {
		t.Fatal("value receiver changed its protected owner")
	}
	if value, found := genericReads(levels, reply); value != 6 || !found {
		t.Fatalf("read = %d, %t", value, found)
	}
	if genericOrdinaryCopy(reply) != 4 || reply.Window.Width() != 3 || genericBuilderRead() != 4 {
		t.Fatal("copied receiver did not keep its independent value")
	}
}
