package trading

import (
	"context"
	"log"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// StrategyArgs supplies one selected trial's account and instrument streams.
type StrategyArgs struct {
	API     API
	Spec    strategySpec
	Account *account
	Policy  supervise.Ref[Policy]
	Every   time.Duration
	Updates chan<- struct{}
	Log     *log.Logger
	StopBot func(error)
}

// Strategy owns its market workers and publishes a report at a fixed interval.
type Strategy struct {
	supervise.Service[StrategyArgs]
	current StrategyReportCell
}

// Init creates instrument children and a complete initial report before publication.
func (s *Strategy) Init(supervisor *supervise.Supervisor) error {
	args := s.Args()
	for index, market := range args.Spec.Markets {
		if _, err := supervise.Start[Maker](supervisor, MakerArgs{
			API: args.API, AccountName: args.Spec.Account, Symbol: market.Symbol, Log: args.Log,
			Policy: args.Policy, Account: args.Account, Index: index,
			Parameters: args.Spec.Parameters, StrategyID: args.Spec.ID, StopBot: args.StopBot,
		}); err != nil {
			return err
		}
	}
	s.publish()
	return nil
}

// Run samples account state without adding report allocations to each market event.
func (s *Strategy) Run(ctx context.Context) {
	args := s.Args()
	args.Log.Printf("strategy started: %s", args.Spec.ID)
	defer args.Log.Printf("strategy stopped: %s", args.Spec.ID)
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	notify(args.Updates)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.publish()
			args.Log.Printf("strategy updated: %s", args.Spec.ID)
			notify(args.Updates)
		}
	}
}

// Read selects one complete report without borrowing mutable account state.
func (s *Strategy) Read() StrategyReportProjection { return s.current.Load() }

// publish copies account and instrument results under their one shared lock.
func (s *Strategy) publish() {
	args := s.Args()
	account := args.Account
	account.mu.Lock()
	totals := account.totals()
	markets := make([]MarketReport, 0, len(args.Spec.Markets))
	for index, market := range args.Spec.Markets {
		values := account.markets[index].totals
		values.Quote = account.quote(index)
		problem := ""
		if err := account.markets[index].problem; err != nil {
			problem = err.Error()
		}
		markets = append(markets, MarketReport{Symbol: market.Symbol, Totals: values, Problem: problem})
	}
	account.mu.Unlock()
	frozen := im.FreezeSlice(markets)
	s.current.Store(StrategyReport{ID: args.Spec.ID, Account: args.Spec.Account, Totals: totals, Markets: frozen})
}

// notify coalesces periodic report changes; it is not called for individual ticks.
func notify(updates chan<- struct{}) {
	select {
	case updates <- struct{}{}:
	default:
	}
}
