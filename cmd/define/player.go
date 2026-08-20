package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// ErrNoPlayer means the platform's audio player is unavailable. The shell
// downgrades this to a warning: a missing player does not make a successful
// lookup into a failed one.
var ErrNoPlayer = errors.New("no audio player available")

// Player plays a local audio file, start to finish.
//
// The repeat loop deliberately lives OUTSIDE this interface (see playN), so the
// number of plays is a property of the shell that a fake can count. A Play(n int)
// signature would hide the count inside the thing under test.
type Player interface {
	Play(ctx context.Context, path string) error
}

// gapBetweenPlays separates repeats so they are distinguishable by ear rather
// than running together as one long sound.
const gapBetweenPlays = 250 * time.Millisecond

type afplayPlayer struct{}

func (afplayPlayer) Play(ctx context.Context, path string) error {
	bin, err := exec.LookPath("afplay")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNoPlayer, err)
	}
	if err := exec.CommandContext(ctx, bin, path).Run(); err != nil {
		return fmt.Errorf("afplay: %w", err)
	}
	return nil
}

// playN plays the file n times, pausing between repeats but not after the last.
// It stops at the first failure rather than pressing on through a broken player.
func playN(ctx context.Context, p Player, path string, n int) error {
	for i := 0; i < n; i++ {
		if err := p.Play(ctx, path); err != nil {
			return err
		}
		if i == n-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(gapBetweenPlays):
		}
	}
	return nil
}
