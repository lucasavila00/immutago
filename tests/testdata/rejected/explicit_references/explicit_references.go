package explicit_references

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	X int
}

func read(value *Good) int { return value.X }
func branchWrite(flag bool) {
	var value Good
	p := new(value)
	value.X = 1
	if flag {
		_ = read(p)
	}
}
func loopWrite(count int) {
	var value Good
	p := new(value)
	for index := 0; index < count; index++ {
		value.X = index
		_ = read(p)
	}
}
