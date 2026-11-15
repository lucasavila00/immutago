package ownercross

import (
	"example.com/consumer/owner"
	"github.com/lucasavila00/immutago/pkg/im"
)

// Publish stores and loads through an exported slot in another package.
func Publish() {
	owner.Current.Store(&owner.Model{Version: im.FreezeString("v1")})
	model := owner.Current.Load()
	_ = model
}
