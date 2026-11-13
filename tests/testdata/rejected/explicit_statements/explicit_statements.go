package explicit_statements

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X     int
	Ready bool
}

func multi() { var value Good; value.X, value.Ready = 1, true }
func receive(ready <-chan int) {
	var value Good
	select {
	case _, value.Ready = <-ready:
	}
}
func send(ready chan<- Good) {
	var value Good
	select {
	case ready <- value:
	}
}
func frozenReceive(ready <-chan int) {
	values := []bool{false}
	frozen := im.FreezeSlice(values)
	select {
	case _, values[0] = <-ready:
	}
	_ = frozen.At(0)
}
func builderReceive(ready <-chan int) {
	values := []bool{false}
	select {
	case _, values[0] = <-ready:
	}
	_ = im.FreezeSlice(values)
}
func returned() im.FrozenSlice[int]              { values := []int{1}; return im.FreezeSlice(values) }
func use(value im.FrozenSlice[int])              {}
func nested()                                    { values := []int{1}; use(im.FreezeSlice(values)) }
func watch(value im.FrozenSlice[int]) <-chan int { return nil }
func selectFreeze() {
	values := []int{1}
	select {
	case <-watch(im.FreezeSlice(values)):
	}
}

func makePair() (Good, Good) { return Good{}, Good{} }
func multipleResults()       { first, second := makePair(); _ = first; _ = second }
func scalarSend(ready chan<- int) {
	var value Good
	select {
	case ready <- value.X:
	}
}
func nilReference() { var p *Good = nil; _ = p }

func frozenMapReceive(ready <-chan int) {
	values := map[string]bool{"first": false}
	frozen := im.FreezeMap(values)
	select {
	case _, values["first"] = <-ready:
	}
	_, _ = frozen.Get("first")
}
func returnedMap() im.FrozenMap[string, int] {
	values := map[string]int{"first": 1}
	return im.FreezeMap(values)
}

func consume(value Good) {}
func deferredValue()     { value := Good{}; defer consume(value) }
func concurrentValue()   { value := Good{}; go consume(value) }
func protectedTypeSwitch(input any) {
	switch input.(type) {
	case Good:
	}
}
