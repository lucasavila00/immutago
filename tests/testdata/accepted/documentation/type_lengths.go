package main

import "github.com/lucasavila00/immutago/pkg/im"

type ArrayLengthValue struct {
	im.Shared
}

type ArrayLengthHolder struct {
	im.Shared
	Values [1]int
}

// OrdinaryLength keeps ordinary Go constant expressions in type syntax.
type OrdinaryLength [len([2]int{})]int
