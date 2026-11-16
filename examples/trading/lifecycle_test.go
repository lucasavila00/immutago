package trading

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
)

// price sends a new market update and waits until the worker has processed it.
func price(venue *fakeVenue, account, symbol string, sequence uint64) {
	venue.emit(account, symbol, Event{Kind: PriceEvent, Tick: Tick{Sequence: sequence, Bid: 99, Ask: 101}})
	synctest.Wait()
}

// marketValues reads one instrument from the immutable active report.
func marketValues(bot *Bot, index int) InstrumentTotals {
	report := bot.Read()
	return NewMarketReportProjection(report.Active().Markets().AtPointer(index)).Totals()
}

// TestPromotionPublishesBeforeRetirement preserves reports and the candidate workers.
func TestPromotionPublishesBeforeRetirement(t *testing.T) {
	files := sampleFiles(t)
	files.set("trading.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		bot := startBot(t, files, venue, new(eventLog))
		defer func() { _ = bot.Stop() }()
		price(venue, "desk-a", "AAA", 1)
		price(venue, "desk-b", "AAA", 1)
		advance()
		old := bot.Read()
		oldTotals := old.Active().Totals()
		venue.blockCancel("desk-a")
		files.set("trading.json", promotedConfig)
		<-venue.entered
		current := bot.Read()
		if !current.Active().ID().Is("tight") || !current.Candidate().IsNil() {
			t.Fatal("selection was not published before cleanup")
		}
		if old.Active().Totals() != oldTotals || !old.Active().ID().Is("wide") {
			t.Fatal("an in-flight report changed")
		}
		close(venue.release)
		synctest.Wait()
		for _, key := range []string{"desk-a/AAA", "desk-a/BBB"} {
			if _, active, _ := venue.counts(key); active != 0 {
				t.Fatal("old subscription survived", key)
			}
		}
		if starts, active, _ := venue.counts("desk-b/AAA"); starts != 1 || active != 1 {
			t.Fatal(starts, active)
		}
	})
}

// TestReadmissionUsesVenueBalancesAndNewIDs ends one trial and later admits its account again.
func TestReadmissionUsesVenueBalancesAndNewIDs(t *testing.T) {
	files := sampleFiles(t)
	activeConfig := string(files.data["trading.json"])
	files.set("trading.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		bot := startBot(t, files, venue, new(eventLog))
		defer func() { _ = bot.Stop() }()
		price(venue, "desk-b", "AAA", 1)
		advance()
		selected := bot.Read()
		old := NewMarketReportProjection(selected.Candidate().Markets().AtPointer(0)).Totals().Quote
		venue.fill("desk-b", old.BidID, old.Bid)
		synctest.Wait()
		files.set("trading.json", activeConfig)
		advance()
		if !bot.Read().Candidate().IsNil() {
			t.Fatal("the ended trial account was retained")
		}
		files.set("trading.json", candidateConfig)
		advance()
		price(venue, "desk-b", "AAA", 1)
		advance()
		current := bot.Read()
		next := NewMarketReportProjection(current.Candidate().Markets().AtPointer(0)).Totals()
		if next.Quote.BidID == old.BidID || next.Inventory != 1 || next.Buys != 0 ||
			current.Candidate().Totals().Cash != 901 {
			t.Fatal("readmission reused an ID or reset venue balances", old, next, current.Candidate().Totals())
		}
	})
}

// TestReverseShutdown waits for child resources before their owner and policy stop.
func TestReverseShutdown(t *testing.T) {
	files := sampleFiles(t)
	files.set("trading.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		venue, events := newFakeVenue(), new(eventLog)
		bot := startBot(t, files, venue, events)
		for _, account := range []string{"desk-a", "desk-b"} {
			for _, symbol := range []string{"AAA", "BBB"} {
				price(venue, account, symbol, 1)
			}
		}
		before := len(events.snapshot())
		if err := bot.Stop(); err != nil {
			t.Fatal(err)
		}
		var stopped []string
		for _, line := range events.snapshot()[before:] {
			if strings.Contains(line, "stopped") {
				stopped = append(stopped, line)
			}
		}
		want := []string{"maker stopped: tight/BBB", "maker stopped: tight/AAA", "strategy stopped: tight",
			"maker stopped: wide/BBB", "maker stopped: wide/AAA", "strategy stopped: wide", "trading stopped", "policy stopped"}
		if !slices.Equal(stopped, want) {
			t.Fatal(stopped)
		}
		checkVenueStopped(t, venue)
	})
}

// checkVenueStopped checks actual subscriptions and venue orders after root cleanup.
func checkVenueStopped(t *testing.T, venue *fakeVenue) {
	t.Helper()
	for _, account := range []string{"desk-a", "desk-b"} {
		for _, symbol := range []string{"AAA", "BBB"} {
			if _, active, _ := venue.counts(account + "/" + symbol); active != 0 {
				t.Fatal("subscription remains open", account, symbol)
			}
		}
		for _, order := range venue.accounts[account].orders {
			if order.state.Open {
				t.Fatal("order remains open", order)
			}
		}
	}
}
