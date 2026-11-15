package valid

import (
	"example.com/consumer/api"
	"testing"
)

func TestSavedErrorsUseGoContracts(t *testing.T) {
	pool := &api.Pool{}
	err := pool.Failure()
	saved := err
	if saved.Error() != "count: 0" {
		t.Fatal(saved)
	}
	count, err := pool.Add("worker")
	if err != nil || count.N != 1 {
		t.Fatal(count, err)
	}
	if saved.Error() != "count: 1" {
		t.Fatal("error did not retain ordinary state")
	}
	reply, err := pool.Read()
	if err != nil || reply.Count.N != 1 {
		t.Fatal("wrong read result")
	}
}
