package main

import (
	"context"
	"errors"
	"sync"
)

// fakeFeed is the stateful double for the news feed (ARCH-MOCK).
//
// It serves canned RSS **bytes**, not parsed items: a fake returning the parsed
// type would take parseRSS out of the path, and the parser is where the risk
// lives — malformed feeds are a Done-when row.
//
// It counts fetches per word, which is what lets "served from cache" be asserted
// on the fake's own state rather than inferred from output. Inferring it is how
// a cache that silently re-fetches passes its own test.
type fakeFeed struct {
	mu    sync.Mutex
	body  []byte
	err   error
	calls map[string]int
}

func newFakeFeed(body []byte) *fakeFeed {
	return &fakeFeed{body: body, calls: map[string]int{}}
}

func (f *fakeFeed) Fetch(_ context.Context, word string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[word]++
	if f.err != nil {
		return nil, f.err
	}
	return f.body, nil
}

// fetches is how many times the network was actually reached for a word.
func (f *fakeFeed) fetches(word string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[word]
}

// failWith makes every later fetch fail, so the not-cached-on-failure rule and
// the stale-fallback rule can each be driven.
func (f *fakeFeed) failWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeFeed) succeed() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = nil
}

var errFeedDown = errors.New("feed unreachable")
