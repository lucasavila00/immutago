package analyzer

import (
	"strings"
	"testing"
)

const processVariableDefinitions = `import (
	"context"
	"io"
	"net/http"
	"os"
	"sync"
	"github.com/lucasavila00/immutago/pkg/proc"
)
type Config struct { im.Scoped; Version int }
type Snapshot struct { im.Shared; Version int }
type Worker struct { proc.Process[proc.None]; n int }
func (w *Worker) Run(ctx context.Context) {}
var _, _, _, _, _ = io.EOF, http.DefaultClient, os.Stdout, sync.Mutex{}, context.Background
`

// checkProcessVariableCases runs each source after the process variable definitions.
func checkProcessVariableCases(t *testing.T, reject bool, cases [][2]string) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc[0], func(t *testing.T) {
			got := checkSource(t, processVariableDefinitions+tc[1])
			if reject != strings.Contains(got, "process method uses package variable") || !reject && got != "" {
				t.Fatalf("diagnostics=%q rejected=%v", got, reject)
			}
		})
	}
}

// TestProcessVariablesAccepted covers each allowed package-level name in a process method.
func TestProcessVariablesAccepted(t *testing.T) {
	checkProcessVariableCases(t, false, [][2]string{
		{"owner slot", `var current im.Owner[Config]
func (w *Worker) Load() int { c := current.Load(); if c == nil { return 0 }; return c.Version }
func (w *Worker) publish() { current.Store(&Config{Version: w.n}) }`},
		{"unchanged scalar", `var limit = 10
func (w *Worker) Limit() int { return limit + w.n }`},
		{"unchanged shared pointer", `var initial = &Snapshot{Version: 1}
func (w *Worker) Initial() int { return initial.Version }`},
		{"unchanged frozen map", `var names = im.FreezeMap(map[string]int{"a": 1})
func (w *Worker) Name(key string) int { v, _ := names.Get(key); return v }`},
		{"unchanged array", `var table = [2]int{1, 2}
func (w *Worker) First() int { return table[0] }`},
		{"init assignment", `var table [4]int
func init() { for i := range table { table[i] = i } }
func (w *Worker) Last() int { return table[3] }`},
		{"value copy in other code", `var limit = 10
func read() int { other := limit; return other }
func (w *Worker) Limit() int { return limit }`},
		{"constant", `const size = 4
func (w *Worker) Size() int { return size }`},
		{"function call", `func helper() int { return 1 }
func (w *Worker) Value() int { return helper() }`},
		{"standard function call", `func (w *Worker) Write(path string) { _, _ = os.ReadFile(path) }`},
		{"variable outside process method", `var seen = map[string]int{}
func count() { seen["hits"]++ }`},
		{"local shadow", `var seen = map[string]int{}
func (w *Worker) Local() int { seen := map[string]int{}; return seen["a"] }`},
	})
}

// TestProcessVariablesRejected covers package variables that a process method cannot use.
func TestProcessVariablesRejected(t *testing.T) {
	checkProcessVariableCases(t, true, [][2]string{
		{"map write", `var seen = map[string]int{}
func (w *Worker) Count() { seen["hits"]++ }`},
		{"map read", `var seen = map[string]int{}
func (w *Worker) Hits() int { return seen["hits"] }`},
		{"slice", `var items []int
func (w *Worker) Len() int { return len(items) }`},
		{"pointer", `type Plain struct{ N int }
var state = &Plain{}
func (w *Worker) Value() int { return state.N }`},
		{"struct", `type Plain struct{ N int }
var state Plain
func (w *Worker) Value() int { return state.N }`},
		{"mutex", `var mu sync.Mutex
func (w *Worker) Lock() { mu.Lock(); mu.Unlock() }`},
		{"channel", `var events = make(chan int)
func (w *Worker) Send() { events <- 1 }`},
		{"error", `type code string
func (c code) Error() string { return string(c) }
var errMissing error = code("missing")
func (w *Worker) check() error { return errMissing }`},
		{"address", `var limit = 10
func (w *Worker) Pointer() { p := &limit; _ = p }`},
		{"scalar assignment in method", `var limit = 10
func (w *Worker) Set() { limit = 2 }`},
		{"scalar changed elsewhere", `var limit = 10
func set(v int) { limit = v }
func (w *Worker) Limit() int { return limit }`},
		{"scalar incremented elsewhere", `var limit = 10
func bump() { limit++ }
func (w *Worker) Limit() int { return limit }`},
		{"array element changed elsewhere", `var table = [2]int{1, 2}
func set() { table[1] = 3 }
func (w *Worker) First() int { return table[0] }`},
		{"array sliced elsewhere", `var table = [2]int{1, 2}
func view() []int { return table[:] }
func (w *Worker) First() int { return table[0] }`},
		{"address taken elsewhere", `var limit = 10
func pointer() *int { return &limit }
func (w *Worker) Limit() int { return limit }`},
		{"range assignment elsewhere", `var limit = 10
func scan(values []int) { for _, limit = range values {} }
func (w *Worker) Limit() int { return limit }`},
		{"closure assignment elsewhere", `var limit = 10
var reset = func() { limit = 0 }
func (w *Worker) Limit() int { return limit }`},
		{"pointer method elsewhere", `type Count struct { im.Shared; N int }
func (c *Count) Get() int { return c.N }
var count Count
func use() int { return count.Get() }
func (w *Worker) Value() int { return count.N }`},
		{"exported scalar", `var Limit = 10
func (w *Worker) Value() int { return Limit }`},
		{"use in closure", `var seen = map[string]int{}
func (w *Worker) Later() { f := func() { seen["a"] = 1 }; f() }`},
		{"use in goroutine", `var seen = map[string]int{}
func (w *Worker) Spawn() { go func() { seen["a"] = 1 }() }`},
		{"use in entry point", `type Reader struct { proc.Process[proc.None] }
var seen = map[string]int{}
func (r *Reader) Run(ctx context.Context) { _ = seen["a"] }`},
		{"holder variable", `type Holder struct { Current im.Owner[Config] }
var holder Holder
func (w *Worker) Value() { holder.Current.Store(&Config{}) }`},
		{"standard error variable", `func (w *Worker) eof(err error) bool { return err == io.EOF }`},
		{"standard client variable", `func (w *Worker) Client() { _ = http.DefaultClient.Timeout }`},
		{"standard file variable", `func (w *Worker) Print() { _, _ = io.WriteString(os.Stdout, "x") }`},
		{"standard slice variable", `func (w *Worker) Args() int { return len(os.Args) }`},
	})
}
