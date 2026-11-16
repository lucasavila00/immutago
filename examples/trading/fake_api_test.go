package trading

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// fakeOrder retains venue history so tests can deliver old acknowledgments and fills.
type fakeOrder struct {
	symbol string
	order  Order
	state  OrderState
}

// fakeAccount is venue-owned state, independent of every bot service lifetime.
type fakeAccount struct {
	cash      int64
	positions map[string]int64
	orders    map[OrderID]*fakeOrder
}

// fakeVenue drives full application tests. It is not a production trading adapter.
type fakeVenue struct {
	subscribeFailures map[string]int
	failAccount       string
	accountEntered    chan struct{}
	accountRelease    chan struct{}
	mu                sync.Mutex
	accounts          map[string]*fakeAccount
	streams           map[string]*fakeStream
	starts            map[string]int
	active            map[string]int
	peak              map[string]int
	events            []string
	unknownPlace      bool
	cancelFailures    int
	fillOnCancel      int64
	closeFailure      string
	blockAccount      string
	entered           chan struct{}
	release           chan struct{}
}

// newFakeVenue funds two independent accounts before the bot starts.
func newFakeVenue() *fakeVenue {
	v := &fakeVenue{accounts: map[string]*fakeAccount{}, streams: map[string]*fakeStream{},
		subscribeFailures: map[string]int{}, starts: map[string]int{}, active: map[string]int{}, peak: map[string]int{}}
	for _, name := range []string{"desk-a", "desk-b"} {
		v.accounts[name] = &fakeAccount{cash: 1000, positions: map[string]int64{}, orders: map[OrderID]*fakeOrder{}}
	}
	return v
}

// Account returns actual fake-venue balances and the number of open orders.
func (v *fakeVenue) Account(ctx context.Context, name string) (Balance, error) {
	v.mu.Lock()
	if v.failAccount == name {
		entered, release := v.accountEntered, v.accountRelease
		v.failAccount = ""
		v.mu.Unlock()
		close(entered)
		select {
		case <-release:
			return Balance{}, errors.New("account admission failed")
		case <-ctx.Done():
			return Balance{}, ctx.Err()
		}
	}
	defer v.mu.Unlock()
	account := v.accounts[name]
	if account == nil {
		return Balance{}, fmt.Errorf("unknown account %s", name)
	}
	result := Balance{Cash: account.cash, Positions: map[string]int64{}}
	for symbol, quantity := range account.positions {
		result.Positions[symbol] = quantity
	}
	for _, order := range account.orders {
		if order.state.Open {
			result.OpenOrders++
		}
	}
	return result, nil
}

// Subscribe opens an independently tracked market and fill stream.
func (v *fakeVenue) Subscribe(_ context.Context, account, symbol string) (Stream, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	key := account + "/" + symbol
	if v.subscribeFailures[key] != 0 {
		if v.subscribeFailures[key] > 0 {
			v.subscribeFailures[key]--
		}
		return nil, errors.New("subscription failed")
	}
	stream := &fakeStream{venue: v, key: key, items: make(chan fakeItem, 64)}
	v.streams[key] = stream
	v.starts[key]++
	v.active[key]++
	v.peak[key] = max(v.peak[key], v.active[key])
	v.events = append(v.events, "subscribe "+key)
	return stream, nil
}

// Place is idempotent and can lose its response after the venue accepts the order.
func (v *fakeVenue) Place(_ context.Context, account, symbol string, order Order) (OrderState, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	target := v.accounts[account]
	if existing := target.orders[order.ID]; existing != nil {
		return existing.state, nil
	}
	state := OrderState{ID: order.ID, Open: true}
	target.orders[order.ID] = &fakeOrder{symbol: symbol, order: order, state: state}
	v.events = append(v.events, "place "+account+"/"+symbol)
	if v.unknownPlace {
		v.unknownPlace = false
		return OrderState{}, errors.New("placement response lost")
	}
	return state, nil
}

// Cancel returns final cumulative fills even if placement acknowledgment was lost.
func (v *fakeVenue) Cancel(ctx context.Context, account string, id OrderID) (OrderState, error) {
	v.mu.Lock()
	if v.cancelFailures != 0 {
		if v.cancelFailures > 0 {
			v.cancelFailures--
		}
		v.mu.Unlock()
		return OrderState{}, errors.New("cancellation not confirmed")
	}
	var release chan struct{}
	if v.blockAccount == account {
		v.blockAccount = ""
		release = v.release
		close(v.entered)
	}
	v.mu.Unlock()
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return OrderState{}, ctx.Err()
		}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	order := v.accounts[account].orders[id]
	if order == nil {
		return OrderState{ID: id}, nil
	}
	if v.fillOnCancel != 0 && order.state.Open {
		v.fillLocked(account, order, v.fillOnCancel)
		v.fillOnCancel = 0
	}
	order.state.Open = false
	v.events = append(v.events, "cancel "+account+"/"+order.symbol)
	return order.state, nil
}

// emit supplies a market event or an old cumulative fill to the current subscription.
func (v *fakeVenue) emit(account, symbol string, event Event) {
	v.mu.Lock()
	stream := v.streams[account+"/"+symbol]
	v.mu.Unlock()
	stream.items <- fakeItem{event: event}
}

// failStream makes one market loop return an I/O error.
func (v *fakeVenue) failStream(account, symbol string) {
	v.mu.Lock()
	stream := v.streams[account+"/"+symbol]
	v.mu.Unlock()
	stream.items <- fakeItem{err: errors.New("market connection lost")}
}

// fill changes venue cash and holdings before it emits a cumulative fill report.
func (v *fakeVenue) fill(account string, id OrderID, price int64) OrderState {
	v.mu.Lock()
	order := v.accounts[account].orders[id]
	v.fillLocked(account, order, price)
	state, symbol := order.state, order.symbol
	v.mu.Unlock()
	v.emit(account, symbol, Event{Kind: OrderEvent, Order: state})
	return state
}

// fillLocked fills one whole unit under the caller's venue lock.
func (v *fakeVenue) fillLocked(account string, order *fakeOrder, price int64) {
	if !order.state.Open {
		return
	}
	order.state.Filled, order.state.Value, order.state.Open = 1, price, false
	target := v.accounts[account]
	if order.order.Side == Buy {
		target.cash -= price
		target.positions[order.symbol]++
	} else {
		target.cash += price
		target.positions[order.symbol]--
	}
}

// counts reads subscription ownership evidence for one account and instrument.
func (v *fakeVenue) counts(key string) (int, int, int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.starts[key], v.active[key], v.peak[key]
}

// blockCancel holds one cancellation to test publication before resource retirement.
func (v *fakeVenue) blockCancel(account string) {
	v.mu.Lock()
	v.blockAccount, v.entered, v.release = account, make(chan struct{}), make(chan struct{})
	v.mu.Unlock()
}

type fakeItem struct {
	event Event
	err   error
}

// fakeStream blocks on a real event or context cancellation, with no timer simulation.
type fakeStream struct {
	venue *fakeVenue
	key   string
	items chan fakeItem
}

// Next returns the next normalized exchange event.
func (s *fakeStream) Next(ctx context.Context) (Event, error) {
	select {
	case item := <-s.items:
		return item.event, item.err
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}

// Close releases the worker's stream.
func (s *fakeStream) Close(_ context.Context) error {
	s.venue.mu.Lock()
	defer s.venue.mu.Unlock()
	s.venue.active[s.key]--
	s.venue.events = append(s.venue.events, "unsubscribe "+s.key)
	if s.venue.closeFailure == s.key {
		return errors.New("subscription close failed")
	}
	return nil
}

// failAdmission holds one account query until a staged strategy has started work.
func (v *fakeVenue) failAdmission(name string) {
	v.mu.Lock()
	v.failAccount, v.accountEntered, v.accountRelease = name, make(chan struct{}), make(chan struct{})
	v.mu.Unlock()
}

// fund adds a distinct venue account for a replacement test.
func (v *fakeVenue) fund(name string) {
	v.mu.Lock()
	v.accounts[name] = &fakeAccount{cash: 1000, positions: map[string]int64{}, orders: map[OrderID]*fakeOrder{}}
	v.mu.Unlock()
}
