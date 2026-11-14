package analyzer

import (
	"strings"
	"testing"
)

const processVariableDefinitions = `import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"github.com/lucasavila00/immutago/pkg/proc"
)
type Config struct { im.Scoped; Version int }
type Snapshot struct { im.Shared; Version int }
type Worker struct { proc.Process[proc.None]; n int }
func (w *Worker) Run(ctx context.Context) {}
var _, _, _, _, _ = io.EOF, http.DefaultClient, os.Stdout, sync.Mutex{}, context.Background
var _ atomic.Int64
var _, _ = errors.Is, fmt.Errorf
`

// checkProcessVariableCases runs each source after the process variable definitions.
func checkProcessVariableCases(t *testing.T, reject bool, cases [][2]string) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc[0], func(t *testing.T) {
			got := checkSource(t, processVariableDefinitions+tc[1])
			if reject != strings.Contains(got, "package variable") || !reject && got != "" {
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
		{"atomic counter", `var hits atomic.Int64
var ready atomic.Bool
func record() { hits.Add(1) }
func (w *Worker) Count() int64 { record(); ready.Store(true); go hits.Add(1); return hits.Load() }
func (w *Worker) Swap() bool { _ = hits.CompareAndSwap(1, 2); return ready.Swap(false) }`},
		{"atomic counters of each scalar type", `var a atomic.Int32
var b atomic.Uint32
var d atomic.Uint64
func (w *Worker) Sum() uint64 { return uint64(a.Add(1)) + uint64(b.Add(1)) + d.Add(1) }`},
		{"standard error compared", `func (w *Worker) eof(err error) bool { return err == io.EOF || io.EOF != err }`},
		{"standard error in errors.Is", `func (w *Worker) eof(err error) bool { return errors.Is(err, io.EOF) }`},
		{"standard error in switch", `func (w *Worker) eof(err error) int {
	switch err { case io.EOF, io.ErrUnexpectedEOF: return 1 }; return 0 }`},
		{"standard error in helper", `func isEOF(err error) bool { return errors.Is(err, io.EOF) }
func (w *Worker) eof(err error) bool { return isEOF(err) }`},
		{"exported error of this package compared", `type code string
func (c code) Error() string { return string(c) }
var ErrMissing error = code("missing")
func (w *Worker) missing(err error) bool { return err == ErrMissing }`},
		{"error set by init", `type code string
func (c code) Error() string { return string(c) }
var errMissing error
func init() { errMissing = code("missing") }
func (w *Worker) missing(err error) bool { return errors.Is(err, errMissing) }`},
		{"empty struct of another package", `func (w *Worker) Body() bool { return http.NoBody == http.NoBody }`},
		{"empty struct", `type marker struct{}
type nested struct{ marker; other struct{} }
func (marker) name() string { return "m" }
var m marker
var n nested
func (w *Worker) Name() string { _ = n; return m.name() }`},
		{"constant", `const size = 4
func (w *Worker) Size() int { return size }`},
		{"function call", `func helper() int { return 1 }
func (w *Worker) Value() int { return helper() }`},
		{"standard function call", `func (w *Worker) Write(path string) { _, _ = os.ReadFile(path) }`},
		{"variable outside process method", `var seen = map[string]int{}
func count() { seen["hits"]++ }`},
		{"helper with owner slot", `var current im.Owner[Config]
func publish(v int) { current.Store(&Config{Version: v}) }
func (w *Worker) Publish() { publish(w.n) }`},
		{"helper with unchanged scalar", `var limit = 10
func scaled(v int) int { return v * limit }
func (w *Worker) Scaled() int { return scaled(w.n) }`},
		{"recursive helper without variables", `func fib(n int) int { if n < 2 { return n }; return fib(n-1) + fib(n-2) }
func (w *Worker) Fib() int { return fib(w.n) }`},
		{"helper outside process method", `var seen = map[string]int{}
func record(k string) { seen[k]++ }
func count() { record("a") }`},
		{"interface method is not followed", `var seen = map[string]int{}
type recorder interface{ record(string) }
type mapRecorder struct{}
func (mapRecorder) record(k string) { seen[k]++ }
func (w *Worker) Run2() { var r recorder = mapRecorder{}; r.record("a") }`},
		{"standard function", `func (w *Worker) Read(path string) { _, _ = os.ReadFile(path) }`},
		{"other process method", `type Other struct { proc.Process[proc.None] }
var seen = map[string]int{}
func (o *Other) Run(context.Context) {}
func (o *Other) Hits() int { return 0 }
func (w *Worker) Ask(pid proc.PID[Other]) int { return pid.Proc().Hits() }`},
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
		{"standard error returned", `func (w *Worker) eof() error { return io.EOF }`},
		{"standard error copied", `func (w *Worker) eof() { err := io.EOF; _ = err }`},
		{"standard error message", `func (w *Worker) eof() string { return io.EOF.Error() }`},
		{"standard error wrapped", `func (w *Worker) eof() error { return fmt.Errorf("read: %w", io.EOF) }`},
		{"standard error in errors.As", `func (w *Worker) eof(err error) bool { return errors.As(err, &io.EOF) }`},
		{"standard error first in errors.Is", `func (w *Worker) eof(err error) bool { return errors.Is(io.EOF, err) }`},
		{"error changed elsewhere", `type code string
func (c code) Error() string { return string(c) }
var errMissing error = code("missing")
func set() { errMissing = code("gone") }
func (w *Worker) missing(err error) bool { return err == errMissing }`},
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
	})
}

// TestProcessAtomicsRejected covers atomics that a process method cannot use.
func TestProcessAtomicsRejected(t *testing.T) {
	checkProcessVariableCases(t, true, [][2]string{
		{"atomic value", `var current atomic.Value
func (w *Worker) Load() { _ = current.Load() }`},
		{"atomic pointer", `type Plain struct{ N int }
var current atomic.Pointer[Plain]
func (w *Worker) Load() int { return current.Load().N }`},
		{"atomic uintptr", `var hits atomic.Uintptr
func (w *Worker) Count() uintptr { return hits.Add(1) }`},
		{"exported atomic", `var Hits atomic.Int64
func (w *Worker) Count() int64 { return Hits.Add(1) }`},
		{"atomic address elsewhere", `var hits atomic.Int64
func pointer() *atomic.Int64 { return &hits }
func (w *Worker) Count() int64 { return hits.Add(1) }`},
		{"atomic method value elsewhere", `var hits atomic.Int64
var add = hits.Add
func (w *Worker) Count() int64 { return hits.Load() }`},
		{"atomic assignment elsewhere", `var hits atomic.Int64
func reset() { hits = atomic.Int64{} }
func (w *Worker) Count() int64 { return hits.Load() }`},
		{"atomic copy in method", `var hits atomic.Int64
func (w *Worker) Count() int64 { p := &hits; return p.Load() }`},
		{"atomic in struct", `type counters struct{ hits atomic.Int64 }
var all counters
func (w *Worker) Count() int64 { return all.hits.Add(1) }`},
		{"atomic array", `var hits [2]atomic.Int64
func (w *Worker) Count() int64 { return hits[0].Add(1) }`},
		{"struct with one scalar beside empty fields", `type marker struct{}
type mixed struct{ marker; n int }
var x mixed
func (w *Worker) Value() int { return x.n }`},
	})
}

// TestForeignErrorChangeRejected covers checked code that changes an error variable of another package.
func TestForeignErrorChangeRejected(t *testing.T) {
	for _, source := range []string{`func f() { io.EOF = nil }`, `func f() *error { return &io.EOF }`,
		`func f(errs []error) { for _, io.EOF = range errs {} }`} {
		if got := checkSource(t, processVariableDefinitions+source); !strings.Contains(got, "error variable io.EOF") {
			t.Fatalf("%s: diagnostics=%q", source, got)
		}
	}
}

// TestProcessCallsRejected covers functions that reach a rejected package variable from a process method.
func TestProcessCallsRejected(t *testing.T) {
	checkProcessVariableCases(t, true, [][2]string{
		{"helper", `var seen = map[string]int{}
func record(k string) { seen[k]++ }
func (w *Worker) Run2(ctx context.Context) { for ctx.Err() == nil { record("hits") } }`},
		{"helper reads", `var seen = map[string]int{}
func hits() int { return seen["hits"] }
func (w *Worker) Hits() int { return hits() }`},
		{"two levels", `var seen = map[string]int{}
func inner(k string) { seen[k]++ }
func outer(k string) { inner(k) }
func (w *Worker) Count() { outer("a") }`},
		{"helper declared later through a chain", `func (w *Worker) Count() { first() }
func first() { second() }
func second() { third() }
func third() { seen["a"]++ }
var seen = map[string]int{}`},
		{"recursive helpers", `var seen = map[string]int{}
func even(n int) bool { if n == 0 { seen["e"]++; return true }; return odd(n - 1) }
func odd(n int) bool { if n == 0 { return false }; return even(n - 1) }
func (w *Worker) Odd() bool { return odd(w.n) }`},
		{"method on ordinary type", `var seen = map[string]int{}
type tracker struct{}
func (tracker) record(k string) { seen[k]++ }
func (w *Worker) Count() { tracker{}.record("a") }`},
		{"pointer method on ordinary type", `var seen = map[string]int{}
type tracker struct{ n int }
func (t *tracker) record(k string) { t.n += seen[k] }
func (w *Worker) Count() { t := &tracker{}; t.record("a") }`},
		{"helper value", `var seen = map[string]int{}
func record(k string) { seen[k]++ }
func (w *Worker) Count() { f := record; f("a") }`},
		{"helper in goroutine", `var seen = map[string]int{}
func record(k string) { seen[k]++ }
func (w *Worker) Count() { go record("a") }`},
		{"generic helper", `var seen = map[string]int{}
func record[T any](k string) { seen[k]++ }
func (w *Worker) Count() { record[int]("a") }`},
		{"helper with changed scalar", `var limit = 10
func set(v int) { limit = v }
func scaled(v int) int { return v * limit }
func (w *Worker) Scaled() int { return scaled(w.n) }`},
		{"helper with standard variable", `func show(s string) { _, _ = io.WriteString(os.Stdout, s) }
func (w *Worker) Show() { show("x") }`},
		{"function value variable", `var seen = map[string]int{}
var record = func(k string) { seen[k]++ }
func (w *Worker) Count() { record("a") }`},
		{"function value factory", `var seen = map[string]int{}
func recorder() func(string) { return func(k string) { seen[k]++ } }
func (w *Worker) Count() { f := recorder(); f("a") }`},
		{"helper in closure", `var seen = map[string]int{}
func record(k string) { seen[k]++ }
func (w *Worker) Count() { func() { record("a") }() }`},
		{"standard client variable", `func (w *Worker) Client() { _ = http.DefaultClient.Timeout }`},
		{"standard file variable", `func (w *Worker) Print() { _, _ = io.WriteString(os.Stdout, "x") }`},
		{"standard slice variable", `func (w *Worker) Args() int { return len(os.Args) }`},
	})
}
