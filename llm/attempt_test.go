package llm

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// outcome's ONE order: admission rejection, then the caller's cancel or
// deadline, then the transport's own mapping.
func TestOutcomeOrderUnit(t *testing.T) {
	errShed := errors.New("shed")
	mapped := fmt.Errorf("%w: mapped", ErrTruncated)
	live := context.Background()
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	budget, cancelBudget := context.WithTimeoutCause(context.Background(), time.Nanosecond, errBudget)
	defer cancelBudget()
	<-budget.Done()

	cases := []struct {
		name    string
		caller  context.Context
		hookErr error
		want    []error
		notWant error
	}{
		{"rejection beats a later cancel", gone, errShed, []error{ErrUnavailable, errShed}, ErrCanceled},
		{"rejection with a live caller", live, errShed, []error{ErrUnavailable, errShed}, ErrCanceled},
		{"caller cancel beats the mapping", gone, nil, []error{ErrCanceled, context.Canceled}, ErrTruncated},
		{"own budget keeps the mapping", budget, nil, []error{ErrTruncated}, ErrCanceled},
		{"nothing else: the mapping", live, nil, []error{ErrTruncated}, ErrCanceled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := outcome(tc.caller, &callState{hookErr: tc.hookErr}, mapped)
			for _, w := range tc.want {
				if !errors.Is(got, w) {
					t.Errorf("outcome = %v, want it to wrap %v", got, w)
				}
			}
			if errors.Is(got, tc.notWant) {
				t.Errorf("outcome = %v, must not wrap %v", got, tc.notWant)
			}
		})
	}
}
