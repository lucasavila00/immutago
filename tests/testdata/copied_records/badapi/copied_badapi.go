package badapi

import "github.com/lucasavila00/immutago/pkg/im"

type Bad struct {
	im.Copied
	Names [0]string
}

type Nested struct {
	im.Copied
	Value [2]Bad
}

type Alias = Nested
