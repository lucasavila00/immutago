//go:build integration

package onlytag

import "github.com/lucasavila00/immutago/pkg/im"

type Reply struct {
	im.Local
	Count int
}

var saved Reply
