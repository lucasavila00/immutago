package supervise

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkGroupForget measures removal of a full group, without setup costs.
func BenchmarkGroupForget(b *testing.B) {
	for _, size := range []int{128, 512, 2048} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				b.StopTimer()
				parent := &Group{}
				children := make([]*child, size)
				for i := range children {
					nested := &Group{}
					parent.add(func(context.Context) {}, func() {}, nested)
					children[i] = nested.entry
				}
				b.StartTimer()
				for _, c := range children {
					parent.forget(c)
				}
			}
		})
	}
}
