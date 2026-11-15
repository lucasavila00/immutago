// Command rollout compares release health from append-only replica access logs.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucasavila00/immutago/examples/rollout"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// main reads command options and waits for an interrupt.
func main() {
	directory := flag.String("dir", "examples/rollout/data", "directory with rollout.json, policy.json, and access logs")
	address := flag.String("addr", "127.0.0.1:8082", "HTTP listen address")
	every := flag.Duration("every", time.Second, "interval between file reads")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, *directory, *address, *every); err != nil {
		log.Fatal(err)
	}
}

// run starts the simple dependency first, then the nested consumer tree.
// The deferred root stop runs after HTTP admission closes and active requests drain.
func run(ctx context.Context, directory, address string, every time.Duration) error {
	logger := log.Default()
	root := supervise.NewRoot(func(value any, _ []byte) { logger.Print(value) })
	defer func() { <-root.Stop() }()
	files := os.DirFS(directory)
	policy, err := supervise.Start[rollout.Policy](root, rollout.PolicyArgs{Files: files, Every: every, Log: logger})
	if err != nil {
		return err
	}
	ref, err := supervise.Start[rollout.Rollout](root, rollout.Args{
		Files: files, Every: every, Log: logger, Policy: policy,
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	logger.Printf("release gate: http://%s/report", listener.Addr())
	return serve(ctx, listener, rollout.HTTPHandler(ref))
}

// serve closes ingress and gives active requests up to five seconds to finish.
func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
		err := <-done
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
