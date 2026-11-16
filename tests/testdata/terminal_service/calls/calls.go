package calls

import (
	"context"

	"example.com/consumer/api"
)

func Misuse(ref api.Ref, ctx context.Context) {
	_ = ref.Borrow()
	_ = ref.Borrow
	_ = ref.Borrow().Count
	ref.Borrow().Init()
	ref.Borrow().Run(ctx)
	ref.Borrow().Children()
	ref.Borrow().Args()
}
