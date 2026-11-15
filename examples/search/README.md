# Document search

Search a local directory of Markdown notes or runbooks. The HTTP server returns
ranked documents, titles, and short text excerpts. One supervised indexer scans
the directory in the background. Each query reads one complete index version.

## Run

From the repository root:

```sh
go run ./examples/search/cmd/search
```

This serves the [sample documents](documents) at `127.0.0.1:8081`. In another
terminal:

```sh
curl -sG --data-urlencode 'q=incident recovery' --data 'limit=1' \
  http://127.0.0.1:8081/search
curl -s http://127.0.0.1:8081/status
```

The sample query returns this response. `total` counts all matching documents
before the result limit:

```json
{
  "version": 1,
  "total": 3,
  "results": [
    {
      "path": "incident-recovery.md",
      "title": "Incident recovery",
      "score": 12,
      "snippet": "… release. ## Restore service Assign one incident lead. Record the affected service and the first failed request. Check the deployment history before you change …"
    }
  ]
}
```

Status returns `version`, `documents`, and `problem`. An empty `problem` means
that the last scan succeeded. Invalid queries return HTTP 400 with a JSON `error`
field.

To use your own notes, set the directory and scan interval:

```sh
go run ./examples/search/cmd/search -dir /path/to/notes -every 2s -addr 127.0.0.1:8081
```

Edit, add, or remove a `.md` file while the server runs. Repeat the query after
the scan interval. A changed title or body creates a new version; a deleted file
leaves the results. A scan with no index changes keeps the version number.
Press Ctrl+C to stop HTTP requests and wait for the indexer to stop.

## Query rules

- Read regular `.md` files in the directory and its subdirectories. The extension
  ignores case. Do not follow symbolic links.
- Require valid UTF-8. Trim outer whitespace, then use a first line that starts
  with `# ` as the title; otherwise use the file name without its extension.
  Collapse body whitespace for snippets. Search Markdown source, including code
  and link text; do not render Markdown or remove its syntax.
- Split text at each character that is not a Unicode letter or digit, then
  convert words to lower case. Do not remove accents, stem words, or interpret
  quotes and operators.
- Require every distinct query word to occur in the title or body. Repeated
  query words do not increase the score. Queries have at most 256 characters
  and 1 to 8 distinct words.
- Add 3 points for each title occurrence and 1 for each body occurrence of a
  query word. Sort by descending score, then by ascending relative file path.
- Return 5 results by default. `limit` accepts 1 to 20. A snippet contains up to
  160 characters near the first matching body word, with an ellipsis at each
  cut end. A title-only match uses the start of the body.

The indexer rescans all files. It does not use a filesystem watcher, an external
search engine, a database, or a network source. A directory scan is not a
filesystem transaction: save complete files before the scan when a group of
edits must appear together.

## Read the code

1. [http.go](http.go): validate the request, call `Read` once, and write the
   ordinary JSON result.
2. [query.go](query.go): read the selected token lists, rank matching document
   IDs, and copy result text.
3. [index.go](index.go): read and parse ordinary files, build the token lists,
   and freeze complete documents and collections.
4. [indexer.go](indexer.go): load the first index in `Init`, poll in `Run`, and
   publish a complete replacement through `IndexCell`.
5. [main.go](cmd/search/main.go): supply `os.DirFS`, start the service, and stop
   the HTTP server and service group on an interrupt.

`Index` and `Document` are Frozen records. Their generated
[projections](zz_immutago.gen.go) read fields directly. Numeric `Posting` records
use `im.Copied`; `im.Value(postings.At(i))` returns an independent value with
direct field access. `Indexer.Read`
returns `IndexProjection`; its root stays fixed throughout a query, including
when another version is published. The query keeps that projection local and does
not capture it. `Status` is a Local result with copied counts and an error
projection.

`Path` and `Title` use `Clone` because the ordinary result owns their text.
The snippet function uses `Unsafe` for local reads by the standard string APIs,
then returns independent text. The status handler uses `Unsafe` to give the
JSON encoder temporary read access. These calls trust the normal Go string
contract; `Unsafe` itself does not enforce a lifetime.

An initial scan or parse error prevents startup. A later error leaves the last
valid index available and sets `problem`. The next successful scan clears the
error, including when the files did not change. Expected file errors do not
cause a supervisor restart. The service's mutable fields and publication lock
still require ordinary Go review.

## Checks

The [query tests](query_test.go) cover tokens, ranking, all-word matching, stable
ties, query bounds, and Unicode snippets. The [service tests](indexer_test.go)
cover edits, additions, deletion, empty directories, scan and parse errors,
recovery, initial failure, and shutdown. A paused scan test runs concurrent HTTP
queries, checks complete versions, and reads a selected old projection after
publication. The [HTTP tests](http_test.go) check status and request errors; the
[command tests](cmd/search/main_test.go) check startup failure and shutdown.

Run these focused checks after a change:

```sh
go generate ./examples/search
go test ./examples/search/...
go run ./cmd/immutago ./examples/search/...
```

Use hosted CI for the full test and race checks.
