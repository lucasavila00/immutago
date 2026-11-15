//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package api

import "github.com/lucasavila00/immutago/pkg/im"

type Count int

func (c *Count) Inc() { *c++ }

type Settings struct {
	im.Frozen
	Limit int
	Count Count
	Names im.FrozenSlice[string]
	Array [2]int
}

func (s *Settings) Read() int { return s.Limit }
func (s Settings) Value() int { return s.Limit }

type Pointer = *Settings

type State struct {
	Settings Pointer
	Next     *State
	Calls    int
}

func (s *State) Begin(cell *SettingsCell) {
	s.Settings = cell.LoadSharedPointer()
	s.Calls++
}

func (s *State) Reset() { s.Settings = nil }

type Embedded struct{ *Settings }
type Outer struct{ Embedded }

var Raw im.Cell[Settings]

func Current() *Settings { return Raw.LoadSharedPointer() }
