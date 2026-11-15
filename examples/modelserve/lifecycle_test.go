package modelserve

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestModelRefreshAndErrors keeps complete old coefficients through bad edits.
func TestModelRefreshAndErrors(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, events := sampleFiles(), new(eventLog)
		router, root := newTestRouter(t, files, events)
		defer func() { <-root.Stop() }()
		if got := readPrediction(t, router, "primary"); got.Version != "baseline-v1" || got.Estimate != 247000 {
			t.Fatal(got)
		}
		files.set("baseline.json", `{"version":"baseline-v2","base":20000,"per_square_meter":2000,"per_room":10000}`)
		time.Sleep(20 * time.Millisecond)
		synctest.Wait()
		if got := readPrediction(t, router, "primary"); got.Version != "baseline-v2" || got.Estimate != 230000 {
			t.Fatal(got)
		}
		files.set("baseline.json", `{"version":`)
		time.Sleep(20 * time.Millisecond)
		synctest.Wait()
		if status := router.Status()["primary"]; status.Version != "baseline-v2" || status.Problem == "" {
			t.Fatal(status)
		}
		files.set("baseline.json", `{"version":"repaired","base":42}`)
		files.set("primary.json", `{"model":"missing.json"}`)
		time.Sleep(20 * time.Millisecond)
		synctest.Wait()
		if got := readPrediction(t, router, "primary"); got.Version != "repaired" || got.Estimate != 42 {
			t.Fatal(got)
		}
		if router.Status()["primary"].Problem == "" {
			t.Fatal("missing replacement lost its error")
		}
	})
}

// TestSwapPublishesBeforeCleanup lets a request finish with its old selected model.
func TestSwapPublishesBeforeCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, events := sampleFiles(), new(eventLog)
		router, root := newTestRouter(t, files, events)
		defer func() { <-root.Stop() }()
		selected, finish := make(chan struct{}), make(chan struct{})
		result := make(chan Prediction, 1)
		go func() {
			model, err := router.Read("primary")
			if err != nil {
				panic(err)
			}
			close(selected)
			<-finish
			prediction, err := Predict(model, Features{Area: 90, Rooms: 3})
			if err != nil {
				panic(err)
			}
			result <- prediction
		}()
		<-selected
		events.blockCleanup("model stopped: baseline.json")
		files.set("primary.json", `{"model":"candidate.json"}`)
		<-events.entered
		if got := readPrediction(t, router, "primary"); got.Version != "candidate-v1" || got.Estimate != 252000 {
			t.Error(got)
		}
		close(finish)
		if got := <-result; got.Version != "baseline-v1" || got.Estimate != 247000 {
			t.Error(got)
		}
		close(events.release)
		synctest.Wait()
		count := files.count("baseline.json")
		time.Sleep(30 * time.Millisecond)
		if files.count("baseline.json") != count {
			t.Error("retired model still polls its file")
		}
	})
}

// TestModelFailureRestartsOnlyThatModel keeps its parent and challenger running.
func TestModelFailureRestartsOnlyThatModel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, events := sampleFiles(), new(eventLog)
		router, root := newTestRouter(t, files, events)
		defer func() { <-root.Stop() }()
		if err := router.SetChallenger("challenger.json"); err != nil {
			t.Fatal(err)
		}
		files.fail("baseline.json", 1)
		time.Sleep(2 * time.Second)
		synctest.Wait()
		lines := strings.Join(events.snapshot(), "\n")
		if strings.Count(lines, "model ready: baseline.json") != 2 ||
			strings.Count(lines, "model ready: candidate.json") != 1 ||
			strings.Count(lines, "switcher ready: primary.json") != 1 ||
			strings.Count(lines, "switcher ready: challenger.json") != 1 {
			t.Fatal(lines)
		}
		if files.count("candidate.json") < 2 {
			t.Fatal("peer stopped polling")
		}
		if got := readPrediction(t, router, "primary"); got.Estimate != 247000 {
			t.Fatal(got)
		}
	})
}

// TestParentRestartWaitsForChildren holds old child cleanup before replacement Init.
func TestParentRestartWaitsForChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, events := sampleFiles(), new(eventLog)
		router, root := newTestRouter(t, files, events)
		defer func() { <-root.Stop() }()
		if err := router.SetChallenger("challenger.json"); err != nil {
			t.Fatal(err)
		}
		events.blockCleanup("model stopped: baseline.json")
		files.fail("primary.json", 1)
		<-events.entered
		oldReads, peerReads := files.count("baseline.json"), files.count("candidate.json")
		time.Sleep(2 * time.Second)
		if files.count("primary.json") != 2 || files.count("baseline.json") != oldReads {
			t.Error("parent replaced before child cleanup")
		}
		if files.count("candidate.json") <= peerReads {
			t.Error("peer arm stopped")
		}
		close(events.release)
		time.Sleep(2 * time.Second)
		synctest.Wait()
		lines := strings.Join(events.snapshot(), "\n")
		if strings.Count(lines, "switcher ready: primary.json") != 2 ||
			strings.Count(lines, "model ready: baseline.json") != 2 {
			t.Error(lines)
		}
	})
}

// TestDynamicRemovalAndShutdownOrder stops only selected subtrees, then newest arms first.
func TestDynamicRemovalAndShutdownOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		files, events := sampleFiles(), new(eventLog)
		router, root := newTestRouter(t, files, events)
		if err := router.SetChallenger("challenger.json"); err != nil {
			t.Fatal(err)
		}
		old, err := router.Read("challenger")
		if err != nil {
			t.Fatal(err)
		}
		router.RemoveChallenger()
		if _, err := router.Read("challenger"); err == nil {
			t.Error("removed arm still routes")
		}
		if got, err := Predict(old, Features{Area: 90, Rooms: 3}); err != nil || got.Estimate != 252000 {
			t.Error(got)
		}
		count := files.count("candidate.json")
		time.Sleep(30 * time.Millisecond)
		if files.count("candidate.json") != count {
			t.Error("removed arm still polls")
		}
		if err := router.SetChallenger("challenger.json"); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		before := len(events.snapshot())
		<-root.Stop()
		var stopped []string
		for _, line := range events.snapshot()[before:] {
			if strings.Contains(line, "stopped:") {
				stopped = append(stopped, line)
			}
		}
		want := []string{"model stopped: candidate.json", "switcher stopped: challenger.json",
			"model stopped: baseline.json", "switcher stopped: primary.json"}
		if !slices.Equal(stopped, want) {
			t.Fatalf("shutdown = %q, want %q", stopped, want)
		}
	})
}
