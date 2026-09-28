---
id: '000043'
status: done
started: 2026-09-01T20:23:56-07:00
created: 2026-09-01
updated: 2026-09-01
actual_hours: N/A
---

# Declare Couch fleet policy

## Problem

Couch refuses to start an actor in the tools repository because tools has no
`.sdlc/fleet.json` declaration. Fleet admission intentionally fails closed when
the repository has not declared its concurrency policy.
