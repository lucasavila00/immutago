package explicit_references

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	X int
}

func read(value *Good) int { return value.X }
func copied()              { var value Good; p := &value; q := p; _ = read(q) }
func rebound() {
	var first Good
	var second Good
	p := &first
	p = &second
	_ = read(p)
}
func cleared() { var value Good; p := &value; p = nil; _ = p }
func delayed() { var value Good; var p *Good; p = &value; _ = read(p) }
func merged(flag bool) {
	var first Good
	var second Good
	var p *Good
	if flag {
		p = &first
	} else {
		p = &second
	}
	_ = read(p)
}

func branchLive(flag bool) {
	var value Good
	p := &value
	value.X = 1
	if flag {
		_ = read(p)
	}
}
func loopLive(count int) {
	var value Good
	p := &value
	for index := 0; index < count; index++ {
		value.X = index
		_ = read(p)
	}
}
