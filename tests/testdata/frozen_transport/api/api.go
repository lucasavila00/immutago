//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package api

import "github.com/lucasavila00/immutago/pkg/im"

type Count struct {
	im.Frozen
	N int
}

func (c Count) Number() int { return c.N }
func (c *Count) Read() int  { return c.N }

type Reader interface {
	im.FrozenInterface
	Number() int
}
type Data struct {
	im.Frozen
	Text   im.FrozenString
	Counts [2]Count
	List   im.FrozenSlice[int]
	Map    im.FrozenMap[string, int]
	Reader Reader
}
type Alias = Data
type Promoted struct {
	*Count
}

var Current Data

func Identity[T any](value T) T              { return value }
func Apply[T any](value T, call func(T) T) T { return call(value) }
func Capture[T any](value T) func() T        { return func() T { return value } }
