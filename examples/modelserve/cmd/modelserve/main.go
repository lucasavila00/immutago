// Command modelserve compares local house-price models over HTTP.
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

	"github.com/lucasavila00/immutago/examples/modelserve"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// main reads options and stops ingress before the supervised model tree.
func main() {
	directory := flag.String("dir", "examples/modelserve/models", "directory with model manifests and coefficients")
	address := flag.String("addr", "127.0.0.1:8082", "HTTP listen address")
	every := flag.Duration("every", time.Second, "interval between file reads")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, *directory, *address, *every); err != nil {
		log.Fatal(err)
	}
}

// run drains HTTP requests before it stops model refresh and manifest polls.
func run(ctx context.Context, directory, address string, every time.Duration) error {
	logger := log.Default()
	root := supervise.NewRoot(func(value any, _ []byte) { logger.Printf("service failure: %v", value) })
	defer func() { <-root.Stop() }()
	router, err := modelserve.NewRouter(root, os.DirFS(directory), every, logger)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: modelserve.HTTPHandler(router), ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	logger.Printf("model serving: http://%s/status", listener.Addr())
	select {
	case err = <-done:
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if shutdownErr := server.Shutdown(shutdown); shutdownErr != nil {
		_ = server.Close()
		return shutdownErr
	}
	if err == nil {
		err = <-done
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
