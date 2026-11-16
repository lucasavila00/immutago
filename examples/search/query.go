package search

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lucasavila00/immutago/pkg/im"
)

// Response owns the text sent in one search response.
type Response struct {
	Version int      `json:"version"`
	Total   int      `json:"total"`
	Results []Result `json:"results"`
}

// Result is one ranked document with a short body excerpt.
type Result struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Score   int    `json:"score"`
	Snippet string `json:"snippet"`
}

type query struct {
	terms []string
	limit int
}

type candidate struct {
	document int
	score    int
	matched  int
	first    int
}

// parseQuery checks query bounds and removes repeated terms.
func parseQuery(text, limitText string) (query, error) {
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > 256 {
		return query{}, fmt.Errorf("q must contain at most 256 valid UTF-8 characters")
	}
	var terms []string
	seen := make(map[string]bool)
	for _, word := range tokenize(text) {
		if !seen[word.text] {
			terms = append(terms, word.text)
			seen[word.text] = true
		}
	}
	if len(terms) == 0 || len(terms) > 8 {
		return query{}, fmt.Errorf("q must contain 1 to 8 distinct words")
	}
	limit := 5
	if limitText != "" {
		value, err := strconv.Atoi(limitText)
		if err != nil || value < 1 || value > 20 {
			return query{}, fmt.Errorf("limit must be an integer from 1 to 20")
		}
		limit = value
	}
	return query{terms: terms, limit: limit}, nil
}

// searchIndex ranks documents that contain every term in the selected version.
func searchIndex(index IndexProjection, request query) Response {
	response := Response{Version: index.Version(), Results: []Result{}}
	candidates := make(map[int]candidate)
	for _, term := range request.terms {
		value, found := index.Terms().Get(term)
		if !found {
			return response
		}
		postings := im.SliceOf(value)
		for i := 0; i < postings.Len(); i++ {
			posting := im.Value(postings.At(i))
			id := posting.Document
			item, found := candidates[id]
			if !found {
				item = candidate{document: id, first: -1}
			}
			item.score += posting.Score
			item.matched++
			first := posting.First
			if first >= 0 && (item.first < 0 || first < item.first) {
				item.first = first
			}
			candidates[id] = item
		}
	}
	var ranked []candidate
	for _, item := range candidates {
		if item.matched == len(request.terms) {
			ranked = append(ranked, item)
		}
	}
	// The scan sorts paths before it assigns IDs, so IDs also resolve score ties.
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].document < ranked[j].document
	})
	response.Total = len(ranked)
	if len(ranked) > request.limit {
		ranked = ranked[:request.limit]
	}
	documents := index.Documents()
	for _, item := range ranked {
		document := NewDocumentProjection(documents.AtPointer(item.document))
		response.Results = append(response.Results, Result{
			Path: document.Path().Clone(), Title: document.Title().Clone(),
			Score: item.score, Snippet: excerpt(document.Body(), item.first),
		})
	}
	return response
}

// excerpt copies up to 160 characters near the first matching body word.
func excerpt(body im.StringProjection, first int) string {
	// Unchecked gives the standard string functions read access. The return value is copied.
	text := body.Unchecked()
	start := max(first, 0)
	for n := 0; n < 40 && start > 0; n++ {
		_, size := utf8.DecodeLastRuneInString(text[:start])
		start -= size
	}
	end := start
	for n := 0; n < 160 && end < len(text); n++ {
		_, size := utf8.DecodeRuneInString(text[end:])
		end += size
	}
	if start == 0 && end == len(text) {
		return strings.Clone(text)
	}
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	suffix := ""
	if end < len(text) {
		suffix = "…"
	}
	return prefix + text[start:end] + suffix
}
