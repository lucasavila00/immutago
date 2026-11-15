package uses

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type LocalError struct {
	im.Local
	N int
}

func (e LocalError) Error() string { return "local" }
func Box() error                   { return LocalError{} }

type WithError struct {
	im.Local
	Err error
}

var Saved api.Reply

func Destinations() {
	var err error
	Saved, err = api.Read()
	_ = err
	var values [1]api.Reply
	values[0], err = api.Read()
	_ = values
	_ = err
}

func Capture() func() int {
	value, _ := api.Read()
	return func() int { return value.Count.N }
}

func Dynamic() {
	callback := api.Read
	value, err := callback()
	_ = value
	_ = err
}

func pair[T any](value T) (T, error) { return value, nil }
func Generic()                       { value, err := pair(api.Reply{}); _ = value; _ = err }
func wrongPair() (api.Reply, bool)   { return api.Reply{}, false }
func WrongPair()                     { value, ok := wrongPair(); _ = value; _ = ok }
