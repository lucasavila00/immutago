package local_rejections

import "example.com/consumer/accepted"

var retained accepted.RecordProjection

func capture(p accepted.RecordProjection) func() int {
	return func() int { return p.Local() }
}

func pointer(p *accepted.RecordProjection) *accepted.RecordProjection { return p }

func write(p accepted.RecordProjection) { p.Unchecked().Local = 2 }
