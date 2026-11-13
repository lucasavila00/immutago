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
