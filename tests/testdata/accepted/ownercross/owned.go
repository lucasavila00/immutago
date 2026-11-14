package ownercross

import "example.com/consumer/owner"

// Publish stores and loads through an exported slot in another package.
func Publish() {
	owner.Current.Store(&owner.Model{Version: "v1"})
	model := owner.Current.Load()
	_ = model
}
