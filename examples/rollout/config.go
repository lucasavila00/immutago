package rollout

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
)

// deploymentSpec names one release and its append-only replica logs.
type deploymentSpec struct {
	Version string   `json:"version"`
	Logs    []string `json:"logs"`
}

// configuration is the operator's desired selection, not the service state.
type configuration struct {
	Primary   deploymentSpec  `json:"primary"`
	Candidate *deploymentSpec `json:"candidate"`
}

// readConfiguration checks a complete edit before any service starts or stops.
func readConfiguration(files fs.FS) (configuration, error) {
	data, err := fs.ReadFile(files, "rollout.json")
	if err != nil {
		return configuration{}, err
	}
	var config configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return configuration{}, err
	}
	if err := checkDeployment(config.Primary); err != nil {
		return configuration{}, err
	}
	if config.Candidate != nil {
		if err := checkDeployment(*config.Candidate); err != nil {
			return configuration{}, err
		}
		if config.Candidate.Version == config.Primary.Version {
			return configuration{}, fmt.Errorf("primary and candidate need different versions")
		}
	}
	return config, nil
}

// checkDeployment requires a named release and distinct relative log paths.
func checkDeployment(spec deploymentSpec) error {
	if spec.Version == "" || len(spec.Logs) == 0 {
		return fmt.Errorf("deployment needs a version and at least one replica log")
	}
	for i, path := range spec.Logs {
		if !fs.ValidPath(path) || slices.Contains(spec.Logs[:i], path) {
			return fmt.Errorf("deployment %s has an invalid or repeated log path %q", spec.Version, path)
		}
	}
	return nil
}

// sameDeployment permits a current candidate to become primary without a restart.
func sameDeployment(a, b deploymentSpec) bool {
	return a.Version == b.Version && slices.Equal(a.Logs, b.Logs)
}
