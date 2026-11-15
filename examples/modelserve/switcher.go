package modelserve

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// SwitcherArgs supplies the manifest that selects an arm's model file.
type SwitcherArgs struct {
	Files    fs.FS
	Manifest string
	Every    time.Duration
	Log      *log.Logger
}

// modelVersion owns the background lifetime of one selected model file.
type modelVersion struct {
	path  string
	group *supervise.Group
	model supervise.Ref[Model]
}

// Switcher publishes new versions before it retires old version groups.
type Switcher struct {
	supervise.Service[SwitcherArgs]
	mu      sync.RWMutex
	current modelVersion
	problem string
}

// Init builds this attempt's first version under its own child group.
func (s *Switcher) Init() error {
	if err := s.reload(); err != nil {
		return err
	}
	s.Args().Log.Printf("switcher ready: %s", s.Args().Manifest)
	return nil
}

// Run polls the manifest; malformed edits leave the current version in service.
func (s *Switcher) Run(ctx context.Context) {
	args := s.Args()
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	defer args.Log.Printf("switcher stopped: %s", args.Manifest)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.reload()
		}
	}
}

// Read selects the current model while holding only the short publication lock.
func (s *Switcher) Read() ParametersProjection {
	s.mu.RLock()
	model := s.current.model
	s.mu.RUnlock()
	return model.Borrow().Read()
}

// Status reports errors from either the manifest or the selected model file.
func (s *Switcher) Status() ServiceStatus {
	s.mu.RLock()
	problem, model := s.problem, s.current.model
	s.mu.RUnlock()
	status := model.Borrow().Status()
	if problem != "" {
		return ServiceStatus{Version: status.Version, Problem: im.ProjectString(problem)}
	}
	return status
}

// reload starts a valid replacement before it changes the request route.
func (s *Switcher) reload() error {
	args := s.Args()
	data, err := fs.ReadFile(args.Files, args.Manifest)
	var manifest struct {
		Model string `json:"model"`
	}
	if err == nil {
		err = json.Unmarshal(data, &manifest)
	}
	if err == nil && !fs.ValidPath(manifest.Model) {
		err = fmt.Errorf("manifest needs a relative model file")
	}
	if err != nil {
		s.setProblem(err)
		return err
	}
	s.mu.RLock()
	previous := s.current
	s.mu.RUnlock()
	if previous.path == manifest.Model {
		s.setProblem(nil)
		return nil
	}
	group := supervise.StartGroup(s.Children())
	model, err := supervise.Start[Model](group, ModelArgs{
		Files: args.Files, Path: manifest.Model, Every: args.Every, Log: args.Log,
	})
	if err != nil {
		<-group.Stop()
		s.setProblem(err)
		return err
	}
	s.mu.Lock()
	s.current = modelVersion{path: manifest.Model, group: group, model: model}
	s.problem = ""
	s.mu.Unlock()
	args.Log.Printf("model selected: %s -> %s", args.Manifest, manifest.Model)
	if previous.group != nil {
		<-previous.group.Stop()
	}
	return nil
}

// setProblem records an ordinary file error without changing the selected version.
func (s *Switcher) setProblem(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.problem = ""
	if err != nil {
		s.problem = err.Error()
	}
}
