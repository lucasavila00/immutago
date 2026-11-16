package trading

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// MakerArgs selects one funded account, instrument, and immutable policy source.
type MakerArgs struct {
	API         API
	AccountName string
	Symbol      string
	Log         *log.Logger
	Policy      supervise.Ref[Policy]
	Account     *account
	Index       int
	Parameters  Parameters
	StrategyID  string
	StopBot     func(error)
}

// Maker owns one market subscription; Run alone processes its events and order slots.
type Maker struct{ supervise.Service[MakerArgs] }

// Init reconciles uncertain orders from the previous attempt before resubscribing.
func (m *Maker) Init() error { return m.cleanup() }

// onEvent handles normalized fills or the full price-to-quote hot path for Run.
func (m *Maker) onEvent(ctx context.Context, event Event) (Quote, error) {
	args := m.Args()
	if event.Kind == OrderEvent {
		args.Account.apply(args.Index, event.Order)
		return args.Account.readQuote(args.Index), nil
	}
	policy := args.Policy.Borrow().Read()
	limits := policy.Limits()
	desired, fresh := args.Account.plan(args.Index, event.Tick, args.Parameters, limits, policy.Version())
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

// Run consumes venue events; its subscription and orders end with this attempt.
func (m *Maker) Run(ctx context.Context) {
	args := m.Args()
	defer func() {
		if err := m.cleanup(); err != nil && ctx.Err() != nil {
			args.StopBot(err)
		}
	}()
	stream, err := args.API.Subscribe(ctx, args.AccountName, args.Symbol)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		panic(err)
	}
	defer func() {
		closeContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := stream.Close(closeContext); err != nil {
			args.Account.setProblem(args.Index, err)
			args.StopBot(err)
		}
	}()
	args.Log.Printf("maker started: %s/%s", args.StrategyID, args.Symbol)
	defer args.Log.Printf("maker stopped: %s/%s", args.StrategyID, args.Symbol)
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

// cancel applies terminal fills before releasing an order's remaining reservation.
func (m *Maker) cancel(ctx context.Context, side Side) error {
	args := m.Args()
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

// cleanup bounds cancellation time and preserves reservations on unknown outcomes.
func (m *Maker) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := errors.Join(m.cancel(ctx, Buy), m.cancel(ctx, Sell))
	if err != nil {
		args := m.Args()
		buy := args.Account.currentOrder(args.Index, Buy).ID
		sell := args.Account.currentOrder(args.Index, Sell).ID
		err = fmt.Errorf("account %s, instrument %s has unresolved orders (bid %d/%d, ask %d/%d): %w",
			args.AccountName, args.Symbol, buy.Session, buy.Sequence, sell.Session, sell.Sequence, err)
	}
	m.Args().Account.setProblem(m.Args().Index, err)
	return err
}
