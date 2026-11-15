package viewapi

import "github.com/lucasavila00/immutago/pkg/im"

// Text returns a local view through the checked package boundary.
func Text() im.FrozenByValueString { return im.FreezeByValueString("text") }

type Reply struct {
	im.ByValue
	Text im.FrozenByValueString
}

// Result returns a containing value with the same local storage rule.
func Result() Reply { return Reply{Text: Text()} }
