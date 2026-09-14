package llm

import (
	"context"
	"fmt"
	"sync"
)

// autoState is nil before selection, an owned flight during discovery, or an
// immutable selected delegate after success. Failures return to the nil state.
type autoState interface{ modelState() }
type modelFlight struct {
	done   chan struct{}
	result *selectedModel
	err    error
}

func (*modelFlight) modelState() {}

type selectedModel struct {
	selection ModelSelection
	client    Client
}

func (*selectedModel) modelState() {}

type autoClient struct {
	cfg   Config
	mu    sync.Mutex
	state autoState
}

func newAutoClient(c Config) Client { return &autoClient{cfg: c} }

// SelectionOf reports a client's pinned default, without performing discovery.
// An empty result means automatic selection has not succeeded or the client does
// not expose selection metadata. Per-request overrides do not change this value.
func SelectionOf(c Client) ModelSelection {
	if reporter, ok := c.(interface{ modelSelection() ModelSelection }); ok {
		return reporter.modelSelection()
	}
	return ModelSelection{}
}
func (a *anthropicClient) modelSelection() ModelSelection {
	return ModelSelection{Provider: a.cfg.provider, ID: a.cfg.Model}
}
func (a *autoClient) modelSelection() ModelSelection {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.state.(*selectedModel); ok {
		return s.selection
	}
	return ModelSelection{}
}
func selectionCancelled(ctx context.Context) error {
	return fmt.Errorf("%w: model selection: %w", ErrUnavailable, ctx.Err())
}

func (a *autoClient) selected(ctx context.Context) (Client, error) {
	if ctx.Err() != nil {
		return nil, selectionCancelled(ctx)
	}
	a.mu.Lock()
	switch s := a.state.(type) {
	case *selectedModel:
		a.mu.Unlock()
		return s.client, nil
	case *modelFlight:
		a.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, selectionCancelled(ctx)
		case <-s.done:
			if ctx.Err() != nil {
				return nil, selectionCancelled(ctx)
			}
			if s.err != nil {
				return nil, s.err
			}
			return s.result.client, nil
		}
	}
	flight := &modelFlight{done: make(chan struct{})}
	a.state = flight
	a.mu.Unlock()
	selection, err := discoverModels(ctx, a.cfg)
	var result *selectedModel
	if err == nil {
		c := a.cfg
		c.AutoModel = false
		c.Model = selection.ID
		c.provider = selection.Provider
		result = &selectedModel{selection: selection, client: New(c)}
	}
	a.mu.Lock()
	flight.result, flight.err = result, err
	if err == nil {
		a.state = result
	} else {
		a.state = nil
	}
	close(flight.done)
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return result.client, nil
}

func (a *autoClient) requestClient(ctx context.Context, r Request) (Client, error) {
	if ctx.Err() != nil {
		return nil, selectionCancelled(ctx)
	}
	if r.Model != "" {
		c := a.cfg
		c.AutoModel = false
		c.Model = r.Model
		c.provider = ""
		return New(c), nil
	}
	return a.selected(ctx)
}
func (a *autoClient) Complete(ctx context.Context, r Request) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()
	c, err := a.requestClient(ctx, r)
	if err != nil {
		return Response{}, err
	}
	return c.Complete(ctx, r)
}
func (a *autoClient) Stream(ctx context.Context, r Request, onDelta func(string)) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()
	c, err := a.requestClient(ctx, r)
	if err != nil {
		return Response{}, err
	}
	return c.Stream(ctx, r, onDelta)
}
