package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type SharedAliasA = im.Shared
type SharedAliasB = im.Shared
type LocalAliasA = im.LocalRef
type LocalAliasB = im.LocalRef

type DuplicateShared struct {
	SharedAliasA
	SharedAliasB
}

type DuplicateLocal struct {
	LocalAliasA
	LocalAliasB
}

type MixedAliases struct {
	SharedAliasA
	LocalAliasA
}
