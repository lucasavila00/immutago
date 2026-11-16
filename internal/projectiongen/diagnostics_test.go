package projectiongen

import (
	"errors"
	"go/types"
	"path/filepath"
	"testing"
)

// TestDeclarationErrors keeps generation failures at the declaration with the shared proof reason.
func TestDeclarationErrors(t *testing.T) {
	for _, test := range []struct {
		name, source, reason string
		line                 int
	}{
		{"slice", "type Table struct { im.Frozen; Rows []int }", "Table: Rows: []int is not provably immutable", 3},
		{"local_field", "type Bad struct { im.Frozen; Title im.StringProjection }",
			"Bad: Title: Frozen storage cannot contain Local values", 3},
		{"defined_pointer", "type Selected *Settings\ntype Bad struct { im.Frozen; S Selected }",
			"Bad: S: defined pointer types are unsupported in Frozen data; use a pointer alias", 4},
		{"mixed", "type Mixed struct { im.Frozen; im.Local; N int }",
			"Mixed: struct cannot embed both im.Frozen and im.Local", 3},
		{"mixed_aliases", "type F = im.Frozen\ntype L = im.Local\ntype Mixed struct { F; L; N int }",
			"Mixed: struct cannot embed both im.Frozen and im.Local", 5},
		{"duplicate", "type F = im.Frozen\ntype Duplicate struct { im.Frozen; F; N int }",
			"Duplicate: struct must embed exactly one immutable marker; each embedded alias counts as a marker", 4},
		{"generic_record", "type Bad[T any] struct { im.Frozen; N int }",
			"Bad: user generic declarations cannot define Frozen or Local types; " +
				"type parameters are unsupported in immutable storage", 3},
		{"generic_interface", "type Bad[T any] interface { im.FrozenInterface; Read() T }",
			"Bad: user generic declarations cannot define Frozen or Local types; " +
				"type parameters are unsupported in immutable storage", 3},
		{"record_member", "type Bad struct { im.Frozen; Unsafe int }",
			"Bad: *Bad.Unsafe conflicts with a generated projection member", 3},
		{"interface_member", "type Bad interface { im.FrozenInterface; IsNil() bool }",
			"Bad: Bad.IsNil conflicts with a generated projection member", 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := module(t, "package example\nimport \"github.com/lucasavila00/immutago/pkg/im\"\n"+
				"type Settings struct { im.Frozen; Limit int }\n")
			write(t, filepath.Join(dir, "bad.go"),
				"package example\nimport \"github.com/lucasavila00/immutago/pkg/im\"\n"+test.source+"\n")
			_, err := Generate(dir)
			var failure types.Error
			if !errors.As(err, &failure) {
				t.Fatalf("generation error has no source position: %v", err)
			}
			position := failure.Fset.Position(failure.Pos)
			if filepath.Base(position.Filename) != "bad.go" || position.Line != test.line || position.Column != 6 {
				t.Errorf("generation position = %s; want bad.go:%d:6", position, test.line)
			}
			if want := test.reason + "; other generation failures may remain"; failure.Msg != want {
				t.Errorf("generation reason = %q; want %q", failure.Msg, want)
			}
		})
	}
}
