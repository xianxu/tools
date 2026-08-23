#!/usr/bin/env bash
# Probe the local cli-proxy-api and re-record the artifacts #11's fake is
# modelled on. Committed rather than run ad hoc: a fake grounded in a trace
# nobody can reproduce is a fake grounded in nothing (ARCH-MOCK).
#
#   scripts/llm-probe.sh            # print a summary of all probes
#   scripts/llm-probe.sh record     # rewrite internal/llm/llmtest/testdata/
#
# The proxy is the PARLEY-MANAGED instance (it carries auto-healing the
# standalone install does not — operator, 2026-08-22):
#   ~/.local/share/nvim/parley/cliproxy/{bin/cli-proxy-api,config.yaml}
set -euo pipefail

CONFIG="${CLIPROXY_CONFIG:-$HOME/.local/share/nvim/parley/cliproxy/config.yaml}"
BASE="${DEFINE_LLM_BASE_URL:-http://127.0.0.1:8317}"
MODEL="${DEFINE_LLM_MODEL:-claude-opus-5}"
OUT="$(cd "$(dirname "$0")/.." && pwd)/internal/llm/llmtest/testdata"

key() { python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['api-keys'][0])" "$CONFIG"; }

# The key is read LAZILY, by the probes that need it. It used to be read at
# startup, which meant `verify` — which needs neither key nor network — died on a
# missing config file. A check that cannot run without the thing it is checking
# against is a check nobody runs.
K=""
need_key() { [ -n "$K" ] || K="$(key)"; }

post() { # post <json-body>
  need_key
  curl -s --max-time 300 "$BASE/v1/messages" \
    -H 'content-type: application/json' \
    -H "x-api-key: $K" \
    -H 'anthropic-version: 2023-06-01' \
    -d "$1"
}

# Fact 1: a NON-TRIVIAL prompt returns a thinking block BEFORE the text block.
# This is the fact the fake's envelope must model. A trivial prompt does not
# reveal it — the first probe of this proxy returned ['text'] alone and nearly
# put a single-block envelope into the fake.
probe_blocks() {
  post '{"model":"'"$MODEL"'","max_tokens":2048,"messages":[{"role":"user","content":"Which of obsequious, ephemeral, meticulous would ALSO correctly fill the blank in: \"The board produced nothing but ______ agreement — every executive praised a plan they had privately called unworkable.\" Think it through, then answer."}]}'
}

# Fact 2: output_config.format (structured outputs) survives the proxy.
#
# max_tokens is 8192, NOT 512. At 512 with adaptive thinking on, thinking ate the
# whole budget: the answer returned stop_reason=max_tokens carrying
# {"verdict":"yes", "reason":": <cut mid-rune>"} -- syntactically valid, every
# required field present, semantically destroyed. It DECODED, so a parser-only
# check called it a success. That capture is preserved as message-truncated.json
# because it is the only honest specimen of the failure mode we have; this probe
# records the healthy case beside it.
#
# The general rule, and it is why the default MaxTokens is 8192: with adaptive
# thinking on, max_tokens must cover the thinking AND the answer. Budget it for
# the answer alone and the answer is what gets cut.
probe_schema() {
  post '{"model":"'"$MODEL"'","max_tokens":8192,"output_config":{"format":{"type":"json_schema","schema":{"type":"object","properties":{"verdict":{"type":"string","enum":["yes","no"]},"reason":{"type":"string"}},"required":["verdict","reason"],"additionalProperties":false}}},"messages":[{"role":"user","content":"Is obsequious a near-synonym of sycophantic?"}]}'
}

# Fact 3: the streamed frame sequence — including the ping event, the space-padded
# data payloads, AND the thinking_delta / signature_delta frames.
#
# The prompt is deliberately non-trivial and the budget deliberately large. An
# earlier version streamed "Say: one two three" at max_tokens 128, which elicits a
# text-only stream — so a Stream implementation that dropped thinking blocks, or
# fed thinking deltas to onDelta as if they were answer text, would have passed the
# entire fake-backed suite while breaking the byte-for-byte thinking echo that #16
# depends on. A capture is evidence only for the shape its recording conditions
# elicit, which is the rule the whole verify step below exists to enforce.
probe_stream() {
  need_key
  curl -sN --max-time 300 "$BASE/v1/messages" \
    -H 'content-type: application/json' -H "x-api-key: $K" -H 'anthropic-version: 2023-06-01' \
    -d '{"model":"'"$MODEL"'","max_tokens":8192,"stream":true,"thinking":{"type":"adaptive","display":"summarized"},"messages":[{"role":"user","content":"Which of obsequious, ephemeral, meticulous would ALSO correctly fill the blank in: \"The board produced nothing but ______ agreement — every executive praised a plan they had privately called unworkable.\" Think it through, then answer."}]}'
}

summarize() {
  python3 -c '
import json,sys
d=json.load(sys.stdin)
u=d.get("usage",{})
print("  blocks      :",[c.get("type") for c in d.get("content",[])])
print("  stop        :",d.get("stop_reason"))
print("  thinking tok:",u.get("output_tokens_details",{}).get("thinking_tokens"))
print("  preamble tok:",u.get("cache_creation_input_tokens") or u.get("cache_read_input_tokens"))
'
}

# verify: every capture must EXHIBIT the shape it was recorded to demonstrate.
#
# This is the fix for a family of three findings (PQ-3, PQ-9, PQ-10), each the same
# mistake: a probe recorded under conditions that do not elicit the shape the fake
# is supposed to model, producing an artifact that looks like evidence and is not.
# A trivial prompt returned one text block; a 512-token budget returned a truncated
# answer; a trivial stream returned no thinking frames. In every case the capture
# was committed and something downstream was then modelled on it.
#
# So the required shape is declared HERE, beside the conditions that produce it,
# and `record` refuses to finish if a capture does not exhibit it.
verify() {
  python3 - "${OUT}" <<'PYEOF'
import json,sys,pathlib
out=pathlib.Path(sys.argv[1]); bad=[]
def need(cond,cap,why):
    if not cond: bad.append(f"{cap}: {why}")

def blocks(name):
    return [c.get("type") for c in json.loads((out/name).read_text())["content"]]

def body(name):
    b=json.loads((out/name).read_text())
    if b.get("type")=="error":
        raise SystemExit(f"CAPTURE VERIFICATION FAILED:\n  - {name}: upstream returned an error envelope "
                         f"({b.get('error',{}).get('type')}) — this is a transient failure, not a capture")
    return b

# message-thinking.json — a non-trivial prompt, so thinking is exercised.
b=body("message-thinking.json")
need("thinking" in blocks("message-thinking.json"), "message-thinking.json",
     "no thinking block — the prompt was too easy; a capture without one cannot show that content[0] is not the text")
need(b["stop_reason"]=="end_turn", "message-thinking.json", f"stop_reason {b['stop_reason']!r}, want a healthy end_turn")

# message-schema.json — structured output that COMPLETED.
b=body("message-schema.json")
txt="".join(c.get("text","") for c in b["content"] if c.get("type")=="text")
need(b["stop_reason"]=="end_turn", "message-schema.json", f"stop_reason {b['stop_reason']!r} — raise max_tokens; thinking shares the budget")
try: json.loads(txt); ok=True
except Exception: ok=False
need(ok, "message-schema.json", "payload does not decode — this capture exists to show output_config works")

# message-truncated.json — the PRESERVED specimen. Never re-recorded.
b=body("message-truncated.json")
need(b["stop_reason"]=="max_tokens", "message-truncated.json", "no longer a truncation — this specimen must not be re-recorded")
txt="".join(c.get("text","") for c in b["content"] if c.get("type")=="text")
try: json.loads(txt); ok=True
except Exception: ok=False
need(ok, "message-truncated.json", "payload no longer PARSES — the whole point is that a truncated answer decodes cleanly")

# stream-sample.sse — must carry thinking and signature deltas, not just text.
sse=(out/"stream-sample.sse").read_text()
for frag,why in [("message_start","no message_start"),
                 ("content_block_delta","no content_block_delta"),
                 ("message_stop","no message_stop"),
                 ("thinking_delta","no thinking_delta — the stream prompt was too easy; #16's thinking echo is untestable against this"),
                 ("signature_delta","no signature_delta — a thinking signature must round-trip byte for byte")]:
    need(frag in sse, "stream-sample.sse", why)

if bad:
    print("CAPTURE VERIFICATION FAILED:"); [print("  -",b) for b in bad]
    print("\nA capture is evidence only for the shape its recording conditions elicit.")
    print("Fix the probe's prompt/budget above, not this check.")
    sys.exit(1)
print("all captures exhibit their required shapes")
PYEOF
}

case "${1:-summary}" in
  verify) verify ;;
  record)
    # Record into a STAGING dir and promote only if every capture verifies.
    #
    # The first version wrote straight into $OUT and verified afterwards, which is
    # how a good capture got clobbered by an upstream {"type":"error",
    # "error":{"type":"overloaded_error"}} — the check then correctly failed, but
    # the artifact it was protecting was already gone. A verification that runs
    # after the destructive step protects nothing.
    mkdir -p "$OUT"
    # ${TMPDIR:-/tmp} rather than mktemp -d: a sandboxed shell may refuse mkdtemp
    # outright, and this script has to run under one.
    STAGE="${TMPDIR:-/tmp}/llm-probe-stage.$$"
    mkdir -p "$STAGE"; trap 'rm -rf "$STAGE"' EXIT
    # message-truncated.json is deliberately NOT re-recorded: it is a preserved
    # specimen of a max_tokens truncation, and re-recording would destroy the very
    # thing it exists to show. It is copied across so verify sees the full set.
    cp "$OUT/message-truncated.json" "$STAGE/" 2>/dev/null || true

    for attempt in 1 2 3; do
      probe_blocks > "$STAGE/message-thinking.json"
      probe_schema > "$STAGE/message-schema.json"
      probe_stream > "$STAGE/stream-sample.sse"
      if OUT="$STAGE" verify; then
        cp "$STAGE/message-thinking.json" "$STAGE/message-schema.json" "$STAGE/stream-sample.sse" "$OUT/"
        echo "recorded into $OUT (attempt $attempt)"
        exit 0
      fi
      echo "attempt $attempt did not verify; retrying" >&2
      sleep 5
    done
    echo "recording failed to produce verifiable captures; $OUT is UNCHANGED" >&2
    exit 1
    ;;
  summary|*)
    echo "model=$MODEL base=$BASE"
    echo "[blocks] non-trivial prompt:"; probe_blocks | summarize
    echo "[schema] structured output:";  probe_schema | summarize
    echo "[stream] event sequence:";     probe_stream | grep '^event:' | sed 's/^event: /  /' | uniq -c
    ;;
esac
