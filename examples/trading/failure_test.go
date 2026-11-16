package trading

import (
	"context"
	"log"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestUnknownPlacementKeepsCashUntilFinalCancel handles a lost acknowledgment and a cancel-time fill.
func TestUnknownPlacementKeepsCashUntilFinalCancel(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		venue.unknownPlace, venue.cancelFailures, venue.fillOnCancel = true, 1, 97
		bot := startBot(t, files, venue, new(eventLog))
		defer func() { _ = bot.Stop() }()
		price(venue, "desk-a", "AAA", 1)
		advance()
		held := bot.Read()
		quote := NewMarketReportProjection(held.Active().Markets().AtPointer(0)).Totals().Quote
		if got := held.Active().Totals(); got.Cash != 1000 || got.Reserved != 97 || !quote.BidPending {
			t.Fatal("unknown placement released cash", got, quote)
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if got := bot.Read().Active().Totals(); got.Cash != 903 || got.Reserved != 0 || got.Inventory != 1 || got.Buys != 1 {
			t.Fatal("final cancellation fill was not applied once", got)
		}
		for _, order := range venue.accounts["desk-a"].orders {
			venue.emit("desk-a", "AAA", Event{Kind: OrderEvent, Order: order.state})
		}
		price(venue, "desk-a", "AAA", 1)
		advance()
		if got := bot.Read().Active().Totals(); got.Cash != 903 || got.Buys != 1 {
			t.Fatal("old price or fill changed the reconciled account", got)
		}
	})
}

// TestCancelFillRepricesFromCurrentInventory includes final fills before the new quote pair.
func TestCancelFillRepricesFromCurrentInventory(t *testing.T) {
	files := sampleFiles(t)
	files.set("trading.json", `{"active":{"id":"skew","account":"desk-a","max_reserved_cash":1000,
 "parameters":{"edge":1,"skew":10},"markets":[{"symbol":"AAA"}]}}`)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		bot := startBot(t, files, venue, new(eventLog))
		defer func() { _ = bot.Stop() }()
		venue.emit("desk-a", "AAA", Event{Kind: PriceEvent, Tick: Tick{Sequence: 1, Bid: 90, Ask: 92}})
		synctest.Wait()
		venue.fillOnCancel = 90
		venue.emit("desk-a", "AAA", Event{Kind: PriceEvent, Tick: Tick{Sequence: 2, Bid: 100, Ask: 102}})
		synctest.Wait()
		advance()
		got := marketValues(bot, 0)
		if got.Inventory != 1 || got.Quote.Bid != 90 || got.Quote.Ask != 102 || got.Buys != 1 {
			t.Fatal(got)
		}
	})
}

// TestStopReportsUnconfirmedOrders preserves reservations and reports the actual client ID.
func TestStopReportsUnconfirmedOrders(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		bot := startBot(t, files, venue, new(eventLog))
		price(venue, "desk-a", "AAA", 1)
		venue.cancelFailures = -1
		if err := bot.Stop(); err == nil || !strings.Contains(err.Error(), "unresolved orders (bid") {
			t.Fatal(err)
		}
		held := bot.state.selected[0].account
		if got := held.read(); got.Reserved != 97 || !held.hasOrders() {
			t.Fatal("failed cancellation erased the pending order", got)
		}
	})
}

// TestStopCancellationDeadline bounds cleanup while retaining an order with an unknown outcome.
func TestStopCancellationDeadline(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		bot := startBot(t, files, venue, new(eventLog))
		price(venue, "desk-a", "AAA", 1)
		venue.blockCancel("desk-a")
		done := make(chan error, 1)
		go func() { done <- bot.Stop() }()
		<-venue.entered
		time.Sleep(2 * time.Second)
		if err := <-done; err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
			t.Fatal(err)
		}
		if !bot.state.selected[0].account.hasOrders() {
			t.Fatal("timed-out cancellation dropped its order")
		}
	})
}

// TestFailedStagedSelectionRetainsUnresolvedAccount covers rollback after a new worker has traded.
func TestFailedStagedSelectionRetainsUnresolvedAccount(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		venue.fund("desk-c")
		bot := startBot(t, files, venue, new(eventLog))
		venue.failAdmission("desk-d")
		replacement := strings.ReplaceAll(candidateConfig, "desk-a", "desk-c")
		replacement = strings.ReplaceAll(replacement, "desk-b", "desk-d")
		// A new strategy ID is required when its account changes.
		files.set("trading.json", strings.ReplaceAll(replacement, "wide", "replacement"))
		<-venue.accountEntered
		synctest.Wait()
		price(venue, "desk-c", "AAA", 1)
		venue.cancelFailures = -1
		close(venue.accountRelease)
		synctest.Wait()
		<-bot.Done()
		if err := bot.Stop(); err == nil {
			t.Fatal("rollback cancellation failure was hidden")
		}
		found := false
		for _, saved := range bot.state.unresolved {
			if saved.account != nil && saved.spec.Account == "desk-c" &&
				saved.account.hasOrders() && saved.account.read().Reserved == 97 {
				found = true
			}
		}
		if !found {
			t.Fatal("the failed staged account was discarded")
		}
	})
}

// TestAdmissionRejectsPreexistingOrders leaves orders owned by an earlier session unchanged.
func TestAdmissionRejectsPreexistingOrders(t *testing.T) {
	files := sampleFiles(t)
	venue := newFakeVenue()
	_, err := venue.Place(context.Background(), "desk-a", "AAA", Order{
		ID: OrderID{Session: 1, Sequence: 1}, Side: Buy, Price: 90, Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	bot, err := Start(Options{API: venue, Files: files, Every: time.Second, Log: log.New(new(eventLog), "", 0)})
	if err == nil || bot != nil || !strings.Contains(err.Error(), "preexisting open orders") {
		t.Fatal(bot, err)
	}
	balance, err := venue.Account(context.Background(), "desk-a")
	if err != nil || balance.OpenOrders != 1 {
		t.Fatal(balance, err)
	}
}
