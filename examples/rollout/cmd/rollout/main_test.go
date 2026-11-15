package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestRun checks initial failure and shutdown of both root branches.
func TestRun(t *testing.T) {
	if err := run(context.Background(), t.TempDir(), "127.0.0.1:0", time.Hour); err == nil {
		t.Fatal("accepted a missing policy")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, "../../data", "127.0.0.1:0", time.Hour); err != nil {
		t.Fatal(err)
	}
}

// TestServeDrainsRequests waits for a live request after cancellation or an unexpected Serve error.
func TestServeDrainsRequests(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "accept_error"}[fail], func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			broken := &failingListener{Listener: listener, failed: make(chan struct{})}
			if fail {
				listener = broken
			}
			entered, release := make(chan struct{}), make(chan struct{})
			finish := sync.OnceFunc(func() { close(release) })
			defer finish()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- serve(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					close(entered)
					<-release
					_, _ = io.WriteString(w, "complete")
				}))
			}()
			response := make(chan error, 1)
			go func() { response <- fetch("http://" + listener.Addr().String()) }()
			<-entered
			if fail {
				broken.fail()
			} else {
				cancel()
			}
			select {
			case err := <-done:
				t.Fatalf("returned before the admitted request finished: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			finish()
			if err := <-response; err != nil {
				t.Fatal(err)
			}
			if err := <-done; (err != nil) != fail || (err != nil && !strings.Contains(err.Error(), "accept failed")) {
				t.Fatal(err)
			}
		})
	}
}

// fetch requires the active request to finish with its complete response.
func fetch(url string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err == nil && string(data) != "complete" {
		return errors.New("incomplete HTTP response")
	}
	return err
}

// failingListener makes Serve fail after it has admitted one connection.
type failingListener struct {
	net.Listener
	accepted bool
	failed   chan struct{}
	once     sync.Once
}

// Accept permits the first connection and then waits for the requested failure.
func (l *failingListener) Accept() (net.Conn, error) {
	if l.accepted {
		<-l.failed
		return nil, errors.New("accept failed")
	}
	l.accepted = true
	return l.Listener.Accept()
}

// fail releases the blocked Accept with an error.
func (l *failingListener) fail() { l.once.Do(func() { close(l.failed) }) }

// Close releases a pending Accept and closes the real listener.
func (l *failingListener) Close() error {
	l.fail()
	return l.Listener.Close()
}
