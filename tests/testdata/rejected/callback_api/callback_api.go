package callback_api

var saved func() *int

func Apply(get func() *int) { *get() = 1 }
func Store(get func() *int) { saved = get }
func ApplyStored()          { *saved() = 1 }

type Getter interface{ Ptr() *int }

func ApplyGetter(get Getter) { *get.Ptr() = 1 }
func Box(value any) any      { return value }
