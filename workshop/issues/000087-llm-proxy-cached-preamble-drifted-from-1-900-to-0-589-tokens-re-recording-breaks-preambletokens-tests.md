---
id: 000087
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ae194e8b52192af91add72a8e1af2bdecbad6c9c' # card fields mirrored from issue-cards; edit via sdlc
---

# llm: proxy cached preamble drifted from ~1,900 to 0-589 tokens; re-recording breaks PreambleTokens tests

## Problem

## Spec

The parley cli-proxy-api's cached preamble has drifted. On 2026-10-21, captures recorded for tools#85 show `cache_read_input_tokens + cache_creation_input_tokens` of 0 to 589 tokens (the reasoning probe: 0 cached, input_tokens 207). The committed message-thinking.json, recorded earlier, shows ~1,900.

Consequences:
- Re-recording `message-thinking.json` / `message-schema.json` with `scripts/llm-probe.sh record` breaks `TestBuildResponseCarriesUsage` and `TestCompleteRoundTrip`, which assert a nonzero `PreambleTokens`. tools#85 kept the old captures and committed only the new tool-use ones.
- Live `TestCaptureDriftAgainstTheLiveService/the_injected_preamble_has_not_moved_materially` fails, which is the drift check doing its job.

Work: find out what the proxy prepends now (and whether it is still a cached prefix or plain input), decide what `Usage.PreambleTokens` should measure, update the expectation in the drift check, re-record the captures, and check define's prompt-budget assumptions that rest on the ~1,900 figure.

## Done when

- The live drift check passes against the current proxy under `CONFORMANCE_STRICT=1`.
- `scripts/llm-probe.sh record` re-records every capture and `go test ./llm/...` stays green.

## Plan

- [ ]

## Log

### 2026-10-21
