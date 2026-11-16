package supervise

import (
	"context"
	"sync/atomic"
	"testing"
)

// lifecycleWorker confirms Run admission before the benchmark stops its group.
type lifecycleWorker struct{ Service[chan struct{}] }

// Run signals readiness and waits for ordered shutdown.
func (s *lifecycleWorker) Run(ctx context.Context) { close(s.Args()); <-ctx.Done() }

// BenchmarkServiceStartStop measures setup, an admitted Run, and complete shutdown.
func BenchmarkServiceStartStop(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		parent := NewRoot(nil)
		started := make(chan struct{})
		if _, err := Start[lifecycleWorker](parent, started); err != nil {
			b.Fatal(err)
		}
		<-started
		<-parent.Stop()
	}
}

// attemptWorker supplies a small Init and Run for the successful-attempt benchmark.
type attemptWorker struct {
	Service[NoArgs]
	state int
}

// Init prepares one value before publication.
func (s *attemptWorker) Init() error { s.state = 1; return nil }

// Run completes this benchmark attempt.
func (*attemptWorker) Run(context.Context) {}

// BenchmarkServiceAttempt measures fresh value setup, publication, Run, and cleanup.
// It excludes watcher creation, registry work, failure reports, and retry backoff.
func BenchmarkServiceAttempt(b *testing.B) {
	parent := NewRoot(nil)
	defer func() { <-parent.Stop() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ref := Ref[attemptWorker]{value: new(atomic.Pointer[attemptWorker])}
	var value *attemptWorker
	run := func(ctx context.Context) {
		value = new(attemptWorker)
		value.bind(NoArgs{}, parent)
		if err := value.Init(); err != nil {
			panic(err)
		}
		ref.value.Store(value)
		value.Run(value.begin(ctx))
	}
	b.ReportAllocs()
	for b.Loop() {
		if attempt(ctx, run, func() { value.end() }, false, func(value any, _ []byte, _ bool) { b.Fatal(value) }) {
			b.Fatal("service attempt failed")
		}
	}
}
