# Boundary Review — tools#85 (whole-issue close)

| field | value |
|-------|-------|
| issue | 85 — llm: public seam for echo — multi-turn, tools, streamed tool-call deltas, cancel |
| repo | tools |
| issue file | workshop/issues/000085-llm-public-seam-for-echo-multi-turn-tools-streamed-tool-call-deltas-cancel.md |
| boundary | whole-issue close |
| milestone | — |
| window | 32e6e8d9a548edb257c2ee7a4b8e1e2d2a8e0b90..abfb3f17c90743f73d0b26a347abbddbbcbaf325 |
| command | sdlc close --issue 85 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-21T23:41:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Prior findings BR-1 and BR-3 look addressed. Now checking whether the new captures are actually replayed through the client.

Real captures parse correctly through production code; the gap is that no committed test replays them. Running the suite now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The seam is finished and works. `go build ./...`, `go vet ./...`, `go test ./llm/... ./conformance/... ./cmd/...` and `go test -race ./llm/...` all pass on the pinned head, and the `-tags conformance` build compiles. All six open prior findings are fixed. To check that the three new real captures work with the production code, I wrote a throwaway test in this checkout and deleted it afterwards. `stream-tool-use.sse` goes cleanly through `Client.Stream`: one call at Index 1, the joined `DeltaToolCallInput` fragments form valid JSON, the End input equals `ToolCalls[0].Input`, and stop is `tool_use`. `message-tool-use.json` goes through `buildResponse` and keeps the thinking `Raw` and its signature. What stands between this and SHIP: Task 6 says "captures replayed in `captures_test.go`", and Task 3 says the fuzz seeds use real frames. Neither was done, but the box is ticked. The captures are only checked by string search, so no committed test runs them through the production parser. Also, the plan and the issue disagree on whether M3 is done (a third finding in a family already in play).

**1. Strengths**
- Every prior partial-response finding is fixed at the level of the rule, not one case: `assertOnlyCompleteBlocks` now switches over every block type (`llm/seam_test.go:382-399`), and `TestPartialKeepsOnlyCompleteBlocks` (`llm/delta_test.go:277`) checks `deltaState.complete` for thinking, text and tool_use.
- `scripts/llm-probe.sh` stages, verifies, then promotes. The new tool captures get verify rules that explain how to fix a bad recording ("strengthen TOOL_PROMPT", "lengthen the input"). The temp folder is removed by `trap` (`:227`).
- The live drift row (`llm/capture_conformance_test.go:148`) checks the one fact the fake relies on: tool input arrives in several frames. `TestBeforeAttemptAgainstTheLiveService` covers the hook, which `Suite` cannot install.
- Switching `echoRequest` to `ToolChoice: "auto"` is explained in its doc comment (`llm/llmtest/suite.go:326-329`): a forced choice is refused alongside extended thinking. It is recorded in the Log too.

**2. Critical findings**
None.

**3. Important findings**
- **The new captures are never run through the production decode path** (ARCH-MOCK, ARCH-PURPOSE). The new rows in `llm/llmtest/captures_test.go:130-159` only search the capture text for strings. The older captures are all replayed through `buildResponse` (`llm/response_test.go:35`) or a stub transport (`llm/provider_render_test.go:94`). Nothing does that for `stream-tool-use.sse`, `message-tool-use.json` or `message-tool-result.json`, and `FuzzDeltaAccumulator` (`llm/delta_test.go:49-51`) still uses only made-up seeds. Plan Task 6 ("captures replayed") and Task 3 ("M3 adds the real ones") are ticked anyway. Real tool_use blocks carry fields the fake never sends, such as `"caller":{"type":"direct"}`. If an SDK bump starts failing on those, only a live run would notice.
  - Fix: in package `llm`, add a test that streams `stream-tool-use.sse` through `rendererTransport` and checks:
    - the joined fragments equal End's `Input`, and End's `Input` equals `ToolCalls[0].Input`;
    - stop is `tool_use`;
    - a Start/End pair at the same Index.
  - Add a second test: `buildResponse(loadCapture("message-tool-use.json"))` gives one `ToolCall`, and `Message()` keeps the thinking `Raw` and `Signature`.
  - Then either add the capture's input fragments as a fuzz seed, or record in Revisions that the seed was dropped.
  - I ran exactly this test and it passes today. It is about 40 lines.

**4. Minor findings**
- **This is the 3rd finding in family `tracker-checkbox-state-diverges`.** The issue now marks M3 done (`abfb3f1`), but the plan still has `- [ ] M3` (plan:376). Both M3 lines include "land; tag `v0.1.0`", which has not happened, and every "Done when" box is still unchecked, including ones already delivered.
  - The rule covering all three cases: the issue's milestone box and the plan's milestone box are one fact. Tick them in the same commit, and only when every deliverable named on that line exists.
  - Fix the class, not this case: move the post-land steps (land, tag, `go get` check) out of the M3 line in both files, into their own "after close" line or the unticked Task 6 item. Then tick M3 in both places together. Tick each "Done when" box at the moment its evidence is logged.
- Plan Task 6 still says "a tool-use capture (one tool, forced)", but what shipped uses `tool_choice: auto`. The reason is in the Log but not in `## Revisions`.
- `atlas/llm.md` "Conformance" puts the new tool-frame clause on one long line, unlike the wrapped prose around it.
- The weather tool definition and prompt are written twice, in `scripts/llm-probe.sh:83-84` and `llm/capture_conformance_test.go:149-156`. One is shell and one is Go, so it is acceptable, but add a comment saying they must change together (ARCH-DRY, note only).

**5. Test coverage notes**
- Every Suite obligation from the plan is there and passes on the fake under `-race`. The Log says the strict live run (`CONFORMANCE_STRICT=1 go test -tags conformance ./llm/... ./conformance/...`) passed apart from the preamble row, which is filed as tools#87. I could not re-run it here (no proxy). What I did confirm is that the conformance build compiles.
- The thinking row in the partial-block check checks `Signature == ""`. A thinking block that received its `signature_delta` but no `content_block_stop` would get past the check. `deltaState.complete` does still drop it, and the unit test pins that.

**6. Architectural notes**
- ARCH-DRY: pass. One `toolInputObject` serves both paths; the probe/test duplication is noted above.
- ARCH-PURE: pass. `wireMessages`, `deltaState` and `outcome` are pure and tested from tables or fuzzing.
- ARCH-PURPOSE: flagged. "Prove against reality" is half delivered: recorded, but not replayed offline (Important finding).
- ARCH-MOCK: flagged, same gap. Otherwise good: the fake sits behind the real wire, and the live Suite, drift and BeforeAttempt checks exist.
- ARCH-CONSTRAINTS: pass. `TestDeltaForLinear` pins the hot-path cost.
- ARCH-SECURE: pass for this diff. Captures hold signatures, not credentials. The probe passes the key to curl as a `-H` argument (visible in the process list), but that pattern predates this diff and the key is a local proxy key.
- ARCH-ORDER: pass. The cancel/finish race is legal on both paths, and the strict case is on the fake.
- ARCH-FUNERAL: pass. The stage folder is removed by the trap; captures are replaced each generation.
- ARCH-EVOLVE: pass. No persisted format changed; the single-turn hash is pinned by `TestSingleTurnHashUnchanged`.
- For echo's v0.2+: cache breakpoints were filed as a follow-up. `record` still overwrites every capture (the cause of tools#87), so a per-capture `record <name>` would stop one new recording from disturbing the older evidence.

**7. Plan revision recommendations**
- `## Revisions`, 2026-10-21 (M3): "The Task 6 tool capture uses `tool_choice: auto` with the call asked for in the prompt. A forced choice is refused alongside extended thinking. The Suite echo obligations do the same."
- `## Revisions`: "M3's land / tag `v0.1.0` / `go get` check happen after close. They are tracked by the unticked Task 6 item, not by the M3 milestone box." Then tick M3 in the plan in the same commit as the issue.
- If the fuzz seed from real frames is not added: "`FuzzDeltaAccumulator` keeps made-up seeds; real frames are pinned by <replay test name> instead."

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      conformance/conformance.go:47 and guard_test.go:25 keep internal/llm with a "(then internal/llm)" or "at the time" note; atlas/define.md:2772 says "(as it was then)".
  - id: BR-2
    disposition: addressed
    note: |
      Plan line 370 now reads "- [x] M1". The new M3 divergence is raised separately as the family's 3rd instance.
  - id: BR-3
    disposition: addressed
    note: |
      llm/conformance_test.go imports conformance in the github.com/xianxu/tools group after the stdlib group; capture_conformance_test.go was regrouped in b9b9dc0 too.
  - id: BR-8
    disposition: addressed
    note: |
      atlas/llm.md now documents Stream with func(Delta), TextOnly, ErrCanceled, BeforeAttempt, Usage.Attempts, the fake controls and the new live suites.
  - id: BR-9
    disposition: addressed
    note: |
      80cae37: assertOnlyCompleteBlocks checks every block type (thinking needs a signature, others need valid Raw), and TestPartialKeepsOnlyCompleteBlocks pins deltaState.complete dropping a thinking block with no stop.
  - id: BR-10
    disposition: addressed
    note: |
      suite.go:196 is renamed "a cancelled stream ends in ErrCanceled or a complete success, never a late callback".
findings:
  - id: new
    severity: Important
    kind: defect
    family: capture-not-replayed-by-production-path
    title: |
      New tool-use captures are only text-searched, never decoded by Stream or buildResponse, yet Task 6 "captures replayed" is ticked
    detail: |
      llm/llmtest/captures_test.go:130-159 only does strings.Contains on stream-tool-use.sse, message-tool-use.json and message-tool-result.json. No test runs them through Client.Stream (rendererTransport) or buildResponse, unlike the older captures (response_test.go:35, provider_render_test.go:94). FuzzDeltaAccumulator seeds (delta_test.go:49-51) are still made up although Task 3 says M3 adds real frames. A throwaway replay test in this review passed (fragments join to End Input equal to ToolCalls[0].Input, stop tool_use, thinking Raw and signature kept), so this is missing regression coverage, not a bug. Commit that test, or record in Revisions that the fuzz seed was dropped.
  - id: new
    severity: Minor
    kind: defect
    family: tracker-checkbox-state-diverges
    title: |
      Issue ticks M3 (abfb3f1) while the plan's Milestones still has "- [ ] M3"; both lines include land and tag v0.1.0, which have not happened
    detail: |
      This is the 3rd finding in this family, so fix the rule, not the instance. Rule: the issue milestone box and the plan milestone box are one fact. Tick them in the same commit, and only when every deliverable named on the line exists. Fix: move land, tag and go get out of the M3 line in both files (keep them in the unticked Task 6 item), then tick M3 in both together. Tick each Done-when box when its evidence is logged; all five are still unchecked.
  - id: new
    severity: Minor
    kind: defect
    family: plan-revision-missing-for-deviation
    title: |
      Plan Task 6 still says the tool capture is "forced", but auto ToolChoice shipped and there is no Revisions entry
    detail: |
      The probe and Suite use tool_choice auto because a forced choice is refused alongside extended thinking. The reason is in the issue Log and suite.go:326, but the plan's Revisions section does not record it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c43d40aa-b76d-412a-af61-01b654196a3a","repository":"github.com/xianxu/tools","issue_id":"000085","recipe":"milestone-review","round":4,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/bba02b266d84ebf0","reviewed_head":"abfb3f17c90743f73d0b26a347abbddbbcbaf325","slot":"tools:1","requested_at":"2026-10-22T06:35:57.753373Z"},"phase":"finished","pid":62661,"started_at":"2026-10-22T06:35:57.839577Z","finished_at":"2026-10-22T06:41:38.004877Z","session_id":"a9a175ba-738f-47a8-8a2e-a9072ce33950","outcome":"returned"}] -->

---

## Re-review — 2026-10-21T23:50:44-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 85 — llm: public seam for echo — multi-turn, tools, streamed tool-call deltas, cancel |
| repo | tools |
| issue file | workshop/issues/000085-llm-public-seam-for-echo-multi-turn-tools-streamed-tool-call-deltas-cancel.md |
| boundary | whole-issue close |
| milestone | — |
| window | 32e6e8d9a548edb257c2ee7a4b8e1e2d2a8e0b90..74ef0b692d07aac73f4ef9c2e01c6f38dc92fb38 |
| command | sdlc close --issue 85 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-21T23:50:44-07:00 |
| verdict | SHIP |

## Review

Mutation check: make the accumulator drop the first fragment of each tool input and confirm the replay test catches it.

Now the Raw-signature echo path:

```verdict
verdict: SHIP
confidence: high
```

All three open findings are fixed at head `74ef0b6`, and I'm raising nothing new. The fix for BR-11 adds two tests in `llm/capture_replay_test.go` that send the recorded tool-use captures through the real code paths, `Client.Stream` and `buildResponse`/`Message()`. I checked that both tests catch real breakage by changing the code in this checkout:
- **Stream test:** I changed the `input_json_delta` fragment text in `llm/delta.go:104`. `TestToolStreamCaptureReplaysThroughStream` failed at `capture_replay_test.go:71`.
- **Echo test:** I made `blockJSON` drop the thinking signature (`llm/wire.go:89,118`). `TestToolRoundCapturesReplayThroughBuildResponse` failed at `:104` with "Message() echo drops the thinking signature".

I restored both files afterwards (`git status` is clean). `go build ./...` and `go vet` pass, `gofmt -l` is clean, and `go test ./llm/... ./conformance/... ./cmd/define/...` passes. The tracker fixes for BR-12 and BR-13 match the code: the M3 lines in the issue and the plan are now the same fact. Land, tag and `go get` are left out of M3 and stay in the unticked last Task 6 item. The `auto` ToolChoice change now has a Revisions entry.

1. **Strengths**
   - `llm/capture_replay_test.go:32-80` checks the stream against itself from several angles. Each tool call must have one Start and one End. Its input must arrive in at least two fragments, and the fragments must join to exactly `End.Input`. `End.Input` must match `Response.ToolCalls` (compared as compacted JSON), and the text deltas must equal `Response.Text`. That is stronger than comparing against a stored copy of the output.
   - `capture_replay_test.go:90-110` follows a real multi-turn echo. It checks that the thinking signature survives `Message()`, and that a second turn built from the capture passes `checkRequest`.
   - `llm/delta_test.go:52-57` adds a seed to `FuzzDeltaAccumulator` from the recorded capture. If the capture stops producing a tool call, the test fails (`f.Fatalf`) instead of quietly skipping the seed.
   - `workshop/lessons.md` turns the two recurring fixes into general rules: replay a capture through the production path, and treat a milestone box as one fact across the issue and the plan. That is the class-level answer the escalation asked for.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The plan's Task 6 line 1 still says "captures replayed in `captures_test.go`". The replay is actually in `capture_replay_test.go`; the Revisions entry names the right tests, so a reader is not misled. Not raised as a finding.

5. **Test coverage notes**
   - `stream-tool-use.sse`, `message-tool-use.json` and `message-tool-result.json` now each go through a production decoder.
   - The signature check in `TestToolRoundCapturesReplayThroughBuildResponse` loops over thinking blocks. It would pass silently if a re-record had no thinking block. Two other checks guard against that: `llm-probe.sh verify` and the shape row in `TestCapturesExhibitTheirShapes`.
   - I did not run the live conformance tests (they need the proxy). The issue Log records the live run, with the preamble drift moved to tools#87.

6. **Architecture**
   - ARCH-DRY: pass. `captureClient` reuses `rendererTransport` and `loadCapture`; widening `loadCapture` to `testing.TB` lets the fuzz seed reuse it too.
   - ARCH-PURE: pass. The replay tests drive the pure decoder through an in-memory transport, with no network.
   - ARCH-PURPOSE: pass. Every new capture is now read by production code, which is the class BR-11 named.
   - ARCH-MOCK: pass. The fake, the recorded captures and the live Suite share the same `Transport` seam, and a live drift check exists.
   - ARCH-CONSTRAINTS: pass. Nothing in this round touches runtime behaviour.
   - ARCH-SECURE: pass. The fixtures use a dummy `APIKey: "test"`, and untrusted tool input is still checked by `toolInputObject` when the block stops.
   - ARCH-ORDER: pass. Nothing changed in this round. The partial-response and cancel-race behaviour was settled in earlier rounds (BR-4, BR-5, BR-9).
   - ARCH-FUNERAL: pass. The only new durable artifacts are three fixtures, replaced when `llm-probe.sh record` runs again.
   - ARCH-EVOLVE: pass. The persisted format change here is the thinking-block echo, and `blockJSON` sends `Raw` back unchanged, so unknown fields survive. The signature test pins that.

7. **Plan revision recommendations:** none needed. Optionally, change "captures_test.go" to "capture_replay_test.go" in Task 6 line 1.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      capture_replay_test.go replays all three captures through Stream and buildResponse; mutating delta.go:104 and wire.go blockJSON each turned a test red; the fuzz corpus is seeded from message-tool-use.json.
  - id: BR-12
    disposition: addressed
    note: |
      Issue and plan M3 lines are both ticked in 74ef0b6, with land, tag and go get moved to the unticked Task 6 item; Done-when boxes 1-4 ticked with Log evidence, box 5 (tag) left open.
  - id: BR-13
    disposition: addressed
    note: |
      Task 6 now says auto choice (see Revisions), and the 2026-10-21 (M3) Revisions entry gives the reason (forced tool_choice is refused alongside extended thinking).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7dc54412-685d-4f8b-934f-6ee3fe04bec4","repository":"github.com/xianxu/tools","issue_id":"000085","recipe":"milestone-review","round":5,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/06a1d7149d731714","reviewed_head":"74ef0b692d07aac73f4ef9c2e01c6f38dc92fb38","slot":"tools:1","requested_at":"2026-10-22T06:46:38.584002Z"},"phase":"finished","pid":70180,"started_at":"2026-10-22T06:46:38.792313Z","finished_at":"2026-10-22T06:50:43.945538Z","session_id":"b9c609d2-8906-4d24-890b-e9fe0deb3b86","outcome":"returned"}] -->
