//go:build integration

package declaration

import "github.com/lucasavila00/immutago/pkg/im"

type AuditEntry struct {
	im.Frozen
	Actor string
}
