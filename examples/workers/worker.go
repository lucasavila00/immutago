// Package workers runs versioned text jobs in supervised child pools.
package workers

import (
	"context"
	"errors"
	"strings"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// WorkerArgs selects one worker and its text conversion version.
type WorkerArgs struct {
	im.Shared
	ID      int
	Version int
}

// Result records a completed job. Available is false if the worker has stopped.
type Result struct {
	im.ByValue
	Text      im.FrozenString
	Worker    int
	Version   int
	Available bool
}

// job carries ordinary text and a private reply channel to a worker.
type job struct {
	text  string
	reply chan string
}

// Worker converts queued text: version 1 uses uppercase; version 2 uses lowercase.
type Worker struct {
	supervise.Service[WorkerArgs]
	jobs    chan job
	ready   chan struct{}
	stopped chan struct{}
}

// Init validates the conversion version and creates the worker's channels.
func (w *Worker) Init() error {
	if w.Args().Version != 1 && w.Args().Version != 2 {
		return errors.New("text version must be 1 or 2")
	}
	w.jobs = make(chan job)
	w.ready = make(chan struct{})
	w.stopped = make(chan struct{})
	return nil
}

// Run receives jobs and returns their converted text until shutdown.
func (w *Worker) Run(ctx context.Context) {
	version := w.Args().Version
	close(w.ready)
	defer close(w.stopped)
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-w.jobs:
			text := strings.ToUpper(request.text)
			if version == 2 {
				text = strings.ToLower(request.text)
			}
			request.reply <- text
		}
	}
}

// Ready reports whether the worker can receive jobs.
func (w *Worker) Ready() bool {
	select {
	case <-w.stopped:
		return false
	default:
	}
	select {
	case <-w.ready:
		return true
	default:
		return false
	}
}

// Process submits text and waits for its result or worker shutdown.
func (w *Worker) Process(text string) Result {
	if !w.Ready() {
		return Result{}
	}
	reply := make(chan string, 1)
	select {
	case w.jobs <- job{text: text, reply: reply}:
	case <-w.stopped:
		return Result{}
	}
	select {
	case result := <-reply:
		return Result{Text: im.FreezeString(result), Worker: w.Args().ID, Version: w.Args().Version, Available: true}
	case <-w.stopped:
		return Result{}
	}
}
