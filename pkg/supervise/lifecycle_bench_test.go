package supervise

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/im"
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

// runWorker supplies a small Init and Run for the service run benchmark.
type runWorker struct {
	Service[NoArgs]
	state int
}

// Init prepares one value before publication.
func (s *runWorker) Init(*Supervisor) error { s.state = 1; return nil }

// Run completes this benchmark Run.
func (*runWorker) Run(context.Context) {}

// BenchmarkServiceRun measures fresh value setup, publication, Run, and cleanup.
// It excludes watcher creation, registry work, and failure reports.
func BenchmarkServiceRun(b *testing.B) {
	parent := NewRoot(nil)
	defer func() { <-parent.Stop() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ref Ref[runWorker]
	b.ReportAllocs()
	for b.Loop() {
		value := new(runWorker)
		value.bind(NoArgs{})
		supervisor := newSupervisor(parent, false)
		if err := value.Init(supervisor); err != nil {
			b.Fatal(err)
		}
		ref = Ref[runWorker]{value: im.Trust(value)}
		watch(parent, supervisor, ctx, "runWorker", value.Run)
		if ref.IsZero() {
			b.Fatal("service did not publish its value")
		}
	}
}
