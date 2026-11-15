package modelserve

import (
	"fmt"
	"io/fs"
	"log"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// arm owns a switcher and all of its model versions.
type arm struct {
	group    *supervise.Group
	switcher supervise.Ref[Switcher]
}

// Router selects a primary or optional challenger arm for each request.
// Its mutex protects arm publication; filesystem dependencies must permit concurrent reads.
type Router struct {
	root       *supervise.Group
	args       SwitcherArgs
	mu         sync.RWMutex
	primary    arm
	challenger arm
}

// NewRouter starts the primary arm before HTTP ingress opens.
func NewRouter(root *supervise.Group, files fs.FS, every time.Duration, logger *log.Logger) (*Router, error) {
	if every <= 0 {
		return nil, fmt.Errorf("poll interval must be positive")
	}
	router := &Router{root: root, args: SwitcherArgs{Files: files, Every: every, Log: logger}}
	primary, err := router.start("primary.json")
	if err != nil {
		return nil, err
	}
	router.primary = primary
	return router, nil
}

// SetChallenger publishes a new challenger arm before it stops the previous one.
func (r *Router) SetChallenger(manifest string) error {
	next, err := r.start(manifest)
	if err != nil {
		return err
	}
	r.mu.Lock()
	previous := r.challenger
	r.challenger = next
	r.mu.Unlock()
	if previous.group != nil {
		<-previous.group.Stop()
	}
	return nil
}

// RemoveChallenger stops only the challenger subtree after it removes the route.
func (r *Router) RemoveChallenger() {
	r.mu.Lock()
	previous := r.challenger
	r.challenger = arm{}
	r.mu.Unlock()
	if previous.group != nil {
		<-previous.group.Stop()
	}
}

// Read selects one immutable model; later replacement does not change this result.
func (r *Router) Read(name string) (ParametersProjection, error) {
	selected, err := r.selectArm(name)
	if err != nil {
		return ParametersProjection{}, err
	}
	return selected.switcher.Borrow().Read(), nil
}

// ArmStatus contains the ordinary JSON view of one selected arm.
type ArmStatus struct {
	Version string `json:"version"`
	Problem string `json:"problem,omitempty"`
}

// Status reports currently routed arms and their latest file errors.
func (r *Router) Status() map[string]ArmStatus {
	result := make(map[string]ArmStatus)
	for _, name := range []string{"primary", "challenger"} {
		selected, err := r.selectArm(name)
		if err != nil {
			continue
		}
		model := selected.switcher.Borrow().Read()
		problem := selected.switcher.Borrow().Status()
		result[name] = ArmStatus{Version: model.Version().Unsafe(), Problem: problem.Unsafe()}
	}
	return result
}

// selectArm copies a reference so requests do not hold the router lock during work.
func (r *Router) selectArm(name string) (arm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	switch name {
	case "primary":
		return r.primary, nil
	case "challenger":
		if r.challenger.group != nil {
			return r.challenger, nil
		}
	}
	return arm{}, fmt.Errorf("arm %q is not available", name)
}

// start gives an arm one group for selective shutdown; the group adds no retry policy.
func (r *Router) start(manifest string) (arm, error) {
	args := r.args
	args.Manifest = manifest
	group := supervise.StartGroup(r.root)
	ref, err := supervise.Start[Switcher](group, args)
	if err != nil {
		<-group.Stop()
		return arm{}, err
	}
	return arm{group: group, switcher: ref}, nil
}
