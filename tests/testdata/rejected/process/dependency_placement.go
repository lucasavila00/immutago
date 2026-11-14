package process

import (
	"context"
	"log/slog"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// SenderArgs are valid process arguments with a dependency.
type SenderArgs struct {
	im.Shared
	Log proc.UncheckedDependency[*slog.Logger]
}

// Sender stores a dependency in a process field.
type Sender struct {
	proc.Process[SenderArgs]
	log proc.UncheckedDependency[*slog.Logger]
}

// Run waits until ctx ends.
func (s *Sender) Run(ctx context.Context) { <-ctx.Done() }

// Logger returns a dependency.
func (s *Sender) Logger() proc.UncheckedDependency[*slog.Logger] { return s.Args().Log }

// Arguments returns the arguments.
func (s *Sender) Arguments() SenderArgs { return s.Args() }

// copyLog calls Value on a local copy of the arguments.
func (s *Sender) copyLog() *slog.Logger {
	args := s.Args()
	return args.Log.Value()
}

// LogState is a Scoped struct with a dependency.
type LogState struct {
	im.Scoped
	Log proc.UncheckedDependency[*slog.Logger]
}

// Keeper publishes a dependency through an Owner field.
type Keeper struct {
	proc.Process[proc.None]
	state im.Owner[LogState]
}

// Run waits until ctx ends.
func (k *Keeper) Run(ctx context.Context) { <-ctx.Done() }

// Envelope stores process arguments in another marked struct.
type Envelope struct {
	im.Shared
	Args SenderArgs
}

// loggers is a frozen collection of dependencies.
var loggers im.FrozenSlice[proc.UncheckedDependency[*slog.Logger]]

// handlers keeps a dependency in an unmarked struct.
type handlers struct {
	log proc.UncheckedDependency[*slog.Logger]
}

// unwrap calls Value outside a process method.
func unwrap(log proc.UncheckedDependency[*slog.Logger]) *slog.Logger { return log.Value() }
