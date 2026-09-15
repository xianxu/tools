package main

import (
	"context"
	"errors"
	"sync"
	"unicode/utf8"
)

const clipboardTextLimit = 1 << 20

// clipboardWriter finishes a context-bounded literal text write before returning.
type clipboardWriter interface {
	Write(context.Context, string) error
}

// clipboardQueue owns one worker and eight pending copies. Callbacks execute on
// that worker without locks; the console, rather than a callback, owns Stop.
type clipboardQueue struct {
	mu      sync.Mutex
	stopped bool
	pending chan clipboardRequest
	cancel  context.CancelFunc
	done    chan struct{}
}

// clipboardRequest carries one immutable snapshot and its owner's completion.
type clipboardRequest struct {
	text string
	done func(error)
}

func newClipboardQueue(writer clipboardWriter) *clipboardQueue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &clipboardQueue{pending: make(chan clipboardRequest, 8), cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(q.done)
		for {
			select {
			case <-ctx.Done():
				return
			case request := <-q.pending:
				if ctx.Err() != nil {
					return
				}
				err := writer.Write(ctx, request.text)
				if request.done != nil {
					request.done(err)
				}
			}
		}
	}()
	return q
}
func (q *clipboardQueue) Submit(text string, done func(error)) error {
	if err := validateClipboardText(text); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped {
		return errors.New("clipboard is closed")
	}
	select {
	case q.pending <- clipboardRequest{text, done}:
		return nil
	default:
		return errors.New("clipboard queue is full")
	}
}
func (q *clipboardQueue) Stop() {
	q.mu.Lock()
	q.stopped = true
	q.cancel()
	q.mu.Unlock()
	<-q.done
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		select {
		case <-q.pending:
		default:
			return
		}
	}
}
func validateClipboardText(text string) error {
	if len(text) > clipboardTextLimit {
		return errors.New("selection exceeds clipboard size limit")
	}
	if !utf8.ValidString(text) {
		return errors.New("selection is not valid UTF-8")
	}
	return nil
}
