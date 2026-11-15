package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type SnapshotAliasA = im.Snapshot
type SnapshotAliasB = im.Snapshot
type LocalAliasA = im.Local
type LocalAliasB = im.Local

type DuplicateSnapshot struct {
	SnapshotAliasA
	SnapshotAliasB
}

type DuplicateLocal struct {
	LocalAliasA
	LocalAliasB
}

type MixedAliases struct {
	SnapshotAliasA
	LocalAliasA
}
