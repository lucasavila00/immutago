package trading

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestFirstServiceFailure checks terminal failures through the public bot and runtime.
func TestFirstServiceFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		service string
		value   string
	}{
		{"maker", "Maker", "market connection lost"},
		{"strategy", "Strategy", "log dependency panic: strategy updated: wide"},
		{"trading", "Trading", "file dependency panic: trading.json"},
		{"subscribe", "Maker", "subscription failed"},
		{"placement", "Maker", "placement response lost"},
		{"cancel", "Maker", "placement response lost"},
		{"close", "Maker", "market connection lost"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if os.Getenv("IMMUTAGO_TRADING_FAILURE_TEST") == test.name {
				runTerminalFailure(t, test.name)
				return
			}
			command := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1")
			command.Env = append(os.Environ(), "IMMUTAGO_TRADING_FAILURE_TEST="+test.name)
			output, err := command.CombinedOutput()
			text := string(output)
			failure := "supervise: terminal failure in trading." + test.service + ": " + test.value
			if err == nil || strings.Count(text, "service failed: "+failure) != 1 ||
				strings.Count(text, "panic: "+failure) != 1 {
				t.Fatalf("terminal failure = %v\n%s", err, text)
			}
			report := strings.Index(text, "service failed: "+failure)
			stack := strings.Index(text, "(*"+test.service+").Run(")
			cleanup := strings.Index(text, "terminal cleanup verified")
			fatal := strings.Index(text, "panic: "+failure)
			if stack < report || cleanup < stack || fatal < cleanup {
				t.Fatalf("failure stack, cleanup, or final panic is missing or out of order\n%s", text)
			}
			if test.name == "cancel" &&
				!strings.Contains(text, "cleanup failed: account desk-a, instrument AAA has unresolved orders") {
				t.Fatalf("unconfirmed orders were not reported\n%s", text)
			}
			if test.name == "close" && !strings.Contains(text, "cleanup failed: subscription close failed") {
				t.Fatalf("subscription cleanup error was not reported\n%s", text)
			}
		})
	}
}

// runTerminalFailure checks resources from the last service's cleanup before the process ends.
func runTerminalFailure(t *testing.T, name string) {
	files := sampleFiles(t)
	files.set("trading.json", candidateConfig)
	synctest.Test(t, func(t *testing.T) {
		venue := newFakeVenue()
		events := &eventLog{output: os.Stderr}
		var bot *Bot
		events.onWrite = func(line string) {
			if line != "policy stopped" {
				return
			}
			checkTerminalServices(t, name, venue, events)
			if name == "cancel" || name == "maker" || name == "placement" {
				checkTerminalAccount(t, name, bot.state.selected[0].account)
			}
			fmt.Fprintln(os.Stderr, "terminal cleanup verified")
		}
		if name == "subscribe" {
			venue.subscribeFailures["desk-a/AAA"] = 1
		}
		bot = startBot(t, files, venue, events)
		defer func() { _ = bot.Stop() }()
		triggerTerminalFailure(t, name, bot, files, venue, events)
		time.Sleep(2 * time.Second)
		t.Fatal("the first service failure did not terminate the process")
	})
}

// checkTerminalServices checks that every service and subscription stops after one start.
func checkTerminalServices(t *testing.T, name string, venue *fakeVenue, events *eventLog) {
	t.Helper()
	for _, key := range []string{"desk-a/AAA", "desk-a/BBB", "desk-b/AAA", "desk-b/BBB"} {
		want := 1
		if name == "subscribe" && key == "desk-a/AAA" {
			want = 0
		}
		if starts, active, peak := venue.counts(key); starts != want || active != 0 || peak != want {
			t.Fatalf("subscription %s: starts=%d active=%d peak=%d", key, starts, active, peak)
		}
	}
	for _, started := range []string{
		"policy started", "trading started", "strategy started: wide", "strategy started: tight",
	} {
		if events.count(started) != 1 {
			t.Fatal("service did not start exactly once", started, events.snapshot())
		}
	}
	if name != "cancel" {
		checkVenueStopped(t, venue)
	}
}

// checkTerminalAccount checks retained orders and fills after failure cleanup.
func checkTerminalAccount(t *testing.T, name string, account *account) {
	t.Helper()
	if name == "cancel" {
		if got := account.read(); got.Cash != 1000 || got.Reserved != 97 || !account.hasOrders() {
			t.Fatal("failed cancellation discarded the order", got)
		}
	}
	if name == "maker" || name == "placement" {
		if got := account.read(); got.Cash != 903 || got.Reserved != 0 || got.Inventory != 1 || got.Buys != 1 {
			t.Fatal("cleanup lost or repeated a fill", got)
		}
	}
}

// triggerTerminalFailure supplies one failure through the selected dependency.
func triggerTerminalFailure(t *testing.T, name string, bot *Bot, files *testFiles, venue *fakeVenue, events *eventLog) {
	t.Helper()
	switch name {
	case "subscribe":
		return
	case "placement", "cancel":
		venue.unknownPlace = true
		if name == "cancel" {
			venue.cancelFailures = -1
		} else {
			venue.fillOnCancel = 97
			venue.blockCancel("desk-a")
		}
		price(venue, "desk-a", "AAA", 1)
		if name == "placement" {
			<-venue.entered
			account := bot.state.selected[0].account
			if got, quote := account.read(), account.readQuote(0); got.Cash != 1000 || got.Reserved != 97 || !quote.BidPending {
				t.Fatal("unknown placement released its reservation", got, quote)
			}
			close(venue.release)
		}
		return
	}
	for _, account := range []string{"desk-a", "desk-b"} {
		for _, symbol := range []string{"AAA", "BBB"} {
			price(venue, account, symbol, 1)
		}
	}
	switch name {
	case "maker":
		venue.fill("desk-a", bot.state.selected[0].account.readQuote(0).BidID, 97)
		synctest.Wait()
		venue.failStream("desk-a", "AAA")
	case "strategy":
		venue.blockCancel("desk-a")
		events.failOnce("strategy updated: wide")
		<-venue.entered
		if events.count("strategy started: wide") != 1 || events.count("policy stopped") != 0 {
			t.Fatal("shutdown passed a child that has not finished cleanup", events.snapshot())
		}
		close(venue.release)
	case "trading":
		files.fail("trading.json", 1, true)
	case "close":
		venue.closeFailure = "desk-a/AAA"
		venue.failStream("desk-a", "AAA")
	}
}
