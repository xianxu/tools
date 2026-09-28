---
id: '000070'
status: done
started: 2026-09-17T17:20:31-07:00
created: 2026-09-16
updated: 2026-09-18
estimate_hours: 6.1
actual_hours: 10.05
---

# define: switch between a light and a dark colour scheme

## Problem

`define` is drawn for a dark terminal. The one colour that cannot follow a light
terminal is the target-language tint: xterm-256 colour 236, a FIXED code the
terminal's theme does not remap, so on a white background every tinted row is a
dark bar. `-language-tint light` exists (#66), but only as a launch flag that
defaults to `dark` — nothing notices the terminal is light, nothing remembers the
choice, and nothing changes it mid-session.
