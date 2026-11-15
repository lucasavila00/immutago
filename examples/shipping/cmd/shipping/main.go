// Command shipping serves delivery quotes from a local rate catalog.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/lucasavila00/immutago/examples/shipping"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// main selects the catalog and listen address and handles Ctrl+C.
func main() {
	path := flag.String("catalog", "examples/shipping/catalog.json", "rate catalog file")
	address := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	listener, err := net.Listen("tcp", *address)
	if err == nil {
		err = run(ctx, *path, listener)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run loads the catalog before serving and drains HTTP requests before stopping the loader.
func run(ctx context.Context, path string, listener net.Listener) error {
	defer func() { _ = listener.Close() }()
	root := supervise.NewRoot(nil)
	defer func() { <-root.Stop() }()
	loader, err := supervise.Start[shipping.Loader](root, shipping.Args{Path: path, Every: time.Second})
	if err != nil {
		return err
	}
	server := &http.Server{Handler: shipping.NewHandler(loader), ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fmt.Fprintf(os.Stderr, "shipping quotes: http://%s/quotes\n", listener.Addr())
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
