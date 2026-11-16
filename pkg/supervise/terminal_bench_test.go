package supervise

import (
	"context"
	"testing"
)

// terminalLifecycleWorker confirms admission before the benchmark stops its group.
type terminalLifecycleWorker struct{ TerminalService[chan struct{}] }

// Run signals readiness and waits for ordered shutdown.
func (s *terminalLifecycleWorker) Run(ctx context.Context) { close(s.Args()); <-ctx.Done() }

// BenchmarkTerminalServiceStartStop measures the same lifecycle as ServiceStartStop.
func BenchmarkTerminalServiceStartStop(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		parent := NewRoot(nil)
		started := make(chan struct{})
		if _, err := Start[terminalLifecycleWorker](parent, started); err != nil {
			b.Fatal(err)
		}
		<-started
		<-parent.Stop()
	}
}
