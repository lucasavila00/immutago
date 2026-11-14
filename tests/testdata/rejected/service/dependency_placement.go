package service

import (
	"context"
	"log/slog"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// SenderArgs are valid service arguments with a dependency.
type SenderArgs struct {
	im.Shared
	Log supervise.ConcurrentSafe[*slog.Logger]
}

// Sender stores a dependency in a service field.
type Sender struct {
	supervise.Service[SenderArgs]
	log supervise.ConcurrentSafe[*slog.Logger]
}

// Run waits until ctx ends.
func (s *Sender) Run(ctx context.Context) { <-ctx.Done() }

// Logger returns a dependency.
func (s *Sender) Logger() supervise.ConcurrentSafe[*slog.Logger] { return s.Args().Log }

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
	Log supervise.ConcurrentSafe[*slog.Logger]
}

// Keeper publishes a dependency through an Owner field.
type Keeper struct {
	supervise.Service[supervise.NoArgs]
	state im.Owner[LogState]
}

// Run waits until ctx ends.
func (k *Keeper) Run(ctx context.Context) { <-ctx.Done() }

// Envelope stores service arguments in another marked struct.
type Envelope struct {
	im.Shared
	Args SenderArgs
}

// loggers is a frozen collection of dependencies.
var loggers im.FrozenSlice[supervise.ConcurrentSafe[*slog.Logger]]

// handlers keeps a dependency in an unmarked struct.
type handlers struct {
	log supervise.ConcurrentSafe[*slog.Logger]
}

// unwrap calls Value outside a service method.
func unwrap(log supervise.ConcurrentSafe[*slog.Logger]) *slog.Logger { return log.Value() }
