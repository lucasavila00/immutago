package rejected

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

var Saved, Found = api.Pair()

func Store()                                 { Saved, Found = api.Pair() }
func Box()                                   { var value any; var found bool; value, found = api.Pair(); _, _ = value, found }
func BoxLast()                               { var value any; var found bool; found, value = api.Reverse(); _, _ = value, found }
func ReturnBox() (bool, any)                 { return api.Reverse() }
func take(bool, ...any)                      {}
func ArgumentBox()                           { take(api.Reverse()) }
func Pointer() (found bool, value *api.View) { return }
func Callback(read func() (api.View, bool))  { _, _ = read() }
func Capture() func() int {
	value, _ := api.Pair()
	return func() int { return value.Count.N }
}
func Write(value api.View) { value.Count.N, Found = api.Numbers() }
func Independent()         { value, found := api.View{}, true; _ = value; _ = found }

func slices() ([]int, bool)         { return nil, false }
func reversedSlices() (bool, []int) { return false, nil }
func FreezeFirst()                  { values, _ := slices(); _ = im.FreezeSlice(values) }
func FreezeLast()                   { _, values := reversedSlices(); _ = im.FreezeSlice(values) }
func DeclareFirst()                 { var values, _ = slices(); _ = im.FreezeSlice(values) }
func DeclareLast()                  { var _, values = reversedSlices(); _ = im.FreezeSlice(values) }
func ReplaceFirst()                 { values := []int{}; values, _ = slices(); _ = im.FreezeSlice(values) }
func ReplaceLast()                  { values := []int{}; _, values = reversedSlices(); _ = im.FreezeSlice(values) }
func ConsumedSlot() {
	values := []bool{false}
	_ = im.FreezeSlice(values)
	var view api.View
	view, values[0] = api.Pair()
	_ = view
}
