package valid

import (
	"testing"

	"example.com/consumer/api"
)

var selected api.State

func TestSelectedVersion(t *testing.T) {
	cell := &api.SettingsCell{}
	if cell.LoadSharedPointer() != nil {
		t.Fatal("empty Cell is not nil")
	}
	cell.Store(api.Settings{Limit: 3})
	state := Begin(cell)
	first := state.Settings
	if first != cell.LoadSharedPointer() || Containers(first) != 9 || Promoted(first) != 3 {
		t.Fatal("saved pointer lost the selected version")
	}
	if Handles(first) != 12 || Capture(first)() != 3 {
		t.Fatal("pointer transport changed the target")
	}
	cell.Store(api.Settings{Limit: 7})
	if first.Limit != 3 || cell.LoadSharedPointer().Limit != 7 {
		t.Fatal("Store changed an old version")
	}
	End(state)
	state = Begin(cell)
	if state.Settings.Limit != 7 {
		t.Fatal("reused request did not select the new version")
	}
	End(state)
	if got := testing.AllocsPerRun(100, func() { selected.Settings = cell.LoadSharedPointer() }); got != 0 {
		t.Fatalf("shared pointer load allocated %g times", got)
	}
}

func BenchmarkRetainedVersion(b *testing.B) {
	cell := &api.SettingsCell{}
	cell.Store(api.Settings{Limit: 3})
	b.Run("copy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			selected.Settings = new(cell.LoadShared())
		}
	})
	b.Run("pointer", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			selected.Settings = cell.LoadSharedPointer()
		}
	})
}
