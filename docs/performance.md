# Collection costs

The analyzer adds no runtime proof checks. This does not make collection
construction and updates free: persistent storage needs allocations.

| Operation | Storage work |
| --- | --- |
| `Slice.Len`, `Map.Len` | Constant time, no allocation |
| `Slice.At` | At most one path through a 32-way tree, no allocation |
| `Slice.Set` | Copy one path and one block of up to 32 elements |
| `Slice.Append` | Copy the tail and affected paths; share other blocks |
| `Map.Get` | At most 13 hash-trie levels and a collision bucket; no allocation |
| `Map.Set`, `Map.Delete` | Copy affected paths and collision buckets |
| `NewSlice`, `NewMap`, `Values` | Copy the mutable input or output boundary |

Map key hashing cost depends on key size. Full-hash collision buckets use linear
key comparisons. Persistent versions retain shared storage until no version
refers to it. Element types must pass the analyzer for shallow boundary copies
to be sufficient.

Run `make bench` for sizes 32, 1,024, and 32,768. Benchmarks report allocations
for reads and updates. Tests also assert allocation-free reads and compare
retained versions with ordinary Go containers after random update sequences.

Initial local measurements used Go 1.27.4 on darwin/arm64, Apple A18 Pro.
`Slice.At` and `Map.Get` reported 0 B/op and 0 allocs/op at all three sizes.
These are local observations, not latency guarantees. The persistent data
structures do not promise the same access cost as native mutable collections.
