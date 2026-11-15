package rollout

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// ReaderArgs selects one append-only log and its deployment's update channel.
type ReaderArgs struct {
	Files   fs.FS
	Path    string
	Every   time.Duration
	Updates chan<- struct{}
	Log     *log.Logger
}

// Reader owns one open log stream. A retry replays the file into replacement totals.
type Reader struct {
	supervise.Service[ReaderArgs]
	current im.Cell[Totals]
}

// Init reads the initial totals and closes its temporary file before returning.
// A staged child can stop without Run, so Init does not retain an open file.
func (r *Reader) Init() error {
	args := r.Args()
	file, err := args.Files.Open(args.Path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	scan := logScan{}
	if err := scan.read(context.Background(), bufio.NewReader(file), args); err != nil {
		return err
	}
	r.current.Store(scan.totals)
	return nil
}

// Run retains the stream position between reads. I/O failure closes the stream
// and asks the runtime to retry this reader. Invalid rows do not cause a retry.
func (r *Reader) Run(ctx context.Context) {
	args := r.Args()
	file, err := args.Files.Open(args.Path)
	if err != nil {
		panic(err)
	}
	defer func() { _ = file.Close() }()
	args.Log.Printf("reader started: %s", args.Path)
	defer args.Log.Printf("reader stopped: %s", args.Path)
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	stream := bufio.NewReader(file)
	scan := logScan{}
	notify(args.Updates)
	for {
		if err := scan.read(ctx, stream, args); err != nil {
			if ctx.Err() != nil {
				return
			}
			panic(fmt.Errorf("read %s: %w", args.Path, err))
		}
		if scan.totals != r.current.LoadShared() {
			r.current.Store(scan.totals)
			notify(args.Updates)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Read returns an independent value. Callers replace the previous replica totals.
func (r *Reader) Read() Totals { return r.current.LoadShared() }

// logScan retains an incomplete final row until its newline arrives.
type logScan struct {
	totals  Totals
	partial string
	line    int
}

// read consumes complete rows through the current end of the append-only stream.
func (s *logScan) read(ctx context.Context, reader *bufio.Reader, args ReaderArgs) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		part, err := reader.ReadString('\n')
		s.partial += part
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		s.line++
		if err := s.totals.addRecord(s.partial); err != nil {
			args.Log.Printf("invalid record %s:%d: %v", args.Path, s.line, err)
		}
		s.partial = ""
	}
}

// notify coalesces updates. Receivers read complete current totals after a signal.
func notify(updates chan<- struct{}) {
	select {
	case updates <- struct{}{}:
	default:
	}
}
