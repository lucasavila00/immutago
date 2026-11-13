package main

import "github.com/lucasavila00/immutago/pkg/im"

type User struct {
	im.Shared
	Name   string
	Scores im.FrozenMap[string, int]
}

func main() {
	scores := map[string]int{"go": 10}
	frozen := im.FreezeMap(scores)
	user := &User{Name: "alice", Scores: frozen}
	score, _ := user.Scores.Get("go")
	println(user.Name, score)
}
