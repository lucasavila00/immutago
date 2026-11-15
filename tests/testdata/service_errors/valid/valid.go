package valid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Reply = api.Reply

var SavedError error
var SavedCount api.Count

func consume(value Reply, err error)  { _ = value.Count.N; SavedError = err }
func Forward() (Reply, error)         { return api.Read() }
func Named() (value Reply, err error) { value, err = api.Read(); return }

func Use(ref supervise.Ref[api.Pool]) {
	value, err := ref.Borrow().Read()
	consume(value, err)
	value, err = (api.Read)()
	consume(value, err)
	_, err = ref.Borrow().Forward()
	SavedError = err
	var other, failure = api.Read()
	consume(other, failure)
	var _, ignored = api.Read()
	SavedError = ignored
	_, _ = api.Read()
	value, _ = api.Read()
	_ = value
	if next, err := ref.Borrow().Named(); err == nil {
		consume(next, err)
	}
	consume(ref.Borrow().Read())
	ref.Borrow().Read()
	SavedError = ref.Borrow().Restart()
	SavedError = ref.Borrow().Failure()
	count, err := ref.Borrow().Add("worker")
	SavedCount = count
	SavedError = err
}

// Ordinary pairs keep ordinary Go calls and destination forms.
func Ordinary(callback func() (int, error), values []int) {
	var err error
	values[0], err = callback()
	SavedError = err
}

func scalar(value Reply) (int, error) { return value.Count.N, nil }
func ReadScalar() int {
	value, err := api.Read()
	if err != nil {
		return 0
	}
	count, err := scalar(value)
	if err != nil {
		return 0
	}
	return count
}

func pair() (api.Reply, bool) { return api.Reply{}, false }
func ReadPair()               { value, ok := pair(); _ = value; _ = ok }
