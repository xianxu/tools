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
K="$(key)"

post() { # post <json-body>
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

# Fact 3: the streamed frame sequence, including the ping event and the
# space-padded data payloads that hand-written frames would omit.
probe_stream() {
  curl -sN --max-time 300 "$BASE/v1/messages" \
    -H 'content-type: application/json' -H "x-api-key: $K" -H 'anthropic-version: 2023-06-01' \
    -d '{"model":"'"$MODEL"'","max_tokens":128,"stream":true,"messages":[{"role":"user","content":"Say: one two three"}]}'
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

case "${1:-summary}" in
  record)
    mkdir -p "$OUT"
    probe_blocks  > "$OUT/message-thinking.json"
    probe_schema  > "$OUT/message-schema.json"
    # message-truncated.json is deliberately NOT re-recorded: it is a preserved
    # specimen of a max_tokens truncation, and re-recording would destroy the very
    # thing it exists to show.
    probe_stream  > "$OUT/stream-sample.sse"
    echo "recorded into $OUT"
    ;;
  summary|*)
    echo "model=$MODEL base=$BASE"
    echo "[blocks] non-trivial prompt:"; probe_blocks | summarize
    echo "[schema] structured output:";  probe_schema | summarize
    echo "[stream] event sequence:";     probe_stream | grep '^event:' | sed 's/^event: /  /' | uniq -c
    ;;
esac
