package cell_holder

import api "example.com/consumer/cell_holder_api"

var Saved *api.Alias
var SavedBox api.Parent
var Callback api.Callback = api.Retain
var Factory = api.New
var Empty = (*api.Holder)(nil)

func Construct() *api.Holder {
	h := api.New()
	SavedBox = api.Parent{Child: h}
	Saved = api.Identity(h)
	return api.Same(SavedBox.Child)
}

func Pointers(h *api.Holder) *api.Holder {
	alias := h
	pointer := &alias
	*pointer = api.New()
	h, alias = alias, h
	parent, other := api.Share(h)
	Saved, SavedBox = other, parent
	defined := api.Pointer(h)
	defined.Calls++
	if h == (*api.Alias)(nil) {
		return nil
	}
	return (*api.Holder)(defined)
}

func Containers(h *api.Holder) *api.Holder {
	array := [1]*api.Holder{h}
	values := []*api.Holder{h, {Calls: 2}}
	values = append(values, array[:]...)
	copy(values, array[:])
	for _, value := range values {
		value.Publish(2)
	}
	lookup := map[*api.Holder]*api.Holder{h: h}
	for key, value := range lookup {
		key.Publish(value.Read())
	}
	delete(lookup, h)
	clear(values)
	channel := make(chan *api.Holder, 1)
	select {
	case channel <- h:
	default:
	}
	return <-channel
}

func Methods(h *api.Holder, nested *api.Nested) {
	h.Calls++
	(*h).Calls = 3
	count := &h.Calls
	*count = 4
	child := &nested.Child
	child.Publish(1)
	call := nested.Child.Publish
	call(2)
	pointer := (*api.Holder).Publish
	pointer(child, 3)
	embedded := api.Embedded{Holder: h}
	embedded.Publish(4)
	promoted := embedded.Publish
	promoted(5)
	api.Embedded.Publish(embedded, 6)
	var store api.Store = embedded
	store.Publish(7)
	var value any = h
	holder, ok := value.(*api.Holder)
	if ok {
		holder.Publish(8)
	}
	switch selected := value.(type) {
	case *api.Holder:
		selected.Publish(9)
	}
	go promoted(10)
	defer pointer(h, 11)
}

func Calls(h *api.Holder) *api.Holder {
	box := api.Box[*api.Holder]{Value: h}
	callback := Callback
	returned := callback(box.Value, h)
	closure := api.Closure(returned)
	go returned.Publish(1)
	defer returned.Publish(2)
	return closure()
}
