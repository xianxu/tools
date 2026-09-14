package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// selected is tested directly so no context.WithTimeout wrapping can trigger
// this barrier before the actual flight waiter evaluates its cancellation case.
type selectionWaitContext struct {
	context.Context
	attached chan struct{}
	once     sync.Once
}

func (c *selectionWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.attached) })
	return c.Context.Done()
}

func TestAutoSelectionFlightPublication(t *testing.T) {
	for _, cancelOwner := range []bool{false, true} {
		name := "success"
		if cancelOwner {
			name = "owner cancellation"
		}
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{}, 2)
			release := make(chan struct{})
			var releaseOnce sync.Once
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.6","owned_by":"openai"}]}`))
			}))
			defer func() { releaseOnce.Do(func() { close(release) }); server.Close() }()
			a := &autoClient{cfg: Config{BaseURL: server.URL, APIKey: "test-key", Timeout: time.Second}}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			owner := make(chan error, 1)
			go func() { _, err := a.selected(ctx); owner <- err }()
			<-started
			const callers = 8
			results := make(chan error, callers)
			for i := 0; i < callers; i++ {
				waiter := &selectionWaitContext{Context: t.Context(), attached: make(chan struct{})}
				go func() { _, err := a.selected(waiter); results <- err }()
				<-waiter.attached
			}
			if cancelOwner {
				cancel()
			} else {
				releaseOnce.Do(func() { close(release) })
			}
			ownerErr := <-owner
			if cancelOwner && !errors.Is(ownerErr, context.Canceled) {
				t.Fatalf("owner error=%v", ownerErr)
			}
			if !cancelOwner && ownerErr != nil {
				t.Fatal(ownerErr)
			}
			for i := 0; i < callers; i++ {
				err := <-results
				if err != ownerErr {
					t.Fatalf("waiter did not receive flight outcome: %v; owner %v", err, ownerErr)
				}
			}
			select {
			case <-started:
				t.Fatal("attached waiter started a separate discovery")
			default:
			}
			if cancelOwner {
				if SelectionOf(a) != (ModelSelection{}) {
					t.Fatal("failed flight pinned selection")
				}
				releaseOnce.Do(func() { close(release) })
				if _, err := a.selected(t.Context()); err != nil {
					t.Fatalf("later retry: %v", err)
				}
				select {
				case <-started:
				default:
					t.Fatal("later request did not retry discovery")
				}
			}
			if got := SelectionOf(a); got != (ModelSelection{"openai", "gpt-5.6"}) {
				t.Fatalf("selection=%+v", got)
			}
			if _, err := a.selected(t.Context()); err != nil {
				t.Fatal(err)
			}
			select {
			case <-started:
				t.Fatal("selected client rediscovered")
			default:
			}
		})
	}
}

func TestAutoSelectionWaiterCancellationPreservesOwner(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.6","owned_by":"openai"}]}`))
	}))
	defer func() { releaseOnce.Do(func() { close(release) }); server.Close() }()
	a := &autoClient{cfg: Config{BaseURL: server.URL, APIKey: "test-key", Timeout: time.Second}}
	owner := make(chan error, 1)
	go func() { _, err := a.selected(t.Context()); owner <- err }()
	<-started
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	waiter := &selectionWaitContext{Context: ctx, attached: make(chan struct{})}
	result := make(chan error, 1)
	go func() { _, err := a.selected(waiter); result <- err }()
	<-waiter.attached
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter=%v", err)
	}
	select {
	case err := <-owner:
		t.Fatalf("waiter cancellation ended owner's flight: %v", err)
	default:
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-owner; err != nil {
		t.Fatalf("owner=%v", err)
	}
	select {
	case <-started:
		t.Fatal("waiter started second discovery")
	default:
	}
	if got := SelectionOf(a); got.ID != "gpt-5.6" {
		t.Fatalf("owner did not publish: %+v", got)
	}
}
