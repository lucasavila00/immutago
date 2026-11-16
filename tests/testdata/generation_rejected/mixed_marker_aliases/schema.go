package mixed_marker_aliases

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type SnapshotAliasA = im.Frozen
type LocalAliasA = im.Local
type MixedAliases struct {
	SnapshotAliasA
	LocalAliasA
}
