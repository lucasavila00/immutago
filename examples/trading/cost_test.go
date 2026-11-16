package trading

import (
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
	"unsafe"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// costVenue retains only the two current orders and counts real adapter calls.
type costVenue struct {
	mu      sync.Mutex
	orders  [2]Order
	states  [2]OrderState
	places  int
	cancels int
	feed    *costStream
}

// Account supplies the dedicated account's initial test balance.
func (*costVenue) Account(context.Context, string) (Balance, error) {
	return Balance{Cash: 10000}, nil
}

// Subscribe returns the one instrument stream used by both cost implementations.
func (v *costVenue) Subscribe(context.Context, string, string) (Stream, error) {
	close(v.feed.ready)
	return v.feed, nil
}

// Place stores the current order in fixed storage and returns its acknowledgment.
func (v *costVenue) Place(_ context.Context, _, _ string, order Order) (OrderState, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	index := int(order.Side) - 1
	state := OrderState{ID: order.ID, Open: true}
	v.orders[index], v.states[index] = order, state
	v.places++
	return state, nil
}

// Cancel returns the final current order state without keeping an order history.
func (v *costVenue) Cancel(_ context.Context, _ string, id OrderID) (OrderState, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for index, state := range v.states {
		if state.ID == id {
			state.Open = false
			v.states[index] = state
			v.cancels++
			return state, nil
		}
	}
	panic("cost fixture canceled an unknown ID")
}

// fill returns a real cumulative fill event for one current order.
func (v *costVenue) fill(side Side) OrderState {
	v.mu.Lock()
	defer v.mu.Unlock()
	index := int(side) - 1
	order := v.orders[index]
	state := OrderState{ID: order.ID, Filled: 1, Value: order.Price}
	v.states[index] = state
	return state
}

// costStream generates a requested event batch inside the actual worker loop.
// Only batch boundaries use channels. Every measured event calls Stream.Next.
type costStream struct {
	venue     *costVenue
	fills     bool
	work      chan int
	done      chan struct{}
	ready     chan struct{}
	remaining int
	started   bool
	events    uint64
}

// Next creates fresh changing prices or a price/buy-fill/sell-fill cycle.
func (s *costStream) Next(ctx context.Context) (Event, error) {
	if s.remaining == 0 {
		if s.started {
			select {
			case s.done <- struct{}{}:
			case <-ctx.Done():
				return Event{}, ctx.Err()
			}
		}
		select {
		case count := <-s.work:
			s.remaining, s.started = count, true
		case <-ctx.Done():
			return Event{}, ctx.Err()
		}
	}
	s.remaining--
	s.events++
	if s.fills {
		switch (s.events - 1) % 3 {
		case 1:
			return Event{Kind: OrderEvent, Order: s.venue.fill(Buy)}, nil
		case 2:
			return Event{Kind: OrderEvent, Order: s.venue.fill(Sell)}, nil
		}
	}
	price := int64(99 + s.events%2)
	return Event{Kind: PriceEvent, Tick: Tick{Sequence: s.events, Bid: price, Ask: price + 2}}, nil
}

// Close has no resource beyond the batch channels created during setup.
func (*costStream) Close(context.Context) error { return nil }

// costFixture owns one running event loop and its measured fixed-storage adapter.
type costFixture struct {
	venue   *costVenue
	account *account
}

// newCostFixture starts either actual supervised workers or the matching direct Go loop.
func newCostFixture(tb testing.TB, fills, native bool) costFixture {
	tb.Helper()
	venue := &costVenue{}
	venue.feed = &costStream{venue: venue, fills: fills, work: make(chan int), done: make(chan struct{}), ready: make(chan struct{})}
	spec := strategySpec{ID: "cost", Account: "cost", MaxReservedCash: 1000, Parameters: Parameters{Edge: 1, Skew: 1}, Markets: []marketSpec{{Symbol: "AAA"}}}
	balance := Balance{Cash: 10000, Positions: map[string]int64{"AAA": 1}}
	account := newAccount(spec, balance, 42, new(atomic.Uint64))
	args := MakerArgs{API: venue, AccountName: "cost", Symbol: "AAA", Log: log.New(io.Discard, "", 0),
		Account: account, Parameters: spec.Parameters, StrategyID: "cost", StopBot: func(err error) { panic(err) }}
	var stop func()
	if native {
		worker := &nativeMaker{args: args}
		source := new(nativePolicyService)
		source.current.Store(&nativePolicy{Version: 1, Limits: Limits{MaxInventory: 5, MaxSpread: 10}})
		worker.policy.Store(source)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { defer close(done); worker.run(ctx) }()
		stop = func() { cancel(); <-done }
	} else {
		root := supervise.NewRoot(func(value any, _ []byte) { panic(value) })
		files := fstest.MapFS{"policy.json": {Data: []byte(`{"version":1,"max_inventory":5,"max_spread":10}`)}}
		policy, err := supervise.Start[Policy](root, PolicyArgs{Files: files, Every: time.Hour, Log: args.Log})
		if err != nil {
			tb.Fatal(err)
		}
		args.Policy = policy
		if _, err := supervise.Start[Maker](root, args); err != nil {
			tb.Fatal(err)
		}
		stop = func() { <-root.Stop() }
	}
	<-venue.feed.ready
	tb.Cleanup(stop)
	return costFixture{venue: venue, account: account}
}

// batch waits until the worker has completed the entire requested event batch.
func (f costFixture) batch(count int) {
	f.venue.feed.work <- count
	<-f.venue.feed.done
}

// checkCostWork proves the benchmark did not measure duplicate events or empty quotes.
func checkCostWork(tb testing.TB, fixture costFixture, count int, fills bool) {
	tb.Helper()
	totals := fixture.account.read()
	prices := count
	if fills {
		prices = (count + 2) / 3
	}
	if totals.Events != uint64(prices) || fixture.venue.places != 2*prices {
		tb.Fatal("missing price or order work", totals, fixture.venue.places, count)
	}
	if !fills && fixture.venue.cancels != 2*(count-1) {
		tb.Fatal("missing cancel work", fixture.venue.cancels, count)
	}
	if fills && (totals.Buys != int64((count+1)/3) || totals.Sells != int64(count/3)) {
		tb.Fatal("missing fill work", totals, count)
	}
}

// TestHotPathAllocations measures actual Stream.Next, service policy reads, risk, and order calls.
func TestHotPathAllocations(t *testing.T) {
	for _, fills := range []bool{false, true} {
		for _, native := range []bool{false, true} {
			name := costName(fills, native)
			t.Run(name, func(t *testing.T) {
				fixture := newCostFixture(t, fills, native)
				allocations := testing.AllocsPerRun(1000, func() { fixture.batch(1) })
				if allocations != 0 {
					t.Fatalf("event path allocates %g times", allocations)
				}
				checkCostWork(t, fixture, 1001, fills)
			})
		}
	}
}

// BenchmarkMarketMaker compares complete event paths with identical transport and synchronization.
func BenchmarkMarketMaker(b *testing.B) {
	for _, fills := range []bool{false, true} {
		for _, native := range []bool{false, true} {
			b.Run(costName(fills, native), func(b *testing.B) {
				fixture := newCostFixture(b, fills, native)
				b.ReportAllocs()
				b.ResetTimer()
				fixture.batch(b.N)
				b.StopTimer()
				checkCostWork(b, fixture, b.N, fills)
			})
		}
	}
}

// costName identifies the workload and policy publication mechanism.
func costName(fills, native bool) string {
	workload, implementation := "Reprice", "Immutago"
	if fills {
		workload = "FillCycle"
	}
	if native {
		implementation = "Go"
	}
	return workload + "/" + implementation
}

// TestNumericRepresentation checks that markers add no space to hot-path data.
func TestNumericRepresentation(t *testing.T) {
	type nativeTick struct {
		Sequence uint64
		Bid, Ask int64
	}
	type nativeID struct{ Session, Sequence uint64 }
	if unsafe.Sizeof(Tick{}) != unsafe.Sizeof(nativeTick{}) || unsafe.Sizeof(OrderID{}) != unsafe.Sizeof(nativeID{}) ||
		unsafe.Sizeof(PolicySnapshot{}) != unsafe.Sizeof(nativePolicy{}) {
		t.Fatal("a marker changed the numeric or policy representation")
	}
}

// TestCostImplementationParity checks the two wrappers against the same complete event batches.
func TestCostImplementationParity(t *testing.T) {
	for _, fills := range []bool{false, true} {
		actual := newCostFixture(t, fills, false)
		native := newCostFixture(t, fills, true)
		actual.batch(30)
		native.batch(30)
		if actual.account.read() != native.account.read() || actual.account.readQuote(0) != native.account.readQuote(0) ||
			actual.venue.places != native.venue.places || actual.venue.cancels != native.venue.cancels {
			t.Fatal("the direct Go comparison performs different work")
		}
	}
}
