package main

import "github.com/lucasavila00/immutago/pkg/im"

// User owns a name and read-only scores.
type User struct {
	im.Marker
	Name   string
	Scores im.FrozenMap[string, int]
}

// scoreCount reads a shared user without retaining its reference.
func scoreCount(user *User) int {
	return user.Scores.Len()
}

// main constructs and reads immutable storage.
func main() {
	scores := map[string]int{"go": 10}
	frozen := im.FreezeMap(scores)
	user := User{Name: "alice", Scores: frozen}

	p := &user
	count := scoreCount(p) // Last use of p.
	user.Name = "bob"      // The borrow has ended.

	next := user.Scores.Clone()
	next["go"] = 20
	updated := im.FreezeMap(next)

	oldScore, oldFound := user.Scores.Get("go")
	newScore, newFound := updated.Get("go")
	println(user.Name, count, oldScore, oldFound, newScore, newFound)
}
