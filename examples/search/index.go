// Package search serves ranked results from a local Markdown directory.
package search

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Index holds one complete document and token index.
type Index struct {
	im.Frozen
	Version   int
	Documents im.FrozenSlice[Document]
	Terms     im.FrozenMap[string, im.FrozenSlice[Posting]]
}

// Document contains the text used for search results.
type Document struct {
	im.Frozen
	Path  string
	Title string
	Body  string
}

// Posting holds one document's score and first body position for a token.
type Posting struct {
	im.Frozen
	Document int
	Score    int
	First    int
}

type sourceDocument struct {
	path  string
	title string
	body  string
}

type token struct {
	text  string
	start int
}

type termCount struct {
	score int
	first int
}

type sourcePosting struct {
	document int
	score    int
	first    int
}

func scanDocuments(files fs.FS) ([]sourceDocument, error) {
	var documents []sourceDocument
	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() || !strings.EqualFold(path.Ext(name), ".md") {
			return nil
		}
		data, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		document, err := parseDocument(name, data)
		if err != nil {
			return err
		}
		documents = append(documents, document)
		return nil
	})
	slices.SortFunc(documents, func(a, b sourceDocument) int { return strings.Compare(a.path, b.path) })
	return documents, err
}

func parseDocument(name string, data []byte) (sourceDocument, error) {
	if !utf8.Valid(data) {
		return sourceDocument{}, fmt.Errorf("parse %s: text is not valid UTF-8", name)
	}
	text := strings.TrimSpace(string(data))
	first, rest, _ := strings.Cut(text, "\n")
	title := strings.TrimSuffix(path.Base(name), path.Ext(name))
	if strings.HasPrefix(first, "# ") {
		title = strings.TrimSpace(strings.TrimPrefix(first, "# "))
		text = rest
	}
	return sourceDocument{
		path: name, title: title, body: strings.Join(strings.Fields(text), " "),
	}, nil
}

func tokenize(text string) []token {
	var tokens []token
	start := -1
	for offset, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = offset
			}
			continue
		}
		if start >= 0 {
			tokens = append(tokens, token{text: strings.ToLower(text[start:offset]), start: start})
			start = -1
		}
	}
	if start >= 0 {
		tokens = append(tokens, token{text: strings.ToLower(text[start:]), start: start})
	}
	return tokens
}

func buildIndex(sources []sourceDocument, version int) Index {
	documents := make([]Document, 0, len(sources))
	terms := make(map[string][]sourcePosting)
	for id, source := range sources {
		documents = append(documents, Document{
			Path: source.path, Title: source.title, Body: source.body,
		})
		counts := make(map[string]termCount)
		for _, word := range tokenize(source.title) {
			count := counts[word.text]
			count.score += 3
			count.first = -1
			counts[word.text] = count
		}
		for _, word := range tokenize(source.body) {
			count, found := counts[word.text]
			if !found || count.first < 0 {
				count.first = word.start
			}
			count.score++
			counts[word.text] = count
		}
		for term, count := range counts {
			terms[term] = append(terms[term], sourcePosting{
				document: id, score: count.score, first: count.first,
			})
		}
	}
	frozenDocuments := im.FreezeSlice(documents)
	entries := make(map[string]im.FrozenSlice[Posting], len(terms))
	for term, sources := range terms {
		postings := make([]Posting, 0, len(sources))
		for _, source := range sources {
			postings = append(postings, Posting{
				Document: source.document, Score: source.score, First: source.first,
			})
		}
		frozenPostings := im.FreezeSlice(postings)
		entries[term] = frozenPostings
	}
	frozenTerms := im.FreezeMap(entries)
	return Index{Version: version, Documents: frozenDocuments, Terms: frozenTerms}
}
