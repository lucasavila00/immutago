package analyzer

import (
	"strings"
	"testing"
)

// TestMarkerAliasesPreserveLocalCalls checks both embedded marker aliases and local pointer aliases.
func TestMarkerAliasesPreserveLocalCalls(t *testing.T) {
	source := `type SharedMarker=im.Shared
type LocalMarker=im.LocalRef
type SharedValue struct{SharedMarker;X int}
type LocalValue struct{LocalMarker;X int}
type Pointer=*LocalValue
func read(value Pointer)int{return value.X}
func makeShared()*SharedValue{return &SharedValue{}}
func localCopy()LocalValue{p:=&LocalValue{X:1};_=read(p);return *p}`
	if got := checkSource(t, source); got != "" {
		t.Fatalf("marker alias changed its policy: %s", got)
	}
}

// TestLocalPointerAliasesCannotReturn checks aliases and defined pointers at the result boundary.
func TestLocalPointerAliasesCannotReturn(t *testing.T) {
	for _, pointerType := range []string{"type Pointer = *Value", "type Pointer *Value"} {
		source := `type Marker=im.LocalRef;type Value struct{Marker;X int};` + pointerType +
			`;func escape()Pointer{return nil}`
		if got := checkSource(t, source); !strings.Contains(got, "returning LocalRef pointers is unsupported") {
			t.Fatalf("local pointer result accepted: %s", got)
		}
	}
}

// TestMarkerCardinality counts compiler identities rather than distinct embedded names.
func TestMarkerCardinality(t *testing.T) {
	for _, marker := range []string{"Shared", "LocalRef"} {
		source := "type First=im." + marker + ";type Second=im." + marker + ";type Value struct{First;Second;X int}"
		if got := checkSource(t, source); !strings.Contains(got, "exactly one immutable marker") {
			t.Fatalf("duplicate %s marker aliases accepted: %s", marker, got)
		}
	}
	mixed := `type First=im.Shared;type Second=im.LocalRef;type Value struct{First;Second}`
	if got := checkSource(t, mixed); !strings.Contains(got, "cannot embed both") {
		t.Fatalf("mixed marker aliases accepted: %s", got)
	}
	if got := checkSource(t, `type Alias=im.Shared;type Child struct{Alias;X int};
 type Parent struct{im.Shared;Child Child}`); got != "" {
		t.Fatalf("single marker with a marked child rejected: %s", got)
	}
}
