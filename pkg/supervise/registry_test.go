package supervise

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
)

// TestForgetUnlinksChild checks removal at each position and repeated removal.
func TestForgetUnlinksChild(t *testing.T) {
	for _, order := range [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {1, 2, 0, 3}} {
		parent := &Group{}
		children := make([]*child, 4)
		for i := range children {
			nested := &Group{}
			parent.add(func(context.Context) {}, func() {}, nested)
			children[i] = nested.entry
		}
		remaining := slices.Clone(children)
		for _, i := range order {
			c := children[i]
			parent.forget(c)
			parent.forget(c)
			remaining = slices.DeleteFunc(remaining, func(other *child) bool { return other == c })
			if c.run != nil || c.prev != nil || c.next != nil {
				t.Fatal("removed child keeps its run or list links")
			}
			if parent.children.count != len(remaining) {
				t.Fatalf("child count = %d, want %d", parent.children.count, len(remaining))
			}
			var forward, backward []*child
			for c := parent.children.first; c != nil; c = c.next {
				forward = append(forward, c)
			}
			for c := parent.children.last; c != nil; c = c.prev {
				backward = append(backward, c)
			}
			slices.Reverse(backward)
			if !slices.Equal(forward, remaining) || !slices.Equal(backward, remaining) {
				t.Fatal("removal changed the order of remaining children")
			}
		}
		parent.forget(&child{})
		if parent.children.first != nil || parent.children.last != nil || parent.children.count != 0 {
			t.Fatal("empty group retains a child")
		}
	}
}

// TestForgetDoesNotAllocate checks a full group removal, including the warmup run.
func TestForgetDoesNotAllocate(t *testing.T) {
	var parents [2]Group
	var children [2][128]*child
	for i := range parents {
		for j := range children[i] {
			nested := &Group{}
			parents[i].add(func(context.Context) {}, func() {}, nested)
			children[i][j] = nested.entry
		}
	}
	next := 0
	allocs := testing.AllocsPerRun(1, func() {
		for _, c := range children[next] {
			parents[next].forget(c)
		}
		next++
	})
	if allocs != 0 {
		t.Fatalf("child removal allocated %v times, want 0", allocs)
	}
}

// TestHaltKeepsChildSnapshot checks order while children are removed and added.
func TestHaltKeepsChildSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Group{}
		entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
		var stopped []int
		var children [3]*child
		for i := range children {
			nested := &Group{}
			parent.add(func(context.Context) {}, func() {
				stopped = append(stopped, i)
				if i == 2 {
					close(entered)
					<-resume
				}
			}, nested)
			children[i] = nested.entry
		}
		go func() {
			parent.halt()
			close(done)
		}()
		<-entered
		parent.forget(children[1])
		parent.forget(children[2])
		parent.add(func(context.Context) {}, func() { stopped = append(stopped, 3) }, nil)
		close(resume)
		<-done
		if !slices.Equal(stopped, []int{2, 1, 0}) {
			t.Fatalf("stop order = %v, want [2 1 0]", stopped)
		}
	})
}

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
