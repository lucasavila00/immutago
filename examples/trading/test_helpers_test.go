package trading

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

const candidateConfig = `{
 "active":{"id":"wide","account":"desk-a","max_reserved_cash":1000,"parameters":{"edge":3,"skew":1},"markets":[{"symbol":"AAA"},{"symbol":"BBB"}]},
 "candidate":{"id":"tight","account":"desk-b","max_reserved_cash":1000,"parameters":{"edge":1,"skew":1},"markets":[{"symbol":"AAA"},{"symbol":"BBB"}]}
}`

const promotedConfig = `{"active":{"id":"tight","account":"desk-b","max_reserved_cash":1000,"parameters":{"edge":1,"skew":1},"markets":[{"symbol":"AAA"},{"symbol":"BBB"}]}}`

// testFiles supplies operator configuration and controlled read failures.
type testFiles struct {
	mu     sync.Mutex
	data   map[string][]byte
	errors map[string]int
	panics map[string]int
}

// sampleFiles reads the same data used by the command before a test starts.
func sampleFiles(t *testing.T) *testFiles {
	t.Helper()
	f := &testFiles{data: map[string][]byte{}, errors: map[string]int{}, panics: map[string]int{}}
	entries, err := os.ReadDir("data")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("data", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		f.data[entry.Name()] = data
	}
	return f
}

// Open creates an independent offset into a configuration file.
func (f *testFiles) Open(name string) (fs.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[name]; !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &testFile{files: f, name: name}, nil
}

// set replaces an operator configuration file.
func (f *testFiles) set(name, data string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[name] = []byte(data)
}

// fail makes the next count reads fail; a negative count keeps failing.
func (f *testFiles) fail(name string, count int, panicRead bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if panicRead {
		f.panics[name] = count
	} else {
		f.errors[name] = count
	}
}

// testFile reads one configuration file with its own offset.
type testFile struct {
	files  *testFiles
	name   string
	offset int
}

// Read supplies stream data or the selected I/O failure.
func (f *testFile) Read(dst []byte) (int, error) {
	f.files.mu.Lock()
	defer f.files.mu.Unlock()
	if count := f.files.panics[f.name]; count != 0 {
		f.files.panics[f.name]--
		panic("file dependency panic: " + f.name)
	}
	if count := f.files.errors[f.name]; count != 0 {
		f.files.errors[f.name]--
		return 0, errors.New("stream I/O failure: " + f.name)
	}
	data := f.files.data[f.name]
	if f.offset >= len(data) {
		return 0, io.EOF
	}
	n := copy(dst, data[f.offset:])
	f.offset += n
	return n, nil
}

// Close finishes the in-memory configuration read.
func (f *testFile) Close() error { return nil }

// Stat supplies the current stream size to fs.ReadFile.
func (f *testFile) Stat() (fs.FileInfo, error) {
	f.files.mu.Lock()
	defer f.files.mu.Unlock()
	return testInfo{name: f.name, size: len(f.files.data[f.name])}, nil
}

type testInfo struct {
	name string
	size int
}

// Name returns the fixture path.
func (i testInfo) Name() string { return i.name }

// Size returns the current byte count.
func (i testInfo) Size() int64 { return int64(i.size) }

// Mode reports an ordinary readable file.
func (testInfo) Mode() fs.FileMode { return 0600 }

// ModTime returns the unused fixture timestamp.
func (testInfo) ModTime() time.Time { return time.Time{} }

// IsDir identifies these fixtures as files.
func (testInfo) IsDir() bool { return false }

// Sys has no operating-system metadata.
func (testInfo) Sys() any { return nil }

// eventLog records application events and can fail one normal logging dependency call.
type eventLog struct {
	mu     sync.Mutex
	lines  []string
	failAt string
	output io.Writer
}

// Write records a log line before an optional one-time dependency panic.
func (e *eventLog) Write(data []byte) (int, error) {
	line := strings.TrimSpace(string(data))
	e.mu.Lock()
	e.lines = append(e.lines, line)
	fail := e.failAt == line
	if fail {
		e.failAt = ""
	}
	e.mu.Unlock()
	if e.output != nil {
		if _, err := e.output.Write(data); err != nil {
			return 0, err
		}
	}
	if fail {
		panic("log dependency panic: " + line)
	}
	return len(data), nil
}

// failOnce selects a real service call on which the test logger will panic.
func (e *eventLog) failOnce(line string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.failAt = line
}

// count counts exact application events.
func (e *eventLog) count(line string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	count := 0
	for _, item := range e.lines {
		if item == line {
			count++
		}
	}
	return count
}

// snapshot copies the log for ordered shutdown checks.
func (e *eventLog) snapshot() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.lines...)
}

// startBot starts the complete application against the deterministic venue fake.
func startBot(t *testing.T, files fs.FS, venue *fakeVenue, events *eventLog) *Bot {
	t.Helper()
	bot, err := Start(Options{API: venue, Files: files, Every: 10 * time.Millisecond, Log: log.New(events, "", 0)})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return bot
}

// advance permits policy reads and periodic reports to finish.
func advance() {
	time.Sleep(30 * time.Millisecond)
	synctest.Wait()
}
