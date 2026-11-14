package explicit_bindings

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	X int
}

func read(value *Good) int { return value.X }
func copied()              { var value Good; p := new(value); q := p; _ = read(q) }
func rebound() {
	var first Good
	var second Good
	p := new(first)
	p = new(second)
	_ = read(p)
}
func cleared() { var value Good; p := new(value); p = nil; _ = p }
func delayed() { var value Good; var p *Good; p = new(value); _ = read(p) }
func merged(flag bool) {
	var first Good
	var second Good
	var p *Good
	if flag {
		p = new(first)
	} else {
		p = new(second)
	}
	_ = read(p)
}

func nilArgument() { _ = read((nil)) }
