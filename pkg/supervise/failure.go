package supervise

import (
	"fmt"
	"os"
)

// Failure identifies the service whose terminal failure stopped the root.
// Value is its original Run panic value or a runtime.Goexit error.
// Stack is captured in that service's goroutine. Groups pass the same failure
// to the root without wrapping it.
type Failure struct {
	Service string
	Value   any
	Stack   []byte
}

// Error names the service and its terminal failure.
func (f *Failure) Error() string {
	return fmt.Sprintf("supervise: terminal failure in %s: %v", f.Service, f.Value)
}

// Unwrap returns the original failure when it is an error.
func (f *Failure) Unwrap() error {
	err, _ := f.Value.(error)
	return err
}

// reportFailure reports one recovered failure before its service waits for children.
func (s *Group) reportFailure(failure *Failure) {
	if s.report != nil {
		s.report(failure, failure.Stack)
		return
	}
	fmt.Fprintf(os.Stderr, "%v\n%s", failure, failure.Stack)
}
