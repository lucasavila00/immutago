//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package service

import "github.com/lucasavila00/immutago/pkg/supervise"

func extract(ref supervise.Ref[Counter]) {
	current := ref.Borrow
	_ = current()
	pointer := ref.Borrow()
	*pointer = Counter{}
}

func overwrite(ref supervise.Ref[Counter]) { *ref.Borrow() = Counter{} }
