package shipping

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Args selects the catalog file and the interval between reload attempts.
type Args struct {
	Path  string
	Every time.Duration
}

// Loader publishes complete catalogs and keeps the last valid catalog on an error.
type Loader struct {
	supervise.Service[Args]
	current CatalogCell
}

// Init loads the first valid catalog before the service becomes available.
func (l *Loader) Init(*supervise.Supervisor) error {
	if l.Args().Every <= 0 {
		return fmt.Errorf("reload interval must be positive")
	}
	return l.reload(l.Args().Path)
}

// Run reloads the file until shutdown. A failed edit leaves current quotes available.
func (l *Loader) Run(ctx context.Context) {
	ticker := time.NewTicker(l.Args().Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := l.reload(l.Args().Path); err != nil {
				log.Printf("shipping catalog: %v", err)
			}
		}
	}
}

// Catalog selects one version for all reads in a quote calculation.
func (l *Loader) Catalog() CatalogProjection {
	return l.current.Load()
}

// reload parses and checks a whole file before replacing the published catalog.
func (l *Loader) reload(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document catalogDocument
	if err := decodeJSON(bytes.NewReader(data), &document); err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	if err := validateCatalog(document); err != nil {
		return err
	}
	l.current.Store(buildCatalog(document))
	return nil
}
