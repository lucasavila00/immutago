package ownedcross

import "example.com/consumer/owned"

// Publish stores and loads through an exported slot in another package.
func Publish() {
	owned.Current.Store(&owned.Model{Version: "v1"})
	model := owned.Current.Load()
	_ = model
}
