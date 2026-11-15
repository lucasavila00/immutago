package trading

import (
	"testing"
	"testing/synctest"
)

// TestBotQuotesFillsAndPolicy drives real services through the exchange boundary.
func TestBotQuotesFillsAndPolicy(t *testing.T) {
	files := sampleFiles(t)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		bot := startBot(t, files, venue, new(eventLog))
		defer func() {
			if err := bot.Stop(); err != nil {
				t.Error(err)
			}
		}()
		venue.emit("desk-a", "AAA", Event{Kind: PriceEvent, Tick: Tick{Sequence: 1, Bid: 99, Ask: 101}})
		synctest.Wait()
		advance()
		selected := bot.Read()
		first := NewMarketReportProjection(selected.Active().Markets().AtPointer(0)).Totals()
		if first.Quote.Bid != 97 || first.Quote.Ask != 0 || first.Quote.PolicyVersion != 1 {
			t.Fatal(first)
		}
		fill := venue.fill("desk-a", first.Quote.BidID, 97)
		venue.emit("desk-a", "AAA", Event{Kind: OrderEvent, Order: fill})
		venue.emit("desk-a", "AAA", Event{Kind: PriceEvent, Tick: Tick{Sequence: 2, Bid: 99, Ask: 101}})
		synctest.Wait()
		advance()
		second := bot.Read()
		market := NewMarketReportProjection(second.Active().Markets().AtPointer(0)).Totals()
		if market.Inventory != 1 || market.Buys != 1 || market.Quote.Bid != 96 || market.Quote.Ask != 102 || second.Active().Totals().Cash != 903 {
			t.Fatal(market, second.Active().Totals())
		}
		files.set("policy.json", `{"version":2,"max_inventory":1,"max_spread":10}`)
		advance()
		venue.emit("desk-a", "AAA", Event{Kind: PriceEvent, Tick: Tick{Sequence: 3, Bid: 99, Ask: 101}})
		synctest.Wait()
		advance()
		final := bot.Read()
		changed := NewMarketReportProjection(final.Active().Markets().AtPointer(0)).Totals()
		if changed.Quote.Bid != 0 || changed.Quote.Ask == 0 || changed.Quote.PolicyVersion != 2 {
			t.Fatal(changed)
		}
		if old := NewMarketReportProjection(selected.Active().Markets().AtPointer(0)).Totals(); old != first {
			t.Fatal("an old immutable report changed", old, first)
		}
	})
}
