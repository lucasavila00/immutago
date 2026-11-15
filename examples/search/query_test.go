package search

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	"github.com/lucasavila00/immutago/pkg/im"
)

// TestTokenize checks case, punctuation, digits, and Unicode word positions.
func TestTokenize(t *testing.T) {
	text := "Deploy_2026 CAFÉ—naïve 日本語 K8s"
	words := tokenize(text)
	var actual []string
	for _, word := range words {
		actual = append(actual, word.text)
		if !strings.HasPrefix(strings.ToLower(text[word.start:]), word.text) {
			t.Fatalf("wrong start for %q: %d", word.text, word.start)
		}
	}
	want := []string{"deploy", "2026", "café", "naïve", "日本語", "k8s"}
	if !slices.Equal(actual, want) {
		t.Fatalf("tokens = %q, want %q", actual, want)
	}
}

// TestSearchRanking checks all-term matching, title weight, and repeated terms.
func TestSearchRanking(t *testing.T) {
	files := fstest.MapFS{
		"a.md": {Data: []byte("# Incident response\nIncident response and recovery.")},
		"b.md": {Data: []byte("# Response checklist\nIncident response response.")},
		"c.md": {Data: []byte("# Operations\nIncident only.")},
	}
	response := runQuery(t, files, "INCIDENT response incident", "")
	if response.Version != 1 || response.Total != 2 || len(response.Results) != 2 {
		t.Fatalf("response = %+v", response)
	}
	first := response.Results[0]
	second := response.Results[1]
	if first.Path != "a.md" || first.Score != 8 || second.Path != "b.md" || second.Score != 6 {
		t.Fatalf("ranked results = %+v", response.Results)
	}
	if first.Title != "Incident response" || first.Snippet != "Incident response and recovery." {
		t.Fatalf("first result text = %+v", first)
	}
	missing := runQuery(t, files, "incident absent", "")
	if missing.Total != 0 || missing.Results == nil {
		t.Fatalf("missing term response = %+v", missing)
	}
}

// TestStablePathOrder checks ties across nested paths and limits after ranking.
func TestStablePathOrder(t *testing.T) {
	files := fstest.MapFS{
		"a/z.md": {Data: []byte("# Notes\nrecovery")},
		"a.md":   {Data: []byte("# Notes\nrecovery")},
		"b.md":   {Data: []byte("# Notes\nrecovery")},
	}
	for i := 0; i < 20; i++ {
		response := runQuery(t, files, "recovery", "2")
		if response.Total != 3 || len(response.Results) != 2 ||
			response.Results[0].Path != "a.md" || response.Results[1].Path != "a/z.md" {
			t.Fatalf("unstable order: %+v", response)
		}
	}
}

// TestQueryBounds checks errors before index reads begin.
func TestQueryBounds(t *testing.T) {
	queries := []string{
		"", "!!!", "one two three four five six seven eight nine", strings.Repeat("a", 257), "\xff",
	}
	for _, text := range queries {
		if _, err := parseQuery(text, ""); err == nil {
			t.Errorf("accepted query %q", text)
		}
	}
	for _, limit := range []string{"0", "21", "many"} {
		if _, err := parseQuery("recovery", limit); err == nil {
			t.Errorf("accepted limit %q", limit)
		}
	}
}

// TestExcerpt checks bounded Unicode text near a match and title-only results.
func TestExcerpt(t *testing.T) {
	text := strings.Repeat("日本語 ", 80) + "recovery " + strings.Repeat("café ", 80)
	value := excerpt(im.ProjectString(text), strings.Index(text, "recovery"))
	if !utf8.ValidString(value) || !strings.Contains(value, "recovery") || utf8.RuneCountInString(value) > 162 {
		t.Fatalf("bad excerpt: %q", value)
	}
	if !strings.HasPrefix(value, "…") || !strings.HasSuffix(value, "…") {
		t.Fatalf("missing excerpt marks: %q", value)
	}
	files := fstest.MapFS{"guide.md": {Data: []byte("# Recovery\nA short guide.")}}
	response := runQuery(t, files, "recovery", "")
	if response.Results[0].Snippet != "A short guide." {
		t.Fatalf("title-only excerpt = %q", response.Results[0].Snippet)
	}
}

// TestDocumentParsing checks title fallback and the documented source-text rules.
func TestDocumentParsing(t *testing.T) {
	files := fstest.MapFS{
		"notes.MD":    {Data: []byte("Deploy\n\n`code` and [links](target).")},
		"ignore.txt":  {Data: []byte("hidden")},
		"folder/a.md": {Data: []byte("# Café\n\nRestore\tservice.")},
		"shortcut.md": {Mode: fs.ModeSymlink, Data: []byte("notes.MD")},
	}
	documents, err := scanDocuments(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 2 || documents[0].title != "Café" || documents[0].body != "Restore service." ||
		documents[1].title != "notes" || documents[1].body != "Deploy `code` and [links](target)." {
		t.Fatalf("parsed documents = %+v", documents)
	}
}

// runQuery builds a complete test index and runs a validated query.
func runQuery(t *testing.T, files fs.FS, text, limit string) Response {
	t.Helper()
	documents, err := scanDocuments(files)
	if err != nil {
		t.Fatal(err)
	}
	request, err := parseQuery(text, limit)
	if err != nil {
		t.Fatal(err)
	}
	var current IndexCell
	current.Store(buildIndex(documents, 1))
	return searchIndex(current.Load(), request)
}
