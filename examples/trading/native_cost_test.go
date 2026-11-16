package trading

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// nativePolicy has the same fields as PolicySnapshot, without a Frozen marker or Cell.
type nativePolicy struct {
	Version uint64
	Limits  Limits
	Problem string
}

// nativePolicyService publishes a complete native policy with the same atomic semantics.
type nativePolicyService struct{ current atomic.Pointer[nativePolicy] }

// nativeMaker uses one atomic service selection and one atomic snapshot selection per price.
// The account engine, locks, order IDs, adapter calls, and stream are shared with the real bot.
type nativeMaker struct {
	args   MakerArgs
	policy atomic.Pointer[nativePolicyService]
}

// onEvent mirrors Maker.onEvent, changing only the policy read to native atomic pointers.
func (m *nativeMaker) onEvent(ctx context.Context, event Event) (Quote, error) {
	args := m.args
	if event.Kind == OrderEvent {
		args.Account.apply(args.Index, event.Order)
		return args.Account.readQuote(args.Index), nil
	}
	policy := m.policy.Load().current.Load()
	limits := policy.Limits
	desired, fresh := args.Account.plan(args.Index, event.Tick, args.Parameters, limits, policy.Version)
	if !fresh {
		return desired, nil
	}
	previous := args.Account.readQuote(args.Index)
	if previous.Bid == desired.Bid && previous.Ask == desired.Ask {
		return previous, nil
	}
	if err := errors.Join(m.cancel(ctx, Buy), m.cancel(ctx, Sell)); err != nil {
		return args.Account.readQuote(args.Index), err
	}
	desired = args.Account.replan(args.Index, args.Parameters, limits)
	for _, side := range [2]Side{Buy, Sell} {
		price := desired.Bid
		if side == Sell {
			price = desired.Ask
		}
		order, needed := args.Account.reserve(args.Index, side, price, limits)
		if needed {
			state, err := args.API.Place(ctx, args.AccountName, args.Symbol, order)
			if err != nil {
				args.Account.setProblem(args.Index, err)
				return args.Account.readQuote(args.Index), err
			}
			args.Account.apply(args.Index, state)
		}
	}
	return args.Account.readQuote(args.Index), nil
}

// cancel mirrors the production order cancellation and cumulative fill handling.
func (m *nativeMaker) cancel(ctx context.Context, side Side) error {
	args := m.args
	order := args.Account.currentOrder(args.Index, side)
	if order.ID.Sequence == 0 {
		return nil
	}
	state, err := args.API.Cancel(ctx, args.AccountName, order.ID)
	if err != nil {
		args.Account.setProblem(args.Index, err)
		return err
	}
	args.Account.apply(args.Index, state)
	args.Account.canceled(args.Index)
	return nil
}

// run uses the same Stream.Next boundary and cleanup contract as Maker.Run.
func (m *nativeMaker) run(ctx context.Context) {
	stream, err := m.args.API.Subscribe(ctx, m.args.AccountName, m.args.Symbol)
	if err != nil {
		panic(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := stream.Close(cleanup); err != nil {
			panic(err)
		}
		if err := errors.Join(m.cancel(cleanup, Buy), m.cancel(cleanup, Sell)); err != nil {
			panic(err)
		}
	}()
	for {
		event, err := stream.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			panic(err)
		}
		if _, err := m.onEvent(ctx, event); err != nil {
			panic(err)
		}
	}
}
