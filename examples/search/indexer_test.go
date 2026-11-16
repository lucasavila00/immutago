package search

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestRefreshFiles checks edits, additions, deletions, and unchanged scans.
func TestRefreshFiles(t *testing.T) {
	synctest.Test(t, testRefreshFiles)
}

// testRefreshFiles edits files between scans while fake time controls the poller.
func testRefreshFiles(t *testing.T) {
	directory := t.TempDir()
	writeDocument(t, directory, "guide.md", "# Recovery\nold procedure")
	root := supervisetest.NewRoot(t)
	indexer := startIndexer(t, root, os.DirFS(directory))
	handler := HTTPHandler(indexer)
	initial := readResponse(t, handler, "/search?q=old")
	if initial.Version != 1 || initial.Total != 1 {
		t.Fatalf("initial response = %+v", initial)
	}
	synctest.Wait()
	writeDocument(t, directory, "guide.md", "# Recovery\nnew procedure")
	awaitStatus(t, indexer, 2, false)
	if old := readResponse(t, handler, "/search?q=old"); old.Total != 0 {
		t.Fatalf("old content remains: %+v", old)
	}
	synctest.Wait()
	writeDocument(t, directory, "extra.md", "# Recovery\nnew checklist")
	awaitStatus(t, indexer, 3, false)
	if added := readResponse(t, handler, "/search?q=new"); added.Total != 2 {
		t.Fatalf("added document missing: %+v", added)
	}
	synctest.Wait()
	if err := os.Remove(filepath.Join(directory, "guide.md")); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, indexer, 4, false)
	remaining := readResponse(t, handler, "/search?q=new")
	if remaining.Total != 1 || remaining.Results[0].Path != "extra.md" {
		t.Fatalf("deleted document remains: %+v", remaining)
	}
	time.Sleep(30 * time.Millisecond)
	if unchanged := indexer.Borrow().Status(); unchanged.Version != 4 {
		t.Fatalf("unchanged files published version %d", unchanged.Version)
	}
	synctest.Wait()
	if err := os.Remove(filepath.Join(directory, "extra.md")); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, indexer, 5, false)
	if empty := readResponse(t, handler, "/search?q=new"); empty.Total != 0 {
		t.Fatalf("empty directory response = %+v", empty)
	}
}

// TestScanFailureRecovery keeps the last index through read and parse errors.
func TestScanFailureRecovery(t *testing.T) {
	synctest.Test(t, testScanFailureRecovery)
}

// testScanFailureRecovery changes files only after the poller has finished its scan.
func testScanFailureRecovery(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "documents")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	writeDocument(t, directory, "guide.md", "# Recovery\nvalid procedure")
	root := supervisetest.NewRoot(t)
	indexer := startIndexer(t, root, os.DirFS(directory))
	handler := HTTPHandler(indexer)
	synctest.Wait()
	if err := os.Rename(directory, directory+"-away"); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, indexer, 1, true)
	if response := readResponse(t, handler, "/search?q=valid"); response.Total != 1 || response.Version != 1 {
		t.Fatalf("scan error lost the index: %+v", response)
	}
	synctest.Wait()
	if err := os.Rename(directory+"-away", directory); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, indexer, 1, false)
	synctest.Wait()
	writeDocument(t, directory, "guide.md", "\xff")
	awaitStatus(t, indexer, 1, true)
	if response := readResponse(t, handler, "/search?q=valid"); response.Total != 1 {
		t.Fatalf("parse error lost the index: %+v", response)
	}
	synctest.Wait()
	writeDocument(t, directory, "guide.md", "# Recovery\nrepaired procedure")
	awaitStatus(t, indexer, 2, false)
	if response := readResponse(t, handler, "/search?q=repaired"); response.Total != 1 {
		t.Fatalf("recovery did not publish: %+v", response)
	}
}

// TestConcurrentRefresh keeps each query on one version while a scan is paused.
func TestConcurrentRefresh(t *testing.T) {
	directory := t.TempDir()
	writeDocument(t, directory, "a.md", "# Release\nalpha versionone")
	writeDocument(t, directory, "b.md", "# Release\nbeta versionone")
	files := &pausedFS{FS: os.DirFS(directory)}
	root := supervisetest.NewRoot(t)
	indexer := startIndexer(t, root, files)
	handler := HTTPHandler(indexer)
	selected := indexer.Borrow().Read()
	firstEntered, firstRelease := files.pause("a.md")
	resumeFirst := sync.OnceFunc(func() { close(firstRelease) })
	defer resumeFirst()
	awaitRead(t, firstEntered)
	writeDocument(t, directory, "a.md", "# Release\nalpha versiontwo")
	writeDocument(t, directory, "b.md", "# Release\nbeta versiontwo")
	entered, release := files.pause("b.md")
	resume := sync.OnceFunc(func() { close(release) })
	defer resume()
	resumeFirst()
	awaitRead(t, entered)
	if response := readResponse(t, handler, "/search?q=release"); response.Version != 1 {
		t.Fatalf("published an incomplete scan: %+v", response)
	}
	done := make(chan struct{})
	problems := make(chan error, 4)
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			problems <- checkConcurrentResponses(handler, done)
		}()
	}
	stopReaders := sync.OnceFunc(func() {
		close(done)
		readers.Wait()
	})
	defer stopReaders()
	resume()
	awaitStatus(t, indexer, 2, false)
	stopReaders()
	close(problems)
	for err := range problems {
		if err != nil {
			t.Error(err)
		}
	}
	request, err := parseQuery("release", "")
	if err != nil {
		t.Fatal(err)
	}
	old := searchIndex(selected, request)
	if old.Version != 1 || !strings.Contains(old.Results[0].Snippet, "versionone") {
		t.Fatalf("selected projection changed: %+v", old)
	}
	current := readResponse(t, handler, "/search?q=release")
	if current.Version != 2 || !strings.Contains(current.Results[0].Snippet, "versiontwo") {
		t.Fatalf("new query did not use the new index: %+v", current)
	}
}

// TestInitialFailure rejects unreadable files, invalid text, and an invalid interval.
func TestInitialFailure(t *testing.T) {
	directory := t.TempDir()
	writeDocument(t, directory, "bad.md", "\xff")
	root := supervisetest.NewRoot(t)
	for _, args := range []IndexerArgs{
		{Files: os.DirFS(filepath.Join(directory, "missing")), Every: time.Second},
		{Files: os.DirFS(directory), Every: time.Second},
		{Files: os.DirFS(directory), Every: 0},
	} {
		if _, err := supervise.Start[Indexer](root, args); err == nil {
			t.Error("startup accepted invalid input")
		}
	}
}

// TestGracefulStop waits for the indexer and checks that no later scan runs.
func TestGracefulStop(t *testing.T) {
	directory := t.TempDir()
	writeDocument(t, directory, "guide.md", "# Recovery\nprocedure")
	files := &pausedFS{FS: os.DirFS(directory)}
	root := supervisetest.NewRoot(t)
	indexer := startIndexer(t, root, files)
	select {
	case <-root.Stop():
	case <-time.After(time.Second):
		t.Fatal("indexer did not stop")
	}
	reads := files.reads.Load()
	time.Sleep(30 * time.Millisecond)
	if files.reads.Load() != reads {
		t.Fatal("file reads continued after stop")
	}
	if status := indexer.Borrow().Status(); status.Version != 1 {
		t.Fatalf("stop lost the last publication: version %d", status.Version)
	}
}

// startIndexer starts the real service with a short test interval.
func startIndexer(t *testing.T, root *supervise.Group, files fs.FS) supervise.Ref[Indexer] {
	t.Helper()
	indexer, err := supervise.Start[Indexer](root, IndexerArgs{Files: files, Every: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return indexer
}

// awaitStatus waits for one publication or error state.
func awaitStatus(t *testing.T, indexer supervise.Ref[Indexer], version int, problem bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status := indexer.Borrow().Status()
		if status.Version == version && (status.Problem.Len() != 0) == problem {
			return
		}
		time.Sleep(time.Millisecond)
	}
	status := indexer.Borrow().Status()
	t.Fatalf("status = version %d, problem %q; want version %d, problem present %t",
		status.Version, status.Problem.Unchecked(), version, problem)
}

// writeDocument replaces one complete file, as an editor's save operation can do.
func writeDocument(t *testing.T, directory, name, content string) {
	t.Helper()
	target := filepath.Join(directory, name)
	if err := os.WriteFile(target+".tmp", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(target+".tmp", target); err != nil {
		t.Fatal(err)
	}
}

// readResponse runs the real handler and decodes its ordinary JSON response.
func readResponse(t *testing.T, handler http.Handler, target string) Response {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status %d: %s", recorder.Code, recorder.Body.String())
	}
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

// checkConcurrentResponses detects mixed document versions during publication.
func checkConcurrentResponses(handler http.Handler, done <-chan struct{}) error {
	for {
		select {
		case <-done:
			return nil
		default:
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/search?q=release", nil))
		var response Response
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			return err
		}
		expected := "versionone"
		if response.Version == 2 {
			expected = "versiontwo"
		}
		if response.Total != 2 || len(response.Results) != 2 || response.Version < 1 || response.Version > 2 {
			return fmt.Errorf("incomplete response: %+v", response)
		}
		for _, result := range response.Results {
			if !strings.Contains(result.Snippet, expected) {
				return fmt.Errorf("mixed response: %+v", response)
			}
		}
	}
}

type pausedFS struct {
	fs.FS
	mu      sync.Mutex
	name    string
	entered chan struct{}
	release chan struct{}
	reads   atomic.Int64
}

// awaitRead waits for a scan to reach the requested file.
func awaitRead(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the scan did not reach the requested file")
	}
}

// pause stops the next read of name until the test releases it.
func (f *pausedFS) pause(name string) (<-chan struct{}, chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.name = name
	f.entered = make(chan struct{})
	f.release = make(chan struct{})
	return f.entered, f.release
}

// ReadFile pauses before the file read, so tests can complete edits first.
func (f *pausedFS) ReadFile(name string) ([]byte, error) {
	f.mu.Lock()
	entered := f.entered
	release := f.release
	if name == f.name {
		f.entered = nil
	} else {
		entered = nil
	}
	f.mu.Unlock()
	if entered != nil {
		close(entered)
		<-release
	}
	f.reads.Add(1)
	return fs.ReadFile(f.FS, name)
}
