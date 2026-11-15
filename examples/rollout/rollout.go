package rollout

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Args supplies the configuration directory and the interval between file reads.
type Args struct {
	Files  fs.FS
	Every  time.Duration
	Log    *log.Logger
	Policy supervise.Ref[Policy]
}

// runningDeployment keeps selective shutdown and the current service reference together.
type runningDeployment struct {
	spec  deploymentSpec
	group *supervise.Group
	ref   supervise.Ref[Deployment]
}

// Rollout owns release selection, the live gate, and both deployment lifetimes.
// Init and Run alone change its fields; HTTP readers use the atomic ReportCell.
type Rollout struct {
	supervise.Service[Args]
	deployments [2]runningDeployment
	updates     chan struct{}
	problem     string
	current     ReportCell
}

// Init builds a complete first selection under this attempt's child group.
func (r *Rollout) Init() error {
	if r.Args().Every <= 0 {
		return fmt.Errorf("file read interval must be positive")
	}
	r.updates = make(chan struct{}, 1)
	config, err := readConfiguration(r.Args().Files)
	if err != nil {
		return err
	}
	return r.selectDeployments(config)
}

// Run applies operator edits and evaluates reports when deployment data changes.
func (r *Rollout) Run(ctx context.Context) {
	args := r.Args()
	args.Log.Print("rollout started")
	defer args.Log.Print("rollout stopped")
	ticker := time.NewTicker(args.Every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.updates:
			r.publish()
		case <-ticker.C:
			config, err := readConfiguration(args.Files)
			if err == nil {
				err = r.selectDeployments(config)
			}
			if err != nil && r.problem != err.Error() {
				r.problem = err.Error()
				args.Log.Printf("configuration: %v", err)
				r.publish()
			}
		}
	}
}

// Read selects one immutable report. It remains valid after retirement or restart.
func (r *Rollout) Read() ReportProjection { return r.current.Load() }

// selectDeployments reuses existing deployments, starts additions, publishes the
// new selection, and then stops retired trees. Both groups keep the same owner.
func (r *Rollout) selectDeployments(config configuration) error {
	desired := []*deploymentSpec{&config.Primary, config.Candidate}
	var next [2]runningDeployment
	var added []*supervise.Group
	for i, spec := range desired {
		if spec == nil {
			continue
		}
		for _, current := range r.deployments {
			if current.group != nil && sameDeployment(current.spec, *spec) {
				next[i] = current
				break
			}
		}
		if next[i].group != nil {
			continue
		}
		started, err := r.startDeployment(*spec)
		if err != nil {
			for _, group := range added {
				<-group.Stop()
			}
			return err
		}
		next[i] = started
		added = append(added, started.group)
	}
	old := r.deployments
	r.deployments, r.problem = next, ""
	r.publish()
	for _, previous := range old {
		if previous.group != nil && previous.group != next[0].group && previous.group != next[1].group {
			<-previous.group.Stop()
		}
	}
	return nil
}

// startDeployment creates one removable subtree; failed setup stops its staged children.
func (r *Rollout) startDeployment(spec deploymentSpec) (runningDeployment, error) {
	args := r.Args()
	group := supervise.StartGroup(r.Children())
	ref, err := supervise.Start[Deployment](group, DeploymentArgs{
		Files: args.Files, Spec: spec, Every: args.Every, Updates: r.updates, Log: args.Log,
	})
	if err != nil {
		<-group.Stop()
		return runningDeployment{}, err
	}
	return runningDeployment{spec: spec, group: group, ref: ref}, nil
}

// publish copies numeric service results into one complete immutable report.
func (r *Rollout) publish() {
	primary := r.deployments[0]
	first := primary.ref.Borrow().Read()
	policy := r.Args().Policy.Borrow().Read()
	var candidate *Release
	verdict, reason := "no_candidate", "Add a candidate to compare releases."
	if second := r.deployments[1]; second.group != nil {
		counts := second.ref.Borrow().Read()
		candidate = &Release{Version: second.spec.Version, Replicas: len(second.spec.Logs), Totals: counts}
		verdict, reason = evaluate(first, counts, policy.Limits())
	}
	r.current.Store(Report{
		Primary:   Release{Version: primary.spec.Version, Replicas: len(primary.spec.Logs), Totals: first},
		Candidate: candidate, Verdict: verdict, Reason: reason, Problem: r.problem,
		PolicyVersion: policy.Version().Unsafe(), PolicyProblem: policy.Problem().Unsafe(),
	})
}
