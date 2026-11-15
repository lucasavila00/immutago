package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
	"testing"
)

// TestVariadicInputs checks zero, individual, and spread arguments.
func TestVariadicInputs(t *testing.T) {
	tree := &implementation.Tree{Weight: 3}
	var reader api.VariadicAlias = tree
	local := api.NewVariadicReaderProjection(im.ProjectionOf(reader))
	values := []int{1, 2}
	var empty []int
	if tree.Sum() != 3 || reader.Sum() != 3 || local.Sum() != 3 {
		t.Fatal("zero arguments")
	}
	if tree.Sum(1, 2) != 6 || reader.Sum(1, 2) != 6 || local.Sum(1, 2) != 6 {
		t.Fatal("individual arguments")
	}
	if tree.Sum(values...) != 6 || reader.Sum(values...) != 6 || local.Sum(values...) != 6 {
		t.Fatal("spread arguments")
	}
	if local.Sum(empty...) != 3 || local.Sum(nil...) != 3 {
		t.Fatal("nil slice")
	}
	named := api.Counts{1, 2}
	if local.Sum(named...) != 6 {
		t.Fatal("named slice")
	}
	local.Retain(values...)
	values[0] = 8
	if implementation.SavedNumbers[0] != 8 {
		t.Fatal("spread storage was copied")
	}
	local.Retain()
	if implementation.SavedNumbers != nil {
		t.Fatal("zero argument slice")
	}
	local.Retain([]int{}...)
	if implementation.SavedNumbers == nil {
		t.Fatal("empty argument slice")
	}
	amounts := []api.Amount{1, 2}
	if tree.Add(4, amounts...) != 7 || reader.Add(4, 1, 2) != 7 || local.Add(4, amounts...) != 7 {
		t.Fatal("named elements")
	}
	if local.Add(4) != 4 || local.Call(func() int { return 5 }) != 5 || reader.Box(1, "x") != 2 {
		t.Fatal("ordinary inputs")
	}
}
