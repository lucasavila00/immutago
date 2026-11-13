package explicit_methods

import "github.com/lucasavila00/immutago/pkg/im"

type Counter int

func (c *Counter) Increment() { *c++ }
func (c Counter) Read() int   { return int(c) }

type Good struct {
	im.Immut
	Counter
}

func promoted()                  { var value Good; pointer := &value; pointer.Increment() }
func promotedRead()              { var value Good; _ = value.Read() }
func (value *Good) OwnRead() int { return int(value.Counter) }
func direct()                    { var value Good; pointer := &value; _ = pointer.OwnRead() }
func expression()                { var value Good; _ = (*Good).OwnRead(&value) }

func (c Counter) Len() int                  { return int(c) }
func (c Counter) At(index int) int          { return int(c) + index }
func (c Counter) Get(index int) (int, bool) { return int(c) + index, true }
func (c Counter) All() int                  { return int(c) }
func (c Counter) Clone() int                { return int(c) }
func reservedNames() {
	var value Good
	_ = value.Len()
	_ = value.At(0)
	_, _ = value.Get(0)
	_ = value.All()
	_ = value.Clone()
}
