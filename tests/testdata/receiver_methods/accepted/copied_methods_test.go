package accepted

import (
	"testing"
	"time"
)

func TestCopiedReceiverCopies(t *testing.T) {
	snapshot := &ScalarSnapshot{Count: 4, Flag: true, Timeout: 2 * time.Second}
	reply := &ScalarReply{Count: 6, Flag: true, Timeout: 3 * time.Second}
	first, second := scalarCopies(snapshot, *reply)
	if first != 5 || second != 7 || snapshot.Count != 4 || reply.Count != 6 {
		t.Fatalf("copies = %d, %d; owners = %d, %d", first, second, snapshot.Count, reply.Count)
	}
	if snapshot.Timeout.Seconds() != 2 || reply.Timeout.Seconds() != 3 {
		t.Fatal("duration receiver did not read the copied field")
	}
	if !snapshot.Flag.Enabled() || !reply.Flag.Enabled() {
		t.Fatal("boolean receiver did not read the copied field")
	}
	if count, found := scalarReads(*snapshot, reply); count != 6 || !found {
		t.Fatalf("read = %d, %t", count, found)
	}
	if scalarZero() != 0 || scalarOrdinaryCopy(*reply)() != 7 {
		t.Fatal("zero or ordinary copied receiver did not match")
	}
	if scalarBuilderReads() != 1 {
		t.Fatal("builder method value did not copy its receiver before Freeze")
	}
}
