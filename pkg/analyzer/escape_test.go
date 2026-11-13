package analyzer

import (
	"regexp"
	"testing"
)

// TestEscapingBorrowCannotLosePermissions checks known erasure paths through helper summaries and transports.
func TestEscapingBorrowCannotLosePermissions(t *testing.T) {
	cases := []struct{ name, source string }{
		{"global_publish", `var Published *int
func save() { var value Value; Published = &value.X }
func mutate() { *Published = 2 }`},
		{"holder_wrapper", `type Holder struct { Pointer *int }
func wrap(pointer *int) Holder { return Holder{Pointer: pointer} }
func f(v Value) { holder:=wrap(&v.X); *holder.Pointer=2 }`},
		{"channel_wrapper", `var channel=make(chan *int,1)
func send(pointer *int) { channel<-pointer }
func f(v Value) { send(&v.X); pointer:=<-channel; *pointer=2 }`},
		{"setter_closure_wrapper", `func setter(pointer *int) func() { return func(){*pointer=2} }
func f(v Value) { change:=setter(&v.X); change() }`},
		{"reader_closure_wrapper", `func reader(pointer *int) func()int { return func()int{return *pointer} }
func f(v Value) { later:=reader(&v.X); v.X=2; _=later() }`},
		{"unknown_effect_wrapper", `var external func(*int)
func wrap(pointer *int) { external(pointer) }
func f(v Value) { wrap(&v.X) }`},
		{"global_field_return", `var Global Value
func field() *int { return &Global.X }
func f() { pointer:=field(); *pointer=2 }`},
		{"init_prefix", `var Global Value
func initializeLater() { Global.X=2 }
func f() { initializeLater() }`},
		{"map_parameter", `func f(values map[string]*Value) { pointer:=values["key"]; pointer.X=2 }`},
		{"channel_parameter", `func f(values <-chan *Value) { pointer:=<-values; pointer.X=2 }`},
		{"panic_transport", `func panicWith(pointer *int) { panic(pointer) }
func f(v Value) { defer func(){if payload:=recover();payload!=nil{*payload.(*int)=2}}(); panicWith(&v.X) }`},
	}
	rejected := regexp.MustCompile(`borrow|shared reference|unsupported|package-level immutable`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := checkSource(t, borrowTestTypes+tc.source)
			if !rejected.MatchString(reports) {
				t.Fatalf("diagnostics = %q; want protected storage rejection", reports)
			}
		})
	}
}

// TestErasedInputsCannotCreateOwnership keeps pointer assertions read-only when their origin is unknown.
func TestErasedInputsCannotCreateOwnership(t *testing.T) {
	cases := []struct{ name, source string }{
		{"interface parameter", `func f(payload any) { pointer:=payload.(*Value); pointer.X=2 }`},
		{"interface channel", `func f(values <-chan any) { pointer:=(<-values).(*Value); pointer.X=2 }`},
		{"interface map", `func f(values map[string]any) { pointer:=values["key"].(*Value); pointer.X=2 }`},
	}
	rejected := regexp.MustCompile(`borrow|shared reference|unsupported`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := checkSource(t, borrowTestTypes+tc.source)
			if !rejected.MatchString(reports) {
				t.Fatalf("diagnostics = %q; want unknown origin rejection", reports)
			}
		})
	}
}

// TestSelectCannotEraseBorrow checks select instructions with defaults that cannot lower to plain channel operations.
func TestSelectCannotEraseBorrow(t *testing.T) {
	cases := []struct{ name, source string }{
		{"send", `var channel=make(chan *int,1)
   func send(pointer *int) { select {case channel<-pointer: default:} }
   func f(v Value) { send(&v.X); pointer:=<-channel; *pointer=2 }`},
		{"receive", `func f(channel <-chan *Value) { select {case pointer:=<-channel: pointer.X=2; default:} }`},
	}
	rejected := regexp.MustCompile(`borrow|shared reference|unsupported`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := checkSource(t, borrowTestTypes+tc.source)
			if !rejected.MatchString(reports) {
				t.Fatalf("diagnostics = %q; want select transport rejection", reports)
			}
		})
	}
}

// TestAggregateElementsRemainShared checks incoming pointers reached through range and value-array elements.
func TestAggregateElementsRemainShared(t *testing.T) {
	cases := []struct{ name, source string }{
		{"map range", `func f(values map[int]*Value) { for _,pointer:=range values { pointer.X=2 } }`},
		{"array parameter", `func f(values [1]*Value) { values[0].X=2 }`},
	}
	rejected := regexp.MustCompile(`borrow|shared reference|unsupported`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := checkSource(t, borrowTestTypes+tc.source)
			if !rejected.MatchString(reports) {
				t.Fatalf("diagnostics = %q; want incoming alias rejection", reports)
			}
		})
	}
}

// TestAlternativeDereferenceCannotEraseLoan checks that summary joins preserve an indirect returned alias.
func TestAlternativeDereferenceCannotEraseLoan(t *testing.T) {
	source := borrowTestTypes + `
 func choose(payload any,direct bool) *Value {
  if direct {return payload.(*Value)}
  return *payload.(**Value)
 }
 func f(v Value) { pointer:=&v; alias:=choose(&pointer,false); v.X=2; _=read(alias) }
 `
	reports := checkSource(t, source)
	rejected := regexp.MustCompile(`borrow|shared reference|unsupported`)
	if !rejected.MatchString(reports) {
		t.Fatalf("diagnostics = %q; want indirect alternative rejection", reports)
	}
}
