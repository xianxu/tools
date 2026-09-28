---
id: '000046'
status: done
started: 2026-09-07T12:42:34-07:00
created: 2026-09-07
updated: 2026-09-07
estimate_hours: 7.02
actual_hours: 5.91
---

# cached, durable and clickable pronunciation audio

## Problem

Three gaps found by hand-running a sitting, all on the same axis: **the sound a
learner wants is either not reachable, or fetched again.**

1. **A sitting fetches every recording afresh.** `repl.go:257` and
   `replraw.go:264` each wrap `d.audio = newCachingAudioSource(d.audio)`.
   `runPlay` (`play_loop.go:24`) does not. So the one loop that plays the SAME
   handful of words over and over is the one loop with no cache — and a word the
   CDN has no recording for costs four candidate requests on every replay,
   because the `misses` set that exists to prevent exactly that is never
   constructed.

2. **The cache dies with the process.** `cachingAudioSource` already models both
   levels the learner asked for — `hits` (the bytes) and `misses` (whether a
   fetch is known to be pointless) — but in two maps that live as long as one
   command. A deck reviewed daily re-fetches the same recordings daily, and
   re-asks the same permanent 404s daily. The distinction the type documents
   ("a miss is PERMANENT — unlike a transport failure") is the argument for
   persisting it, and the type makes it without acting on it.

3. **A cloze's option words cannot be clicked.** Every other word on screen can:
   `Region` is a click registry, `playRegion` already fetches and plays, and
   `TestEveryRegionKindIsActionable` derives its loop from `numRegionKinds`. But
   regions are produced by `Render(entry)` alone, and `marksIn` finds them by
   locating that rendered entry inside the written text — so they attach to the
   REVEAL and never to a prompt. A cloze prompt is a blanked sentence and four
   words, none of which is the rendered entry; the four words a learner is
   choosing between are the four they most want to hear.
