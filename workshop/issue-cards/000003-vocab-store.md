---
id: '000003'
status: done
started: 2026-08-20T17:51:17-07:00
created: 2026-08-20
updated: 2026-08-21
estimate_hours: 1.61
actual_hours: 1.67
---

# vocabulary store: per-user YAML deck in a brain, behind a Store seam

## Problem

The vocabulary features all need per-user persistent state, and none of them
should know where it lives or what format it is in. A database is the eventual
destination; YAML files in a git repo are the quick, inspectable start.
