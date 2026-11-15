package search

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// IndexerArgs supplies the document directory and the scan interval.
type IndexerArgs struct {
	Files fs.FS
	Every time.Duration
}

// Status reports the current index and the last scan error.
type Status struct {
	im.Local
	Version   int
	Documents int
	Problem   im.StringProjection
}

// Indexer builds complete versions while callers read the published index.
type Indexer struct {
	supervise.Service[IndexerArgs]
	current IndexCell
	last    []sourceDocument
	version int
	mu      sync.RWMutex
	problem string
}

// Init requires a complete first scan before the service starts.
func (s *Indexer) Init() error {
	if s.Args().Every <= 0 {
		return fmt.Errorf("scan interval must be positive")
	}
	return s.reload()
}

// Run keeps the last valid index on an error and tries again at the next tick.
func (s *Indexer) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Args().Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.reload()
		}
	}
}

// Read selects one complete version for a query.
func (s *Indexer) Read() IndexProjection { return s.current.Load() }

// Status reads the publication and error under the short publication lock.
func (s *Indexer) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index := s.current.Load()
	return Status{
		Version: index.Version(), Documents: index.Documents().Len(),
		Problem: im.ProjectString(s.problem),
	}
}

func (s *Indexer) reload() error {
	documents, err := scanDocuments(s.Args().Files)
	if err != nil {
		s.mu.Lock()
		s.problem = err.Error()
		s.mu.Unlock()
		return err
	}
	if s.version > 0 && slices.Equal(documents, s.last) {
		s.mu.Lock()
		s.problem = ""
		s.mu.Unlock()
		return nil
	}
	index := buildIndex(documents, s.version+1)
	s.mu.Lock()
	s.current.Store(index)
	s.problem = ""
	s.mu.Unlock()
	s.last = documents
	s.version++
	return nil
}
