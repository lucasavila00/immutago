package service

import "github.com/lucasavila00/immutago/pkg/supervise"

func extract(ref supervise.Ref[Counter]) {
	current := ref.Current
	_ = current()
	pointer := ref.Current()
	*pointer = Counter{}
}

func erase(ref supervise.Ref[Counter]) {
	var hidden any = ref
	_ = hidden
}

func overwrite(ref supervise.Ref[Counter]) { *ref.Current() = Counter{} }
