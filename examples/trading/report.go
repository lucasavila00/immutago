//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package trading

import "github.com/lucasavila00/immutago/pkg/im"

// MarketReport connects one instrument's name to its complete numeric state.
type MarketReport struct {
	im.Frozen
	Symbol  string
	Totals  InstrumentTotals
	Problem string
}

// StrategyReport is a complete periodic snapshot, outside the per-event hot path.
type StrategyReport struct {
	im.Frozen
	ID      string
	Account string
	Totals  Totals
	Markets im.FrozenSlice[MarketReport]
}

// Report retains selected strategy snapshots after their workers stop.
type Report struct {
	im.Frozen
	Active        *StrategyReport
	Candidate     *StrategyReport
	Problem       string
	PolicyVersion uint64
	PolicyProblem string
}
