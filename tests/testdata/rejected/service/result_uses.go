package service

import "github.com/lucasavila00/immutago/pkg/supervise"

func (*Counter) Read() Reply { return Reply{} }

var saved *Reply

func retain(ref supervise.Ref[Counter]) {
	message := ref.Borrow().Read()
	pointer := new(message)
	saved = pointer
}

func change(ref supervise.Ref[Counter]) {
	message := ref.Borrow().Read()
	message.Count = 1
}

func capture(ref supervise.Ref[Counter]) {
	message := ref.Borrow().Read()
	pointer := new(message)
	callback := func() { _ = pointer }
	callback()
}
