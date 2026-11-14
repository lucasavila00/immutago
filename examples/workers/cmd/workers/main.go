// Command workers replaces and removes groups while they process text jobs.
package main

import (
	"fmt"
	"time"

	"github.com/lucasavila00/immutago/examples/workers"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// main starts two variants, replaces one version, and removes one variant.
func main() {
	root := supervise.NewRoot(nil)
	config, err := supervise.Start[workers.VariantConfig](root, supervise.NoArgs{})
	if err != nil {
		panic(err)
	}
	names := im.FreezeSlice([]string{"blue", "green"})
	config.Current().Set(names)
	dispatcher, err := supervise.Start[workers.Dispatcher](root, workers.DispatcherArgs{
		Config: config, Workers: 2, Every: time.Millisecond,
	})
	if err != nil {
		panic(err)
	}
	dispatcher.Current().Refresh()
	waitVersion(dispatcher, "blue", 1)
	first := dispatcher.Current().Process("blue", "Hello World")
	fmt.Printf("blue v%d: %s\n", first.Version, first.Text)
	if !dispatcher.Current().Load("blue", 2) {
		panic("version 2 did not start")
	}
	waitVersion(dispatcher, "blue", 2)
	next := dispatcher.Current().Process("blue", "Hello World")
	fmt.Printf("blue v%d: %s\n", next.Version, next.Text)
	names = im.FreezeSlice([]string{"green"})
	config.Current().Set(names)
	dispatcher.Current().Refresh()
	removed := dispatcher.Current().Process("blue", "unused")
	fmt.Printf("blue removed: %t\n", !removed.Available)
	waitVersion(dispatcher, "green", 1)
	remaining := dispatcher.Current().Process("green", "Still Running")
	fmt.Printf("green v%d: %s\n", remaining.Version, remaining.Text)
	<-root.Stop()
}

// waitVersion waits for the background dispatcher to publish a prepared pool.
func waitVersion(dispatcher supervise.Ref[workers.Dispatcher], name string, version int) {
	deadline := time.Now().Add(5 * time.Second)
	for dispatcher.Current().Version(name) != version {
		if time.Now().After(deadline) {
			panic("worker pool did not become ready")
		}
		time.Sleep(time.Millisecond)
	}
}
