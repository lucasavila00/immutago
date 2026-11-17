package trading

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io/fs"
	"log"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Options supplies the host's venue adapter and the two operator configuration files.
type Options struct {
	API   API
	Files fs.FS
	Every time.Duration
	Log   *log.Logger
}

// Bot owns the service root and retains unresolved account state after failed cleanup.
type Bot struct {
	root     *supervise.Supervisor
	ref      supervise.Ref[Trading]
	state    state
	mu       sync.Mutex
	problem  error
	stopOnce sync.Once
	done     chan struct{}
}

// Start starts policy before the strategy tree, so policy stops after its consumers.
func Start(options Options) (*Bot, error) {
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	bot := &Bot{state: state{session: binary.LittleEndian.Uint64(nonce[:])}, done: make(chan struct{})}
	bot.root = supervise.NewRoot(func(value any, stack []byte) {
		options.Log.Printf("service failed: %v\n%s", value, stack)
	})
	policy, err := supervise.Start[Policy](bot.root, PolicyArgs{
		Files: options.Files, Every: options.Every, Log: options.Log,
	})
	if err == nil {
		bot.ref, err = supervise.Start[Trading](bot.root, tradingArgs{
			Files: options.Files, API: options.API, Every: options.Every, Log: options.Log,
			Policy: policy, State: &bot.state, StopBot: bot.stopWithError,
		})
	}
	if err != nil {
		<-bot.root.Stop()
		return nil, err
	}
	return bot, nil
}

// Read selects a complete report without sharing account locks with its caller.
func (b *Bot) Read() ReportProjection { return b.ref.Borrow().Read() }

// Stop waits for reverse-order cleanup and reports any unconfirmed venue cancellation.
// The error names the account, instrument, and unresolved IDs. Their state remains in Bot.
func (b *Bot) Stop() error {
	b.requestStop()
	<-b.done
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.problem
}

// stopWithError records cleanup failures and requests shutdown without waiting on its caller.
func (b *Bot) stopWithError(err error) {
	b.mu.Lock()
	b.problem = errors.Join(b.problem, err)
	b.mu.Unlock()
	b.requestStop()
}

// Done closes after requested or failed-cleanup shutdown has finished.
func (b *Bot) Done() <-chan struct{} { return b.done }

// requestStop starts one waiter outside the service tree and never waits on a child caller.
func (b *Bot) requestStop() {
	b.stopOnce.Do(func() {
		finished := b.root.Stop()
		go func() {
			<-finished
			close(b.done)
		}()
	})
}
