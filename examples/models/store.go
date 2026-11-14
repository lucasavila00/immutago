package models

import "os"

// Store saves a validated model document before it becomes current.
type Store interface {
	Save(document string) error
}

// FileStore keeps the last validated model document in Path.
type FileStore struct {
	Path string
}

// Save replaces the saved document. Give each Fetcher its own output path.
func (s FileStore) Save(document string) error {
	return os.WriteFile(s.Path, []byte(document), 0600)
}
