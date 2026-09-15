# Captures

Recorded responses from the parley-managed `cli-proxy-api`. **Do not hand-edit
these files, and do not hand-write new ones.** They are evidence; an edited
capture is evidence of nothing, and the fake is modelled on them.

Regenerate with `scripts/llm-probe.sh record`, which stages, verifies and only
then promotes — a capture that does not exhibit its required shape never lands.

| file | exists to show | required shape (enforced by `llm-probe.sh verify`) |
|---|---|---|
| `message-thinking.json` | `content[0]` is **not** the text | a `thinking` block present; `stop_reason: end_turn` |
| `message-schema.json` | `output_config` survives the proxy | payload decodes; `stop_reason: end_turn` |
| `message-truncated.json` | **a truncated answer still parses** | `stop_reason: max_tokens` **and** the payload decodes |
| `stream-sample.sse` | the streamed frame shape | `message_start`…`message_stop`, plus `thinking_delta` and `signature_delta` |

Between them the three JSON captures exhibit `[thinking, text]`, `[text]` and
`[thinking, text, thinking]` — block presence *and* order both vary, which is why
nothing may assert a sequence or index into `content`.

`message-truncated.json` is a **preserved specimen** and is never re-recorded: it
was produced by asking for 512 tokens with adaptive thinking on, so thinking ate
the budget and the answer got the remainder. It is the only honest example we have
of a response that is wrong in a way the parser cannot see.

## The rule these enforce

*A capture is evidence only for the shape its recording conditions elicit.*

Three findings in one family were the same mistake: a trivial prompt returned one
text block; a 512-token budget returned a truncated answer; a trivial stream
returned no thinking frames. Each artifact looked like evidence, was committed, and
had something modelled on it. So the conditions live in the probe script beside the
capture, and `verify` refuses a capture that does not demonstrate its own point.

## Annotated language stream

`stream-language.sse` was captured on 2026-09-15 from the production answer prompt
with Spanish selected. It contains Spanish “buenos días” and “Buenos días, ¿me da
un café?” inside English explanation, including markers split across real deltas.
The answer integration fake replays these exact bytes. Regenerate separately from
the generic transport captures, from the repository root:

```sh
CONFORMANCE_STRICT=1 DEFINE_LANGUAGE_CAPTURE="$PWD/internal/llm/llmtest/testdata/stream-language.sse" go test -tags conformance ./cmd/define -run '^TestLanguageAnnotationsAgainstLiveService$' -count=1 -v
```

Inspect the captured language ownership as well as the mechanical check: valid
syntax alone does not establish that the model identified the languages correctly.
