package main

import (
	"strings"
	"sync"
)

// syncBuf is a buffer a test can read while the code under test writes it.
//
// The package had this shape already — pty_conformance_test.go's ptyOut — and
// #16's first draft hand-rolled an unsynchronised version twice, which `go test
// -race` caught as eleven race reports (ARCH-DRY). One type, used everywhere a
// test observes output produced by another goroutine.
type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func (s *syncBuf) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Len()
}

// TakeAll returns everything written since the last TakeAll and resets.
func (s *syncBuf) TakeAll() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.b.String()
	s.b.Reset()
	return out
}
