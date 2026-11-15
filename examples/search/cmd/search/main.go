// Command search serves a local Markdown directory over HTTP.
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

	"github.com/lucasavila00/immutago/examples/search"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

func main() {
	directory := flag.String("dir", "examples/search/documents", "directory with Markdown files")
	address := flag.String("addr", "127.0.0.1:8081", "HTTP listen address")
	every := flag.Duration("every", time.Second, "interval between directory scans")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, *directory, *address, *every); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, directory, address string, every time.Duration) error {
	root := supervise.NewRoot(func(value any, _ []byte) { log.Print(value) })
	defer func() { <-root.Stop() }()
	indexer, err := supervise.Start[search.Indexer](root, search.IndexerArgs{
		Files: os.DirFS(directory), Every: every,
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: search.HTTPHandler(indexer), ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	log.Printf("document search: http://%s/search?q=incident", listener.Addr())
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
