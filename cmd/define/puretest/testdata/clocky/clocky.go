// Package clocky exists to be REJECTED: it imports only `time`, which any
// allowlist permits, and then reads the wall clock — the hazard an import list
// structurally cannot see.
package clocky

import "time"

func Elapsed(since time.Time) time.Duration { return time.Since(since) }
