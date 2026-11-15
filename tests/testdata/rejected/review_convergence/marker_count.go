package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type SharedAliasA = im.Shared
type SharedAliasB = im.Shared
type LocalAliasA = im.Scoped
type LocalAliasB = im.Scoped

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
