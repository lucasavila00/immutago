package models_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/examples/models"
	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestStopCancelsRefresh checks that shutdown cancels HTTP work already in progress.
func TestStopCancelsRefresh(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, firstModel)
			return
		}
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer server.Close()
	root := supervisetest.NewRoot(t)
	store := &recordingStore{}
	factory := func(string) models.Store { return store }
	client := server.Client()
	client.Timeout = 10 * time.Second
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	args := models.FetcherArgs{
		URL: server.URL, Every: time.Millisecond,
		Log:      log,
		Client:   client,
		NewStore: factory,
	}
	_, err := supervise.Start[models.Fetcher](root, args)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("background HTTP request did not start")
	}
	stopped := root.Stop()
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not cancel the HTTP request")
	}
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("loader did not stop after request cancellation")
	}
}
