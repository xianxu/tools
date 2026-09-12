package main

import (
	"fmt"
	"strconv"
	"strings"
)

// maxSoundTimes bounds /sound for the same reason maxHistoryDays bounds
// /history: a fat-fingered "/sound 1000" wedges the session behind twenty
// minutes of playback with Ctrl-C as the only way out.
const maxSoundTimes = 20

// soundUsage is /sound's row in the registry, built from maxSoundTimes so the
// text and the parser share one limit.
var soundUsage = fmt.Sprintf("With nothing, how many times each pronunciation plays. With N, play it N times "+
	"for the rest of this session; 0 turns playback off, and %d is the most.", maxSoundTimes)

// parseSoundArgs reads /sound's optional count. The second return distinguishes
// "set it to n" from "tell me what it is" — 0 is a legitimate value (playback
// off), so a zero count cannot double as "no argument given".
func parseSoundArgs(args []string) (times int, set bool, err error) {
	switch len(args) {
	case 0:
		return 0, false, nil
	case 1:
	default:
		return 0, false, fmt.Errorf("/sound takes one count, not %q", strings.Join(args[1:], " "))
	}

	n, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, false, fmt.Errorf("%q is not a number of times", args[0])
	}
	if n < 0 {
		return 0, false, fmt.Errorf("%d is not a number of times; 0 turns playback off", n)
	}
	if n > maxSoundTimes {
		return 0, false, fmt.Errorf("%d would take a while to sit through; the limit is %d", n, maxSoundTimes)
	}
	return n, true, nil
}

// runSound is /sound: how many times a pronunciation plays, for the rest of this
// session.
func runSound(c commandCtx, args []string) int {
	times, set, err := parseSoundArgs(args)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /sound: %v\n", err)
		return 2
	}
	if !set {
		fmt.Fprintf(c.stdout, "  playing %s\n", soundTimes(c.times))
		return 0
	}
	// One-shot and piped runs have no session to change. Accepting the command
	// silently would be a lie about what it did.
	if c.setTimes == nil {
		fmt.Fprintln(c.stderr, "define: /sound needs an interactive session; use -sound to set it for one run")
		return 2
	}
	c.setTimes(times)
	fmt.Fprintf(c.stdout, "  now playing %s\n", soundTimes(times))
	return 0
}

func soundTimes(n int) string {
	switch n {
	case 0:
		return "nothing — playback is off"
	case 1:
		return "once"
	default:
		return fmt.Sprintf("%d×", n)
	}
}
