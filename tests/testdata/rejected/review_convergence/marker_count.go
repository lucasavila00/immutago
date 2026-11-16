package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type SnapshotAliasA = im.Frozen
type SnapshotAliasB = im.Frozen
type LocalAliasA = im.Local
type LocalAliasB = im.Local

type DuplicateLocal struct {
	LocalAliasA
	LocalAliasB
}
