# Publish immutable data

The [shipping application](../../examples/shipping/README.md) reads an editable
rate file while HTTP requests calculate quotes. It builds a complete catalog,
publishes it, and lets each request read one version. This guide follows that path.
Use [Get started](getting-started.md) to install the tools and generate a first record.

## Build the catalog

The [catalog declaration](../../examples/shipping/catalog.go) uses `im.Frozen`
for records and frozen collections for shared entries:

```go
type Catalog struct {
    im.Frozen
    Version      string
    Currency     string
    Destinations im.FrozenMap[string, string]
    Services     im.FrozenSlice[CarrierService]
}
```

The JSON decoder fills ordinary Go structs. Validation checks those structs
before `buildCatalog` constructs immutable records. Each record has its final
field values at construction. To collect weight bands, the builder appends
complete records, then freezes the slice:

```go
bands := make([]Band, 0, len(sourceBands))
for _, sourceBand := range sourceBands {
    bands = append(bands, Band{UpToGrams: sourceBand.UpToGrams, PriceCents: *sourceBand.PriceCents})
}
frozenBands := im.FreezeSlice(bands)
```

Freeze copies no entries. The checker rejects aliases of `bands` and any use
of it after Freeze. `frozenBands` can share its immutable storage with later
readers. The builder repeats this process for the maps and outer slice.
The [collection reference](../reference/frozen.md) lists all builder operations.

## Publish the completed value

The [loader](../../examples/shipping/loader.go) owns a generated `CatalogCell`.
After validation and construction succeed, it calls:

```go
l.current.Store(buildCatalog(document))
```

Store publishes the new root atomically. A failed file read or validation never
reaches Store, so the previous catalog stays available. Each read selects one root:

```go
func (l *Loader) Catalog() CatalogProjection {
    return l.current.Load()
}
```

The generated projection reads fields without copying the containing record.
A later Store does not change the version already selected by a reader.
See [Cells](../reference/cell.md) for empty values and allowed storage forms.

## Calculate a quote and keep its result

The [HTTP handler](../../examples/shipping/http.go) selects once and passes the
projection to the calculation:

```go
catalog := loader.Borrow().Catalog()
response := Quote(catalog, request)
```

`Quote` reads the same catalog for every parcel. It copies numeric results and
clones the strings that it puts in the ordinary JSON response:

```go
response := Response{
    CatalogVersion: catalog.Version().Clone(), Currency: catalog.Currency().Clone(),
    Destination: request.Destination, Quotes: make([]Offer, 0),
}
```

A projection is a Local value: direct calls can pass or return it, but it cannot
enter persistent storage. `Clone` gives independent string storage. `Unsafe`
gives a shared native string when a Go API needs one; the caller can retain that
string, and the checker does not track its lifetime.

Use `im.Local` to combine projections with computed values, as the search
indexer's [Status result](../../examples/search/indexer.go) does. Use `im.Copied`
for a record whose data fields all copy by value and whose callers need to keep
or change those copies. The [type rules](../reference/types.md) give the complete
field lists. Frozen records remain immutable even when all fields are numbers.
