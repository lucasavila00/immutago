package trading

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
)

// marketSpec selects one instrument subscription.
type marketSpec struct {
	Symbol string `json:"symbol"`
}

// strategySpec fixes one trial's budget, quoting rule, and market subscriptions.
type strategySpec struct {
	ID         string       `json:"id"`
	Account    string       `json:"account"`
	Budget     int64        `json:"budget"`
	Parameters Parameters   `json:"parameters"`
	Markets    []marketSpec `json:"markets"`
}

// configuration selects the active strategy and an optional paper trial.
type configuration struct {
	Active    strategySpec  `json:"active"`
	Candidate *strategySpec `json:"candidate"`
}

// readConfiguration validates a complete selection before changing its workers.
func readConfiguration(files fs.FS) (configuration, error) {
	data, err := fs.ReadFile(files, "trading.json")
	if err != nil {
		return configuration{}, err
	}
	var config configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return configuration{}, err
	}
	if err := checkStrategy(config.Active); err != nil {
		return configuration{}, err
	}
	if config.Candidate != nil {
		if err := checkStrategy(*config.Candidate); err != nil {
			return configuration{}, err
		}
		if config.Candidate.ID == config.Active.ID || config.Candidate.Account == config.Active.Account || !slices.Equal(config.Active.Markets, config.Candidate.Markets) {
			return configuration{}, fmt.Errorf("strategies need different IDs, different funded accounts, and the same ordered markets")
		}
	}
	return config, nil
}

// checkStrategy requires bounded parameters and distinct named instruments.
func checkStrategy(spec strategySpec) error {
	if spec.ID == "" || spec.Account == "" || spec.Budget <= 0 || spec.Budget > 1_000_000_000_000 || len(spec.Markets) == 0 ||
		spec.Parameters.Edge <= 0 || spec.Parameters.Edge > 1_000_000 || spec.Parameters.Skew < 0 || spec.Parameters.Skew > 1_000_000 {
		return fmt.Errorf("strategy needs an ID, account, markets, budget 1..1000000000000, edge 1..1000000, and skew 0..1000000")
	}
	for i, market := range spec.Markets {
		if market.Symbol == "" {
			return fmt.Errorf("strategy %s needs named instruments", spec.ID)
		}
		for _, previous := range spec.Markets[:i] {
			if previous.Symbol == market.Symbol {
				return fmt.Errorf("strategy %s repeats an instrument", spec.ID)
			}
		}
	}
	return nil
}

// sameStrategy permits promotion without resetting the trial account or its workers.
func sameStrategy(a, b strategySpec) bool {
	return a.ID == b.ID && a.Account == b.Account && a.Budget == b.Budget && a.Parameters == b.Parameters && slices.Equal(a.Markets, b.Markets)
}
