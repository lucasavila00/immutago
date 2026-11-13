# Immutability contract

Embed `immutable.Marker` directly by value. Every field must be a primitive
boolean, number, or string; an array of immutable elements; another marked
struct; or `immutable.Slice[T]` / `immutable.Map[K, V]` with immutable arguments.
Type aliases retain their type identity. The analyzer uses `go/types`, the Go
compiler's type checker, rather than source names or marker methods.

```go
import "github.com/lucasavila00/immutago/pkg/immutable"

type User struct {
    immutable.Marker
    Name string
    Tags immutable.Slice[string]
}

var user = User{Name: "Ada", Tags: immutable.NewSlice("go")}
```

Use literals to construct marked structs. Replacing a variable is allowed;
writing its fields is not. Collection updates return new values. Collection
inputs and outputs are copied at their mutable boundaries. Reads do not need
runtime immutability checks. Use `All()` to range over a collection without
copying it to a native slice or map.

Unmarked structs cannot contain marked values or immutable collections,
including through embedding or mutable containers. This restriction keeps
partially immutable object graphs outside the initial supported model.

Pointers, native slices and maps, interfaces, channels, and functions do not
prove immutable field ownership. Pointers to immutable storage, addresses of
its fields, and mutable slice views of its arrays are unsupported. Generic
constraints must prove every possible element immutable. Unknown cases fail;
there are no suppression comments, registration hooks, or trusted user types.
The library's own collection representation is the fixed implementation boundary.
Unsafe and reflection imports in analyzed consumer packages are rejected.
Resolved unsafe pointer values and reflection values are also unsupported,
including when obtained from another package.

Analyze all consumer packages, including tests, with `immutago ./...` or
`go vet -vettool=<absolute path to immutago> ./...`. The tool does not rewrite
Go or enforce rules at runtime. Running ordinary `go build` alone does not
check immutability. Dependencies must also be analyzed if they use these types;
unexamined dependency code is not certified by a clean consumer-package run.

# Analysis sources

The implementation uses Go's resolved types and structural type predicates.
Shared obligations form a graph. A definite failure propagates to every owner;
recursive collections succeed only when their obligations hold. The checker
also inspects source operations for writes and alias creation.

Expand the supported model only with proof tests. Broad coverage is a goal,
not a measured percentage.
