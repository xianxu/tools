---
id: 000052
status: open
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
github_issue:
---

# run the model on this Mac: an Apple fm backend for macOS 27

## Problem

Every model call define makes leaves the machine. `llm.Resolve`
(`internal/llm/config.go`) knows one transport, Anthropic's Messages API, reached
through the parley proxy or with a key, and `anthropic.go`'s `New` is the only
`llm.Client`. Banding, authoring, the entail and veto judges, the learner model and
questions all need it; without the proxy or a key they do nothing.

macOS 27 (ships 2026-09-14) puts Apple's on-device model on the command line:
`fm`, preinstalled, free, offline, no key. Apple says the on-device model was
"rebuilt from the ground up" for this release, and its WWDC example shows an
8,192-token context. That is plausibly enough model for the classification-shaped
tasks, and it makes the call local.

Decided 2026-09-12: target macOS 27's `fm` only. No Swift helper for macOS 26's
Foundation Models framework (a Swift-only API, a 4,096-token context, the older
model).
