package rollout

import (
	"context"
	"io/fs"
	"log"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// DeploymentArgs supplies the release's streams and its parent's update channel.
type DeploymentArgs struct {
	Files   fs.FS
	Spec    deploymentSpec
	Every   time.Duration
	Updates chan<- struct{}
	Log     *log.Logger
}

// Deployment combines complete replica totals. Its readers belong to this attempt.
type Deployment struct {
	supervise.Service[DeploymentArgs]
	readers []supervise.Ref[Reader]
	updates chan struct{}
	current im.Cell[Totals]
}

// Init stages readers and computes the first report without waiting for their Run.
func (d *Deployment) Init() error {
	args := d.Args()
	d.updates = make(chan struct{}, 1)
	for _, path := range args.Spec.Logs {
		reader, err := supervise.Start[Reader](d.Children(), ReaderArgs{
			Files: args.Files, Path: path, Every: args.Every, Updates: d.updates, Log: args.Log,
		})
		if err != nil {
			return err
		}
		d.readers = append(d.readers, reader)
	}
	d.combine()
	return nil
}

// Run updates the release report when a reader publishes new totals or restarts.
func (d *Deployment) Run(ctx context.Context) {
	args := d.Args()
	args.Log.Printf("deployment started: %s", args.Spec.Version)
	defer args.Log.Printf("deployment stopped: %s", args.Spec.Version)
	notify(args.Updates)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.updates:
			if d.combine() {
				args.Log.Printf("deployment updated: %s", args.Spec.Version)
				notify(args.Updates)
			}
		}
	}
}

// Read returns the latest complete sum as independent numeric data.
func (d *Deployment) Read() Totals { return im.Value(d.current.Load()) }

// combine replaces the sum instead of adding deltas, so reader replay cannot double-count.
func (d *Deployment) combine() bool {
	var total Totals
	for _, reader := range d.readers {
		total.add(reader.Borrow().Read())
	}
	if total == im.Value(d.current.Load()) {
		return false
	}
	d.current.Store(total)
	return true
}
