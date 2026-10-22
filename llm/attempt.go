package llm

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// callState is ONE call's private record: how many attempts went out, and
// whether Config.BeforeAttempt turned one away. It rides the call's context to
// the SDK middleware, so concurrent calls on one client never share it.
//
// The abort exists because the SDK RETRIES a middleware that returns no response
// (requestconfig.go shouldRetry: res == nil means a connection error), so a
// rejection returned plainly would buy backoff and two more hook calls. Instead
// the middleware records the error and cancels the call's own derived context;
// the SDK's retry loop checks ctx.Err() right after the handler and exits before
// any backoff.
type callState struct {
	task, model string
	abort       context.CancelFunc

	mu       sync.Mutex
	number   int // hook invocations, so Attempt.Number counts retries
	attempts int // admitted and dispatched
	hookErr  error
}

type callStateKey struct{}

// beginCall derives the context the SDK runs on and attaches the call's state.
func beginCall(ctx context.Context, r Request) (context.Context, *callState) {
	ctx, abort := context.WithCancel(ctx)
	st := &callState{task: r.Task, model: r.Model, abort: abort}
	return context.WithValue(ctx, callStateKey{}, st), st
}

func (st *callState) dispatched() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.attempts
}

func (st *callState) rejection() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.hookErr
}

// attemptMiddleware runs once per physical HTTP attempt, inside the SDK's retry
// loop. Discovery requests carry no callState and pass straight through.
func attemptMiddleware(hook func(context.Context, Attempt) error) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		st, _ := req.Context().Value(callStateKey{}).(*callState)
		if st == nil {
			return next(req)
		}
		st.mu.Lock()
		n := st.number
		st.number++
		st.mu.Unlock()
		if hook != nil {
			if err := hook(req.Context(), Attempt{Task: st.task, Model: st.model, Number: n}); err != nil {
				// A hook that gave up because the call's context ended has not
				// rejected anything: that outcome is the cancel's, and the caller
				// check reports it.
				if req.Context().Err() == nil {
					st.mu.Lock()
					st.hookErr = err
					st.mu.Unlock()
					st.abort()
				}
				return nil, err
			}
		}
		st.mu.Lock()
		st.attempts++
		st.mu.Unlock()
		return next(req)
	}
}

// outcome decides how a failed call ended, in ONE fixed order: an admission
// rejection, then the caller's cancel or deadline, then whatever the transport
// mapped (stall, budget, status). One owner, so no two paths can disagree.
func outcome(caller context.Context, st *callState, mapped error) error {
	if err := st.rejection(); err != nil {
		return fmt.Errorf("%w: admission: %w", ErrUnavailable, err)
	}
	if callerDone(caller) {
		return canceled(caller.Err())
	}
	return mapped
}
