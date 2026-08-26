# Boundary Review — tools#21 (milestone M2)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 56018cc1d92aefef561fda239ab6fc2cde4813e9..f23ed0200b7c90af61c8d650de337b5ea01e8dd6 |
| command | sdlc milestone-close --issue 21 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-26T15:52:11-07:00 |
| verdict | unknown |

## Review

Failed to authenticate. API Error: 401 OAuth access token has been revoked.

---

## Re-review — 2026-08-26T16:09:13-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 56018cc1d92aefef561fda239ab6fc2cde4813e9..f23ed0200b7c90af61c8d650de337b5ea01e8dd6 |
| command | sdlc milestone-close --issue 21 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-26T16:09:13-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The writer itself is the best-designed thing in this issue so far — the contract is written down normatively, the byte-exact table actually distinguishes the rules it names, and rule 3 (a known span straddling the hold point drags it back) is a genuine discovery that the tests found rather than the reasoning. But two shipped defects block the boundary. First, `decidedEnd` declares a token closed on the *trimmed* token end, so a chunk boundary landing on a joiner or mid-rune (`don'` + `t`, `hot-` + `dog`, `caf\xc3` + `\xa9`) permanently loses the match — chunk-independence, which is the writer's entire reason to exist and M3's foundation, is violated, and `FuzzHighlightWriterIsChunkIndependent` is structurally unable to see it because its vocabulary is a constant containing no joiner or multi-byte entry. Second, `Vocabulary.Load()` is called only in `runEditor`, but M2 wired highlighting into `lookupAndRender` — so on the one-shot `define <word>` path and the piped-stdin path the set is empty and no definition ever highlights. Both are verified below by running code, not by reading it.

## 1. Strengths

- **The contract is normative and the tests are byte-exact.** `cmd/define/highlightwriter_test.go:34` and the rule-1 case at `:81` (`\x1b[1;36mhot\x1b[0m dog` unchanged) genuinely distinguish a reorder from a correct result — escape-stripped comparison could not, and the plan said so before the code existed.
- **Rule 3 is real and is pinned.** Reverting the straddle-drag in `decidedEnd` (`cmd/define/highlightwriter.go:186-196`) reddens *"a phrase spanning a space is highlighted whole"* and *"a phrase split mid-word across writes"*. Verified by hand-mutation.
- **`sgrState` is genuinely pure and the table is one row per rule**, not one per escape code (`cmd/define/sgr_test.go:5-38`). The accumulate-until-reset decision is right and is the one a naive `open = seq` would get wrong; `"SGRs accumulate until a reset"` kills that mutant.
- **`TestDefinitionBodyHighlightsADeckWord` drives `lookupAndRender`, not `highlightText`** — that is M1's lesson correctly applied one level down, and it is why the *remaining* gap (finding 2) is about entry paths rather than about the chain below `d.vocab`.
- **The invariant does hold over the coloured render.** I ran the no-data-loss check over all 32 corpus entries with `RenderOpts{Color: true}` and a 7-word deck: every entry highlights, none loses a visible byte. The resume logic is correct; it is only *unasserted* (finding 8).
- `gofmt -l`, `go vet ./...`, `go test ./...` all clean.

## 2. Critical findings

### C1 — `decidedEnd` releases bytes that can still change (`cmd/define/highlightwriter.go:172`)

```go
if !onlyPhraseGap(region[toks[len(toks)-1].end:]) {
    return len(region) // punctuation closed the last token
}
```

`toks[...].end` is the **trimmed** end (`appendTrimmed`, `highlight.go:66`), so a trailing `'` or `-` is not a phrase gap and reads as "punctuation closed the token". The same is true of an incomplete UTF-8 sequence, which `range` yields as `RuneError`. Measured, one-call vs split:

| deck word | chunks | one call | split |
|---|---|---|---|
| `don't` | `don'` + `t` | `\x1b[1;32mdon't\x1b[0m` | `don't` (lost) |
| `hot-dog` | `hot-` + `dog` | highlighted | lost |
| `café` | `caf\xc3` + `\xa9` | highlighted | lost |

Also reproduces from the tokenizer's own table: `"'obsequious' don't café a priori"` split at byte 17, and byte-at-a-time.

**Fix sketch:** the writer may release text only on evidence the last token *cannot grow*. Scan back from the end of `region` over word runes, joiners, and an incomplete final rune; if that run reaches the end of the region, the token is still open — fall through to the `hold` computation instead of returning `len(region)`. `utf8.DecodeLastRuneInString` returning `RuneError, 1` is the incomplete-rune test.

**Why it escaped:** `FuzzHighlightWriterIsChunkIndependent` (`highlightwriter_test.go:181`) pins `v := vocab("obsequious", "hot dog", "hot")` as a constant. The vocabulary is part of the input space this property quantifies over, and with no joiner-bearing or multi-byte entry in it the failing class is unreachable at any exec count — 803k execs prove nothing about it. `highlight_test.go:23-38` (`TestWordRuns`) already *is* the enumeration of word-character classes (`don't`, `hot-dog`, `covid-19`, `café`, `'obsequious'`); the writer's fuzz corpus should be derived from that table rather than hand-picked, and the deck should be fuzzed alongside the text.

### C2 — definitions never highlight outside the raw editor (`cmd/define/replraw.go:84`)

`voc.Load()` has exactly one call site, inside `runEditor`. M2 wired highlighting into `lookupAndRender` (`main.go:511`), which three entry paths reach:

| entry path | `Load()` called? | highlights? |
|---|---|---|
| one-shot `define <word>` → `defineOnce` | no | **no** |
| `replLines` (piped stdin, terminal stdout, `repl.go:222`) | no | **no** |
| `replRaw` → `runEditor` | yes | yes |

Verified: an unloaded `storeVocabulary` holding `obsequious`, driven through `lookupAndRender` for `sycophantic`, produces no highlight; calling `Load()` first makes the same call produce one. Two of three paths are dead, including the one the README's new sentence describes ("Looking up `sycophantic` when you already know `obsequious` shows you the connection in the gloss itself" — false for `define sycophantic`).

> **This is the 4th finding in family `behaviour-claimed-without-a-failing-test`.** Earlier rounds fixed instances. Do not fix this instance alone — state the rule that covers all of them, and fix that.

**The rule:** *the production-chain enumeration must begin at the process entry points, not at the dependency the test injects.* The `## Log` entry enumerates "lookupAndRender reads `d.vocab` → `highlightText` → writer → stdout" — it starts one hop *after* the thing that fills `d.vocab`, and injecting a pre-populated `memVocabulary` in the test makes that hop invisible exactly the way `withStore`'s vocab merge was invisible in M1 round 2. The enumeration this rule demands is the table above: one row per entry path that can reach the render, each asserted through production wiring. Note any fix must preserve M1 round 3's `opt.color` gate (`replraw.go:82-85`) — loading the deck under `-no-color` is IO for a disabled feature.

## 3. Important findings

### I3 — the headword line is re-styled, which the Spec excludes (`cmd/define/main.go:511`)

Spec, *Deliberately out of scope*: "**Re-styling the headword line.** The head is already bold cyan; highlighting is for body text, answers, and input." `highlightText` is applied to the whole rendered string, so a deck word looked up again renders its own headword green inside the cyan. Measured for `sycophantic` with `sycophantic` in the deck:

```
"\x1b[1;36m\x1b[1;32msycophantic\x1b[0m\x1b[1;36m\x1b[0m  \x1b[2msyc·o·phan·tic\x1b[0m"
```

This is the common case for a learning tool — you revisit words. Nothing tests it either way.

> **This is the 2nd finding in family `feature-leaks-across-namespace`.** Do not fix this instance alone — state the rule.

**The rule:** *the vocabulary is withheld per region by an explicit decision at the boundary, not by wrapping whatever string is at hand.* `highlightSetFor` (`highlight.go:174`) already is that decision for the command namespace; the render surface needs the same shape. The enumeration to write and sweep: for every region a renderer produces (headword line, syllabification, pronunciation, part-of-speech label, sense body, examples, and for M3 the answer stream and the `## The word on screen` block), state admit or withhold, and pin the withholds. `-raw` is already correctly withheld at `main.go:494`.

### I4 — `Write` returns `(0, err)`, which the plan's contract and the sibling writer both contradict (`cmd/define/highlightwriter.go:57`)

Plan contract rule 4: *"On a downstream error it returns `(n, err)` where `n` is the number of `p`'s bytes whose output was fully written"*, and Task 6 Step 2 (ticked) says to *"assert … that the returned count is in the caller's units."* The code returns `0`, and `TestHighlightWriterPropagatesDownstreamErrors:150` asserts `n == 0` — it pins the opposite of the plan. In the same package, `crlfWriter` returns caller-unit progress and `crlf_test.go:47` exists specifically to defend that ("Returning 0 on a partial write claims nothing was consumed").

Returning `0` is defensible *because* the writer poisons itself, so no retry can double-emit — but that reasoning is nowhere and the two writers in one package now answer the same question opposite ways. Decide once, and either fix the code or revise the plan.

> **This is the 2nd finding in family `plan-record-not-updated`.** Do not fix this instance alone — state the rule.

**The rule:** *when the implementation deviates from the plan's stated layout or contract, the `## Revisions` entry lands in the same commit as the deviation.* The enumeration to sweep against the tree, all currently wrong:

| plan says | code is |
|---|---|
| Core concepts: `sgrState` in `cmd/define/highlight.go` | `cmd/define/sgr.go` |
| Core concepts: `highlightWriter` in `cmd/define/highlight.go` | `cmd/define/highlightwriter.go` |
| Task 5/6 Files: modify `highlight.go`, test `highlight_test.go` | new `sgr.go`/`sgr_test.go`, `highlightwriter.go`/`highlightwriter_test.go` |
| Contract rule 4: `(n, err)` in caller units | `(0, err)` |
| Task 6 Step 6: `FuzzHighlightWriter` | `FuzzHighlightWriterIsChunkIndependent` |
| Task 7 Step 2: add a case to `invariant_test.go` | new test in `highlightwriter_test.go`, different property (see I8) |

The file split is *better* than the plan; the plan is what should move.

### I5 — two more near-identical helper pairs (ARCH-DRY)

- `stripEscapes` (`highlightwriter_test.go:207`) vs `stripANSI` (`render_test.go:89`) — same package, same job; the new one is strictly better (it handles a CSI with a non-`m` final byte, which `stripANSI` runs past).
- `onlyPhraseGap` (`highlightwriter.go:200`) vs `phraseGap` (`highlight.go:100`) — identical bodies, differing only on the empty string. `onlyPhraseGap(s)` is `s == "" || phraseGap(s)`. If anyone ever admits another gap character, changing one and not the other makes the writer resolve a hold the matcher would have joined — a silently dropped phrase.

> **This is the 2nd finding in family `copy-pasted-helper`.** Do not fix these instances alone — state the rule.

**The rule:** *before adding a helper to `package main`, grep the package for one with the same job; if one exists, extend it rather than writing a second.* M1 round 3 already applied this once (`warnTo`), and two fresh pairs landed in the very next milestone, so the rule needs a mechanical enumeration, not a resolution: sweep `cmd/define` (production *and* test files — both instances here cross that line) for functions with the same shape and consolidate.

## 4. Minor findings

- `atlas/define.md:490` — "A short write with a nil error is a failure — `crlfWriter`, **which this wraps in raw mode**". It does not; that wiring is M3 Task 8 Step 5, unchecked. **This is the 2nd finding in family `atlas-claims-unbuilt-surface`** — the rule ("doc prose at a boundary describes only what that milestone shipped") was recorded in `lessons.md` after M1 round 2 and a fresh instance landed anyway; the sweep to run is every present-tense architectural claim added to `atlas/`+`README` in the window against the code at HEAD.
- `cmd/define/zz_probe_test.go` — an untracked probe file (`TestProbeOneShotDefinitionHighlight` etc.) sitting in the working tree; outside the reviewed commit but a `git add -A` at close will commit it. **This is the 2nd finding in family `dead-test-scaffolding`** — the rule: a scratch probe is deleted in the turn that reads its output, never left for a later `git add` to decide.
- `highlightwriter_test.go:263` — `TestDefinitionHighlightingIsOffWithoutColour` asserts absence of `"\x1b[1;32m"` specifically. The plan's own M1 Task 4 Step 2 says assert absence of `"\x1b"` entirely, "a test that only checks for green passes while emitting bold". Here `opt.color=false` makes the stronger assertion available for free.
- `newHighlightWriter` accepts `on == ""` and then emits a bare `sgrOff` around every known span (`highlightwriter.go:127`). `highlightText` guards it; M3 will construct the writer directly and will not get that guard for free.
- `highlightwriter_test.go:274-275` — `Render(ParseEntry(raw), RenderOpts{Color: false})` is computed twice; the second call is the same value as `plain`.
- `sgrState.open` grows without bound until a reset arrives (`sgr.go:16`). Harmless for `Render`, which resets; worth remembering when M3 puts arbitrary model output through it.

## 5. Test coverage notes

- **I8 — the no-data-loss invariant runs only over the colour-OFF render.** `TestHighlightingLosesNothing:271` renders with `RenderOpts{Color: false}`, so `w.sgr.resume()` — the ANSI-nesting logic that is the entire reason definitions and answers share a mechanism — is never exercised over the real corpus. Production always feeds `Color: opt.color`. I ran the coloured version and it passes (32/32 entries, all highlighting, no visible-text change), so this is a missing assertion rather than a live bug: change `Color: false` to `Color: true` and it covers the case the atlas claims it covers.
- **The same test can go inert silently.** It guards `len(d.entries) != 0` but not "at least one highlight occurred". Measured today: 6 of 32 entries actually highlight with `vocab("obsequious", "sycophantic", "a priori", "bank")`. A corpus refresh that drops those words leaves the test green and vacuous — the exact failure mode this plan's lessons keep naming. Add a `hits > 0` guard.
- The short-write test (`:161`) reuses `crlf_test.go`'s `shortWriter` as the plan asked; the error-injection test adds a new `failAfter`, which is fine — `crlf_test.go` has no error-returning fixture to reuse.
- No test covers `Flush` on an empty writer, a `Write` after `Flush`, or a second `Flush`. All currently behave sanely; M3's "flush on every exit path" will call `Flush` from paths that may have already flushed.

## 6. Architectural notes

- **ARCH-DRY — flag.** Three duplicate pairs now (`warnf` consolidated in M1; `stripEscapes`/`stripANSI` and `onlyPhraseGap`/`phraseGap` added in M2). See I5. The one-mechanism-for-two-surfaces decision itself is a DRY *win* and should be affirmed — it is why C1 is one bug rather than two.
- **ARCH-PURE — pass, with a note.** `sgrState`, `scanEscape`, `decidedEnd`, `highlightSpans` are pure and tested with no IO; `emit` is the single byte-touching site and knows the one thing it has to (short write ≠ success). The one blur is that `drain` interleaves the decision with the writes; if M3's flush-path work gets hairy, the extractable pure step is `(pend, sgr, final) → (output, remaining, sgr')`. Not worth doing pre-emptively.
- **ARCH-PURPOSE — flag.** The shadow-sweep over "definitions highlight" finds two of three consumer paths not deriving from the source (C2), and one surface the Spec excluded being highlighted anyway (I3). The purpose of M2 is "a deck word appearing in a definition body renders bold green" — on the most common invocation of this binary, it does not.
- **ARCH-MOCK — pass.** No new external dependency in M2. `memVocabulary` remains the stateful double behind the same `Vocabulary` boundary production uses, and `TestDefinitionBodyHighlightsADeckWord` runs the real chain against it. M3 will need the same discipline for the `llmtest.Fake` stream — the plan's Task 8 Step 1 already refuses to script the answer, which is the right call.
- **For M3:** fix C1 before wiring the stream. SSE deltas split on arbitrary byte boundaries, so mid-rune splits are not hypothetical, and `crlfWriter` nesting (Step 5) will make a byte-loss bug much harder to localise once it is two writers deep.

## 7. Plan revision recommendations

Append a `## Revisions` section to `workshop/plans/000021-highlight-learned-plan.md` with:

1. **File layout.** Core concepts table and Task 5/6 `Files:` blocks say `cmd/define/highlight.go` / `highlight_test.go`; the code lives in `sgr.go`, `sgr_test.go`, `highlightwriter.go`, `highlightwriter_test.go`. The split is deliberate and better — record it so the table stops claiming otherwise.
2. **Contract rule 4.** Either restate it as "`Write` returns `(0, err)` on a downstream failure; the writer is poisoned, so no retry is possible and caller-unit progress would be meaningless" (and say why this differs from `crlfWriter`), or change the code to match the current text. Do not leave the plan and `TestHighlightWriterPropagatesDownstreamErrors` asserting opposite things.
3. **A new contract rule (5, renumbering `Flush` to 6): release only what cannot change.** State that a trailing joiner or an incomplete UTF-8 sequence is not evidence a token has closed, with the `don'`/`t` and `caf\xc3`/`\xa9` cases as the normative examples — this is the rule C1 violates and it belongs beside rule 3, which it is a sibling of.
4. **Task 6 Step 6.** Rewrite to require the fuzz corpus and the deck to be derived from `highlight_test.go`'s `TestWordRuns` table, and state explicitly that a constant vocabulary makes the property unfalsifiable for any word class the constant omits.
5. **Task 7 Step 2.** The invariant landed as a different property in a different file than the step describes, and it runs colour-off. Record the property actually delivered and add colour-on.
6. **A new Task 7 step: wire `Vocabulary.Load()` for every entry path** (one-shot, `replLines`, `runEditor`), with the `opt.color` gate preserved, and one test per path. Today the plan has no step that would have caught C2.
7. **Spec/Task 7:** decide the headword line. Either add the exclusion as an implementation step (highlight the body only) or amend the Spec's out-of-scope list to say the head does highlight after all — but it cannot stay excluded in prose and highlighted in code.

```findings
findings:
  - id: new
    severity: Critical
    family: release-only-what-cannot-change
    title: |
      highlightWriter loses a match when a chunk splits on a joiner or mid-rune
    detail: |
      decidedEnd (cmd/define/highlightwriter.go:172) tests the bytes after the
      TRIMMED token end, so a trailing apostrophe/hyphen or an incomplete UTF-8
      sequence reads as "punctuation closed the token" and the region is
      released. Measured: don' + t, hot- + dog, caf\xc3 + \xa9 each highlight in
      one call and are lost when split, violating the writer's chunk-independence
      contract. FuzzHighlightWriterIsChunkIndependent cannot see the class — its
      vocabulary is the constant vocab("obsequious","hot dog","hot"), which holds
      no joiner-bearing or multi-byte entry, so no exec count reaches it. Fix the
      release test to require evidence the last token cannot grow, and derive the
      fuzz deck and seeds from TestWordRuns' own class table.
  - id: new
    severity: Critical
    family: behaviour-claimed-without-a-failing-test
    title: |
      Definitions never highlight on the one-shot or piped-stdin paths
    detail: |
      4th in this family — do not fix only this instance. Vocabulary.Load() has
      one call site, runEditor (cmd/define/replraw.go:84), but M2 wired
      highlighting into lookupAndRender (main.go:511), which is also reached by
      one-shot `define <word>` and by replLines. Verified: an unloaded
      storeVocabulary holding "obsequious" produces no highlight for
      "sycophantic"; calling Load() first produces one. 2 of 3 entry paths dead,
      including the one README's new sentence describes. THE RULE: the
      production-chain enumeration begins at the process entry points, not at the
      dependency the test injects — TestDefinitionBodyHighlightsADeckWord injects
      a pre-populated memVocabulary and so cannot see the Load hop, the same
      blindness as M1 round 2's withStore merge. The enumeration to write and
      sweep is one row per entry path reaching the render. Any fix must keep M1
      round 3's opt.color gate on Load.
  - id: new
    severity: Important
    family: feature-leaks-across-namespace
    title: |
      The headword line is re-styled, which the Spec lists as out of scope
    detail: |
      2nd in this family — do not fix only this instance. highlightText is
      applied to the whole rendered string (main.go:511), so a deck word looked
      up again renders its own headword green inside the bold cyan:
      "\x1b[1;36m\x1b[1;32msycophantic\x1b[0m\x1b[1;36m\x1b[0m". The Spec's
      out-of-scope list says "Re-styling the headword line". Nothing tests it
      either way. THE RULE: the vocabulary is withheld per region by an explicit
      decision at the boundary — the shape highlightSetFor already has for the
      command namespace — not by wrapping whatever string is at hand. Enumerate
      every region a renderer produces (headword, syllabification, pronunciation,
      part-of-speech, sense body, examples, and for M3 the answer stream), state
      admit or withhold for each, and pin the withholds.
  - id: new
    severity: Important
    family: plan-record-not-updated
    title: |
      Plan file layout, contract rule 4 and two test names no longer match the code
    detail: |
      2nd in this family — do not fix only this instance. Six divergences:
      sgrState and highlightWriter are in sgr.go/highlightwriter.go not
      highlight.go; Task 5/6 Files blocks name the wrong files; contract rule 4
      promises (n, err) in caller units while Write returns (0, err) and the test
      asserts n==0 — the opposite — and crlfWriter in the same package returns
      caller-unit progress with a test defending exactly that; the fuzz target
      was renamed; Task 7 Step 2's invariant landed as a different property in a
      different file. THE RULE: a deviation from the plan's stated layout or
      contract lands its "## Revisions" entry in the same commit as the
      deviation. Decide the Write count question once, and record it.
  - id: new
    severity: Important
    family: copy-pasted-helper
    title: |
      Two more near-identical helper pairs land in the same package
    detail: |
      2nd in this family — do not fix only these instances. stripEscapes
      (highlightwriter_test.go:207) duplicates stripANSI (render_test.go:89) in
      the same package; onlyPhraseGap (highlightwriter.go:200) duplicates
      phraseGap (highlight.go:100), differing only on the empty string, so
      admitting another gap character in one and not the other silently drops a
      phrase. THE RULE: before adding a helper to package main, grep the package
      for one with the same job and extend it. M1 round 3 already applied this
      once (warnTo) and two fresh pairs landed in the next milestone, so run the
      mechanical sweep across cmd/define — production and test files both, since
      one of these pairs crosses that line.
  - id: new
    severity: Minor
    family: atlas-claims-unbuilt-surface
    title: |
      atlas says highlightWriter wraps crlfWriter in raw mode; that is M3
    detail: |
      2nd in this family. atlas/define.md:490 states the crlfWriter wrapping in
      the present tense; it is Task 8 Step 5 and unchecked. The rule was recorded
      in lessons.md after M1 round 2 and a fresh instance landed anyway — sweep
      every present-tense architectural claim added to atlas/ and README in this
      window against the code at HEAD.
  - id: new
    severity: Minor
    family: dead-test-scaffolding
    title: |
      cmd/define/zz_probe_test.go left untracked in the working tree
    detail: |
      2nd in this family. Outside the reviewed commit, but a `git add -A` at
      close will commit it. THE RULE: a scratch probe is deleted in the turn that
      reads its output, never left for a later add to decide.
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      The no-data-loss invariant runs only over the colour-OFF render
    detail: |
      TestHighlightingLosesNothing renders with Color:false, so sgrState.resume —
      the ANSI-nesting logic that is the whole reason definitions and answers
      share a mechanism — is never exercised over the real corpus, though
      production always feeds Color:opt.color. Verified the coloured version
      passes (32/32 entries), so this is a missing assertion, not a live bug. The
      same test also lacks a "hits > 0" guard; measured, only 6 of 32 corpus
      entries currently highlight, so a corpus refresh could leave it green and
      vacuous.
  - id: new
    severity: Minor
    family: behaviour-claimed-without-a-failing-test
    title: |
      The no-colour test asserts absence of green, not absence of escapes
    detail: |
      highlightwriter_test.go:263 checks for "\x1b[1;32m" only. The plan's own M1
      Task 4 Step 2 rule says assert absence of "\x1b" entirely — "a test that
      only checks for green passes while emitting bold" — and with opt.color
      false the stronger assertion is free here.
  - id: new
    severity: Minor
    family: one-absence-representation-per-seam
    title: |
      newHighlightWriter accepts on == "" and emits a bare reset around each match
    detail: |
      highlightText guards v == nil || on == "" before constructing the writer,
      but the constructor does not, so a direct caller with an empty style emits
      sgrOff around every known span (highlightwriter.go:127). M3 wires the
      stream by constructing the writer directly and will not inherit that guard.
```
