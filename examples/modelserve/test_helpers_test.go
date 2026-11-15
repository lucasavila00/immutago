package modelserve

import (
	"bytes"
	"io/fs"
	"log"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// testFiles permits concurrent edits and unexpected dependency failures.
type testFiles struct {
	mu       sync.Mutex
	data     map[string][]byte
	failures map[string]int
	reads    map[string]int
}

// sampleFiles supplies the same two models as the runnable application.
func sampleFiles() *testFiles {
	return &testFiles{data: map[string][]byte{
		"primary.json":    []byte(`{"model":"baseline.json"}`),
		"challenger.json": []byte(`{"model":"candidate.json"}`),
		"baseline.json":   []byte(`{"version":"baseline-v1","base":25000,"per_square_meter":2200,"per_room":8000}`),
		"candidate.json":  []byte(`{"version":"candidate-v1","base":30000,"per_square_meter":2100,"per_room":11000}`),
	}, failures: make(map[string]int), reads: make(map[string]int)}
}

// Open supplies the fs.FS method; file reads use the atomic ReadFile method below.
func (f *testFiles) Open(name string) (fs.File, error) {
	data, err := f.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return fstest.MapFS{name: &fstest.MapFile{Data: data}}.Open(name)
}

// ReadFile returns independent data and can fail an attempt with an unexpected panic.
func (f *testFiles) ReadFile(name string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads[name]++
	if n := f.failures[name]; n != 0 {
		if n > 0 {
			f.failures[name]--
		}
		panic("filesystem failure: " + name)
	}
	data, ok := f.data[name]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return bytes.Clone(data), nil
}

// set publishes one complete file edit.
func (f *testFiles) set(name, data string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[name] = []byte(data)
}

// fail schedules a fixed number of panics, or repeated panics when count is negative.
func (f *testFiles) fail(name string, count int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures[name] = count
}

// count reads a file's access count without racing its poller.
func (f *testFiles) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads[name]
}

// eventLog records lifecycle messages and can hold one cleanup operation open.
type eventLog struct {
	mu      sync.Mutex
	lines   []string
	hold    string
	entered chan struct{}
	release chan struct{}
}

// Write records an event before it waits for a deliberately slow log destination.
func (l *eventLog) Write(data []byte) (int, error) {
	line := strings.TrimSpace(string(data))
	l.mu.Lock()
	l.lines = append(l.lines, line)
	block := l.hold != "" && strings.Contains(line, l.hold)
	if block {
		l.hold = ""
		close(l.entered)
	}
	l.mu.Unlock()
	if block {
		<-l.release
	}
	return len(data), nil
}

// blockCleanup makes the next matching shutdown message wait until release.
func (l *eventLog) blockCleanup(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hold, l.entered, l.release = message, make(chan struct{}), make(chan struct{})
}

// snapshot copies events for assertions outside the writer's lock.
func (l *eventLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.lines...)
}

// newTestRouter starts the real services with a synchronized filesystem and log.
func newTestRouter(t *testing.T, files *testFiles, events *eventLog) (*Router, *supervise.Group) {
	t.Helper()
	root := supervise.NewRoot(nil)
	router, err := NewRouter(root, files, 10*time.Millisecond, log.New(events, "", 0))
	if err != nil {
		<-root.Stop()
		t.Fatal(err)
	}
	return router, root
}

// readPrediction selects the model through the same path as the HTTP handler.
func readPrediction(t *testing.T, router *Router, name string) Prediction {
	t.Helper()
	model, err := router.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Predict(model, Features{Area: 90, Rooms: 3})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
