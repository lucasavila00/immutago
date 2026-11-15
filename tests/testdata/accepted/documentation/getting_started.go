package main

import "github.com/lucasavila00/immutago/pkg/im"

type User struct {
	im.Shared
	Name   im.FrozenString
	Scores im.FrozenMap[im.FrozenString, int]
}

func main() {
	scores := map[im.FrozenString]int{im.FreezeString("go"): 10}
	frozen := im.FreezeMap(scores)
	user := &User{Name: im.FreezeString("alice"), Scores: frozen}
	score, _ := user.Scores.Get(im.FreezeString("go"))
	println(user.Name.Clone(), score)
}
