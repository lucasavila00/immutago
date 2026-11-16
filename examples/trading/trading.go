package trading

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"sync/atomic"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// savedAccount keeps actual balances and pending order IDs across service retries.
type savedAccount struct {
	spec    strategySpec
	account *account
}

// state retains only selected accounts and unresolved accounts from a failed stop.
// A failed retirement stops the bot; unresolved has at most two entries.
type state struct {
	selected   [2]savedAccount
	unresolved [2]savedAccount
	session    uint64
	sequence   atomic.Uint64
}

// tradingArgs supplies the policy dependency, venue, and state owned by the bot.
type tradingArgs struct {
	Files   fs.FS
	API     API
	Every   time.Duration
	Log     *log.Logger
	Policy  supervise.Ref[Policy]
	State   *state
	StopBot func(error)
}

// runningStrategy connects one selective lifetime group to its current service.
type runningStrategy struct {
	saved savedAccount
	group *supervise.Group
	ref   supervise.Ref[Strategy]
}

// Trading owns strategy selection and publishes it before retiring old workers.
type Trading struct {
	supervise.Service[tradingArgs]
	running  [2]runningStrategy
	updates  chan struct{}
	problem  string
	stopping bool
	current  ReportCell
}

// Init rebuilds owned services with the same account state after a retry.
func (t *Trading) Init() error {
	t.updates = make(chan struct{}, 1)
	config, err := readConfiguration(t.Args().Files)
	if err != nil {
		return err
	}
	return t.selectStrategies(config)
}

// Run applies operator edits and incorporates periodic strategy reports.
func (t *Trading) Run(ctx context.Context) {
	args := t.Args()
	args.Log.Print("trading started")
	defer args.Log.Print("trading stopped")
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.updates:
			t.publish()
		case <-ticker.C:
			config, err := readConfiguration(args.Files)
			if err == nil {
				err = t.selectStrategies(config)
			}
			if t.stopping {
				return
			}
			if err != nil && t.problem != err.Error() {
				t.problem = err.Error()
				args.Log.Printf("configuration: %v", err)
				t.publish()
			}
		}
	}
}

// Read selects one immutable report that remains valid after strategy retirement.
func (t *Trading) Read() ReportProjection { return t.current.Load() }

// selectStrategies reuses current accounts and workers before stopping retired groups.
func (t *Trading) selectStrategies(config configuration) error {
	args := t.Args()
	desired := []*strategySpec{&config.Active, config.Candidate}
	var next [2]runningStrategy
	var added []runningStrategy
	for index, spec := range desired {
		if spec == nil {
			continue
		}
		next[index] = t.findRunning(*spec)
		if next[index].group != nil {
			continue
		}
		saved, err := t.selectAccount(*spec)
		if err == nil {
			next[index], err = t.startStrategy(saved)
		}
		if err != nil {
			for _, started := range added {
				t.retire(started)
			}
			return err
		}
		added = append(added, next[index])
	}
	old := t.running
	t.running, t.problem = next, ""
	args.State.selected = [2]savedAccount{next[0].saved, next[1].saved}
	t.publish()
	for _, previous := range old {
		if previous.group != nil && previous.group != next[0].group && previous.group != next[1].group {
			t.retire(previous)
		}
	}
	return nil
}

// selectAccount admits an idle dedicated account or reuses the current retry state.
func (t *Trading) selectAccount(spec strategySpec) (savedAccount, error) {
	args := t.Args()
	for _, selected := range args.State.selected {
		if selected.account == nil {
			continue
		}
		if sameStrategy(selected.spec, spec) {
			return selected, nil
		}
		if selected.spec.ID == spec.ID || selected.spec.Account == spec.Account {
			return savedAccount{}, fmt.Errorf("selected IDs and accounts keep their settings; " +
				"use a new ID and account for a trial")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	balance, err := args.API.Account(ctx, spec.Account)
	if err != nil {
		return savedAccount{}, err
	}
	if balance.OpenOrders != 0 {
		return savedAccount{}, fmt.Errorf("account %s has preexisting open orders", spec.Account)
	}
	return savedAccount{spec: spec, account: newAccount(spec, balance, args.State.session, &args.State.sequence)}, nil
}

// startStrategy stages one owned group; failed setup closes any staged children.
func (t *Trading) startStrategy(saved savedAccount) (runningStrategy, error) {
	args := t.Args()
	group := supervise.StartGroup(t.Children())
	ref, err := supervise.Start[Strategy](group, StrategyArgs{
		API: args.API, Spec: saved.spec, Account: saved.account, Policy: args.Policy,
		Every: args.Every, Updates: t.updates, Log: args.Log, StopBot: args.StopBot,
	})
	if err != nil {
		<-group.Stop()
		return runningStrategy{}, err
	}
	return runningStrategy{saved: saved, group: group, ref: ref}, nil
}

// publish combines immutable reports without retaining mutable account data.
func (t *Trading) publish() {
	active := t.running[0].ref.Borrow().Read()
	policy := t.Args().Policy.Borrow().Read()
	var candidate *StrategyReport
	if second := t.running[1]; second.group != nil {
		selected := second.ref.Borrow().Read()
		candidate = selected.Unsafe()
	}
	t.current.Store(Report{
		Active: active.Unsafe(), Candidate: candidate, Problem: t.problem,
		PolicyVersion: policy.Version(), PolicyProblem: policy.Problem().Unsafe(),
	})
}

// retire retains uncertain orders on both rollback and ordinary replacement paths.
func (t *Trading) retire(previous runningStrategy) {
	<-previous.group.Stop()
	if !previous.saved.account.hasOrders() {
		return
	}
	t.stopping = true
	for index, saved := range t.Args().State.unresolved {
		if saved.account == nil {
			t.Args().State.unresolved[index] = previous.saved
			return
		}
	}
}

// findRunning selects an unchanged strategy without restarting its workers.
func (t *Trading) findRunning(spec strategySpec) runningStrategy {
	for _, current := range t.running {
		if current.group != nil && sameStrategy(current.saved.spec, spec) {
			return current
		}
	}
	return runningStrategy{}
}
