// Command limits makes requests while a file supplies their current quota.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/lucasavila00/immutago/examples/configuration"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// main reports requests until the user sends an interrupt.
func main() {
	path := flag.String("config", "examples/configuration/limits.json", "JSON file with per-key limits")
	key := flag.String("key", "api", "key to limit")
	flag.Parse()
	if err := run(*path, *key); err != nil {
		log.Fatal(err)
	}
}

// run starts the file, configuration, and limiter services in dependency order.
func run(path, key string) error {
	root := supervise.NewRoot(func(value any, _ []byte) { log.Print(value) })
	defer func() { <-root.Stop() }()
	cache, err := supervise.Start[configuration.FileCache](root, configuration.FileArgs{
		Path: path, Every: 100 * time.Millisecond,
	})
	if err != nil {
		return err
	}
	loader, err := supervise.Start[configuration.ConfigLoader](root, configuration.LoaderArgs{
		Cache: cache, Every: 100 * time.Millisecond,
	})
	if err != nil {
		return err
	}
	limiter, err := supervise.Start[configuration.Limiter](root, configuration.LimiterArgs{
		Loader: loader, Key: key, Window: time.Second,
	})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			decision := limiter.Current().Take()
			usage := limiter.Current().Usage()
			fmt.Printf("allowed=%t used=%d limit=%d window=%d\n",
				decision == configuration.Allowed, usage.Used, usage.Limit, usage.Window)
			if problem := loader.Current().Problem(); problem.Len() != 0 {
				fmt.Println("configuration:", problem.Unsafe())
			}
		}
	}
}
