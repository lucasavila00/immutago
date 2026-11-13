package main

import "github.com/lucasavila00/immutago/pkg/im"

type User struct {
	im.Immut
	Name   string
	Scores im.FrozenMap[string, int]
}

func main() {
	scores := map[string]int{"go": 10}
	frozen := im.FreezeMap(scores)
	user := &User{Name: "alice", Scores: frozen}
	shared := user
	user = &User{Name: "bob", Scores: frozen}

	next := shared.Scores.Clone()
	next["go"] = 20
	updated := im.FreezeMap(next)
	oldScore, _ := shared.Scores.Get("go")
	newScore, _ := updated.Get("go")
	println(shared.Name, user.Name, oldScore, newScore)
}
