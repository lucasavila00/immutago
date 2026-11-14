package main

import "github.com/lucasavila00/immutago/pkg/im"

type ArrayLengthValue struct {
	im.Shared
}

type ArrayLengthHolder struct {
	im.Shared
	Values [len([1]ArrayLengthValue{})]int
}
