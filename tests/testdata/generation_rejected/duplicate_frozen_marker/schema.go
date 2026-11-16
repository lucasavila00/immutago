package duplicate_frozen_marker

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type SnapshotAliasA = im.Frozen
type SnapshotAliasB = im.Frozen
type DuplicateSnapshot struct {
	SnapshotAliasA
	SnapshotAliasB
}
