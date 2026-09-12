---
id: 000052
status: open
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
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

## Spec

A second `llm.Client` that execs `fm respond`, beside `anthropic.go`, chosen by
configuration. What is known of the CLI (WWDC26 session 334):

    fm respond "<prompt>" --instructions "<system>" --schema schema.json [--model pcc]

- `--schema` constrains the output, and the answer is JSON. `fm schema object
  --name X --string field [--array]` generates a schema file.
- On-device is the default. `--model pcc` sends the request to Private Cloud
  Compute: a much bigger model with reasoning and a 32K context, but off the
  machine and under usage limits.

How define's request maps:

| `llm.Request` | `fm respond` |
|---|---|
| `System` | `--instructions` |
| `Prompt` | positional argument (or stdin, if supported) |
| `Schema` | `--schema <file>`, written from the schema `llm.SchemaFor` derives |
| `Model` | `--model` (on-device by default, or `pcc`) |
| `Effort`, `MaxTokens` | no documented equivalent |

Open questions that only the real CLI answers. They decide the design, so they are
probed before any plan:

1. **Schema dialect.** Does `--schema` accept the JSON Schema define already
   derives (enums, required fields, nested objects, descriptions), or only what
   `fm schema` emits? If only the latter, a translator is part of the work.
   Output that fails to parse is `ErrMalformed`, as the `Request.Schema` contract
   already requires callers to handle.
2. **Input and output.** Can the prompt come from stdin? Does output stream (the
   `Stream` half of `llm.Client`, which questions use)? Is token usage reported?
3. **Failure.** What exit status and message when the model is unavailable (Apple
   Intelligence off, the model not yet downloaded) or the context overflows? These
   must classify as `ErrUnavailable` and degrade quietly, as a dead proxy does.
4. **Context.** Confirm 8,192 on this Mac. Prompt sizes measured from the request
   goldens (characters / 4): ask ~310 tokens, band ~400, author ~540, entail ~620,
   all well inside. The learner model is ~740 plus ~11 per deck word, so it fits
   up to roughly 600 deck words; past that it needs a cap or chunking.

Which tasks move is measured, not assumed. Run the harvest and reflect conformance
tests against the on-device model (and against `pcc`), and move only the tasks that
hold up. Questions and authoring are the likely holdouts on a small model. Whether
define routes per task (some to Apple, the rest to Claude) is a plan decision, made
with those numbers.

`--llm-check` sends no schema, so a PONG proves the connection and says nothing
about structured output. This backend's gate is the schema tasks.

Tests: a stateful fake `fm` behind the same exec seam (ARCH-MOCK), the `llmtest`
obligation suite run against that fake, and the same suite live under
`-tags conformance` on macOS 27.

## Done when

- [ ] An `fm` backend implements `llm.Client`, is chosen by configuration, and
      needs no key.
- [ ] The schema tasks (band, author, entail, veto, learner model) get
      schema-valid JSON through `fm respond --schema`, asserted live under
      `-tags conformance` on macOS 27.
- [ ] Unavailability (not macOS 27, no `fm`, Apple Intelligence off) is
      `ErrUnavailable`, and define degrades quietly.
- [ ] The `llmtest` obligation suite passes against a stateful fake `fm`.
- [ ] The conformance results decide which tasks run on the Apple model, and the
      Log records them.
- [ ] `atlas/llm.md` and the README's model-connection section describe the
      backend and its setting.

## Plan

- [ ] Upgrade this Mac to macOS 27 (ships 2026-09-14). Record `fm --help`,
      `fm respond --help` and `fm schema --help` in the Log.
- [ ] Probe the four open questions with define's real band prompt and schema.
- [ ] Run the harvest and reflect conformance tests against on-device and `pcc`;
      record pass/fail and band agreement per task.
- [ ] Design via `sdlc start-plan` with those answers.

## Log

### 2026-09-12

Asked whether the model call can be made local with Apple's own services.

- Siri has no LLM API. App Intents lets Siri call into apps, not the reverse.
- macOS 26's Foundation Models framework is Swift-only, with a 4,096-token
  context; reaching it from Go needs a Swift helper binary. Rejected in favour of
  macOS 27's `fm`.
- macOS 27 releases 2026-09-14. `fm respond`, `fm chat` and `fm schema` come
  preinstalled; the on-device model is "rebuilt from the ground up: smarter,
  better at instruction following"; `model.contextSize` prints 8192 in session
  241's example; Private Cloud Compute has reasoning and a 32K context; the Python
  SDK is `apple_fm_sdk`.
- This Mac: arm64 (Apple silicon), macOS 26.6.2.

Sources:
- [WWDC26 241: What's new in the Foundation Models framework](https://developer.apple.com/videos/play/wwdc2026/241/)
- [WWDC26 334: Build AI-powered scripts with the fm CLI and Python SDK](https://developer.apple.com/videos/play/wwdc2026/334/)
- [WWDC26 339: Bring an LLM provider to the Foundation Models framework](https://developer.apple.com/videos/play/wwdc2026/339/)
- [Apple Intelligence guide, WWDC26](https://developer.apple.com/wwdc26/guides/apple-intelligence/)
- [9to5Mac: macOS 27 launches September 14](https://9to5mac.com/2026/09/09/apple-confirms-macos-27-golden-gate-launch-date-september-14/)
