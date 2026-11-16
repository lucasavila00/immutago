//go:build integration

package onlytag

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type AuditEntry struct {
	im.Frozen
	Actor string
}
