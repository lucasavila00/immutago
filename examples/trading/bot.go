package trading

import (
	"crypto/rand"
	"encoding/binary"
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
	root    *supervise.Group
	ref     supervise.Ref[Trading]
	state   state
	mu      sync.Mutex
	problem error
}

// Start starts policy before the strategy tree, so policy stops after its consumers.
func Start(options Options) (*Bot, error) {
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	bot := &Bot{state: state{session: binary.LittleEndian.Uint64(nonce[:])}}
	bot.root = supervise.NewRoot(func(value any, _ []byte) { options.Log.Printf("attempt failed: %v", value) })
	policy, err := supervise.Start[Policy](bot.root, PolicyArgs{Files: options.Files, Every: options.Every, Log: options.Log})
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
	<-b.root.Stop()
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.problem
}

// stopWithError records the first cleanup failure and requests shutdown without waiting on its caller.
func (b *Bot) stopWithError(err error) {
	b.mu.Lock()
	if b.problem == nil {
		b.problem = err
	}
	b.mu.Unlock()
	b.root.Stop()
}
