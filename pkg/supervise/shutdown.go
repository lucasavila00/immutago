package supervise

import "fmt"

// ShutdownError reports the wait in progress when the Shutdown context ended.
// Use errors.As to find it: a *Failure can also wrap a context error.
// Waiting is the Go type string of the service that had not stopped, or
// "accepted Init calls" when the supervisor waited for Init. It is empty when no wait was in progress.
// Err is the context error.
type ShutdownError struct {
	Waiting string
	Err     error
}

// Error names the wait and the context error.
func (e *ShutdownError) Error() string {
	if e.Waiting == "" {
		return fmt.Sprintf("supervise: shutdown did not complete: %v", e.Err)
	}
	return fmt.Sprintf("supervise: shutdown waited for %s: %v", e.Waiting, e.Err)
}

// Unwrap returns the context error.
func (e *ShutdownError) Unwrap() error { return e.Err }

// terminal returns the first failure of a stopped root, or nil.
func (s *Supervisor) terminal() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.root == s && s.failure != nil {
		return s.failure
	}
	return nil
}

// blocked names the wait of a supervisor in shutdown. The caller holds the tree lock.
// It follows the last entry in progress into its nested or service-owned supervisor.
func (s *Supervisor) blocked() string {
	select {
	case <-s.stopped:
		return ""
	default:
	}
	c := s.waiting
	if c == nil {
		if s.stopping && !s.halting && s.accepted > 0 {
			return "accepted Init calls"
		}
		return ""
	}
	if c.group != nil {
		return c.group.blocked()
	}
	if c.done != nil {
		select {
		case <-c.done:
			return ""
		default:
		}
	}
	if c.owned != nil {
		if name := c.owned.blocked(); name != "" {
			return name
		}
	}
	return c.name
}
