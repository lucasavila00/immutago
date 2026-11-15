package models_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/examples/models"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

const firstModel = `{"name":"first","weights":[2,3]}`
const secondModel = `{"name":"second","weights":[4,1]}`

// source serves a model that the test can replace without racing HTTP handlers.
type source struct {
	mu       sync.Mutex
	body     string
	status   int
	requests int
}

// serve copies one response under the lock, then writes it to the HTTP client.
func (s *source) serve(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	body, status := s.body, s.status
	s.requests++
	s.mu.Unlock()
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

// replace changes the next response and returns the count before that change.
func (s *source) replace(body string, status int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.body, s.status = body, status
	return s.requests
}

// count returns the number of HTTP requests received so far.
func (s *source) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

// startFetcher passes a safe client, logger, and store factory through explicit promises.
func startFetcher(
	t *testing.T, server *httptest.Server, factory func(string) models.Store,
) supervise.Ref[models.Fetcher] {
	t.Helper()
	root := supervisetest.NewRoot(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	args := models.FetcherArgs{
		URL: server.URL, Every: 5 * time.Millisecond,
		Log:      log,
		Client:   server.Client(),
		NewStore: factory,
	}
	fetcher, err := supervise.Start[models.Fetcher](root, args)
	if err != nil {
		t.Fatal(err)
	}
	return fetcher
}

// TestFetcherRefresh checks real HTTP refreshes, saved documents, and failed-load retention.
func TestFetcherRefresh(t *testing.T) {
	src := &source{body: firstModel, status: http.StatusOK}
	server := httptest.NewServer(http.HandlerFunc(src.serve))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "model.json")
	factory := func(string) models.Store { return models.FileStore{Path: path} }
	fetcher := startFetcher(t, server, factory)
	assertScore(t, fetcher, 13)
	assertSaved(t, path, firstModel)
	src.replace(secondModel, http.StatusOK)
	waitName(t, fetcher, "second")
	assertScore(t, fetcher, 11)
	assertSaved(t, path, secondModel)

	before := src.replace("unavailable", http.StatusServiceUnavailable)
	waitRequests(t, src, before+2)
	assertScore(t, fetcher, 11)
	assertSaved(t, path, secondModel)
	before = src.replace(`{"name":"broken","weights":`, http.StatusOK)
	waitRequests(t, src, before+2)
	assertScore(t, fetcher, 11)
	assertSaved(t, path, secondModel)
}

// assertScore checks the public service API with a fixed two-feature prediction.
func assertScore(t *testing.T, fetcher supervise.Ref[models.Fetcher], want float64) {
	t.Helper()
	features := im.FreezeSlice([]float64{2, 3})
	prediction := fetcher.Current().Predict(features)
	if prediction.Score != want {
		t.Fatalf("prediction = %v, want %v", prediction.Score, want)
	}
}

// assertSaved checks the document written by the real FileStore backend.
func assertSaved(t *testing.T, path, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("saved model = %q, want %q", data, want)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitName waits for the asynchronous loader to publish a specific model.
func waitName(t *testing.T, fetcher supervise.Ref[models.Fetcher], want string) {
	t.Helper()
	features := im.FreezeSlice([]float64{2, 3})
	deadline := time.Now().Add(5 * time.Second)
	for {
		prediction := fetcher.Current().Predict(features)
		if prediction.Name.Clone() == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("model %q was not published", want)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitRequests waits for two serial refresh attempts, so the first has finished.
func waitRequests(t *testing.T, src *source, count int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for src.count() < count {
		if time.Now().After(deadline) {
			t.Fatalf("HTTP requests = %d, want at least %d", src.count(), count)
		}
		time.Sleep(time.Millisecond)
	}
}

// recordingStore is a thread-safe test backend shared by two independent fetchers.
type recordingStore struct {
	mu       sync.Mutex
	document string
	writes   int
	failing  bool
}

// Save records each validated document under a lock.
func (s *recordingStore) Save(document string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing {
		return fmt.Errorf("store is unavailable")
	}
	s.document = document
	s.writes++
	return nil
}

// fail makes later saves return an error without replacing the stored document.
func (s *recordingStore) fail() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failing = true
	return s.writes
}

// read returns the current saved document and write count.
func (s *recordingStore) read() (string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.document, s.writes
}

// TestConcurrentReaders checks that each prediction uses one whole model version.
func TestConcurrentReaders(t *testing.T) {
	src := &source{body: firstModel, status: http.StatusOK}
	server := httptest.NewServer(http.HandlerFunc(src.serve))
	defer server.Close()
	store := &recordingStore{}
	factory := func(string) models.Store { return store }
	first := startFetcher(t, server, factory)
	second := startFetcher(t, server, factory)
	ctx, cancel := context.WithCancel(context.Background())
	failures := make(chan string, 1)
	var readers sync.WaitGroup
	defer func() {
		cancel()
		readers.Wait()
	}()
	for range 4 {
		readers.Go(func() { readScores(first, ctx.Done(), failures) })
		readers.Go(func() { readScores(second, ctx.Done(), failures) })
	}
	src.replace(secondModel, http.StatusOK)
	waitName(t, first, "second")
	waitName(t, second, "second")
	cancel()
	readers.Wait()
	select {
	case result := <-failures:
		t.Fatalf("mixed-version prediction = %s", result)
	default:
	}
	document, writes := store.read()
	if document != secondModel || writes < 4 {
		t.Fatalf("store got %q after %d writes", document, writes)
	}
}

// readScores reports any prediction that does not belong to one complete version.
func readScores(fetcher supervise.Ref[models.Fetcher], done <-chan struct{}, failures chan<- string) {
	features := im.FreezeSlice([]float64{2, 3})
	for {
		select {
		case <-done:
			return
		default:
			prediction := fetcher.Current().Predict(features)
			validFirst := prediction.Name.Clone() == "first" && prediction.Score == 13
			validSecond := prediction.Name.Clone() == "second" && prediction.Score == 11
			if !validFirst && !validSecond {
				message := fmt.Sprintf("%s:%v", prediction.Name.Clone(), prediction.Score)
				select {
				case failures <- message:
				default:
				}
			}
		}
	}
}

// TestStoreFailureRetainsModel checks that failed persistence prevents publication.
func TestStoreFailureRetainsModel(t *testing.T) {
	src := &source{body: firstModel, status: http.StatusOK}
	server := httptest.NewServer(http.HandlerFunc(src.serve))
	defer server.Close()
	store := &recordingStore{}
	factory := func(string) models.Store { return store }
	fetcher := startFetcher(t, server, factory)
	saved := store.fail()
	before := src.replace(secondModel, http.StatusOK)
	waitRequests(t, src, before+2)
	assertScore(t, fetcher, 13)
	document, writes := store.read()
	if document != firstModel || writes != saved {
		t.Fatalf("store changed after failure: %q, %d writes", document, writes)
	}
}
