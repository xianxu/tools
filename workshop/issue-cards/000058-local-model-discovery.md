---
id: '000058'
status: done
started: 2026-09-13T22:21:55-07:00
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 2.634
actual_hours: 0.84
---

# define: discover local proxy models and select by preference

## Problem

`define` assumes `claude-opus-5` even when a user's local CLIProxyAPI has
only Codex or Antigravity configured. A Homebrew installation should use the
providers available through that user's proxy without requiring a model export.
The preceding invalid-key report is separate: model discovery itself requires
a key accepted by the proxy.
