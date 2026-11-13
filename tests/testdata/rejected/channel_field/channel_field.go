package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad struct { // want "immutable field Value: .*not provably immutable"
	im.Marker
	Value chan int
}
