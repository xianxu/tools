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

---

## Re-review — 2026-08-26T16:39:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 5cbcc80c7c02e3f341c65b405e048ffe5e28dec6..5cbcc80c7c02e3f341c65b405e048ffe5e28dec6 |
| command | sdlc milestone-close --issue 21 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-26T16:39:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both round-1 Criticals are genuinely fixed and I verified each by reverting it in an overlay: reverting `tokenStillOpen` reddens all four rows of `TestHighlightWriterByteAtATimeMatchesOneCall`, and deleting `d.vocab.Load()` from `vocabularyFor` reddens all three rows of `TestEveryEntryPathHighlightsDefinitions` plus the editor-loop test. The headword withhold is real and pinned (re-styling the headword reddens `TestTheHeadwordLineIsNotHighlighted`). Suite, `go vet` and `gofmt` are clean; the tree is clean. What keeps this from SHIP is two Importants, both cheap and both worth landing before M3 wires the stream: (1) `decidedEnd`'s *other* release path still releases bytes that can grow — measured, a deck word whose first rune is multi-byte loses its match when a chunk splits that rune, the same class as BR-13 in the one branch the fix did not touch; (2) four behaviours this window added by comment are pinned by nothing — most importantly the enclosing-style resume at the example region, which is the M2 Done-when clause itself and whose deletion passes the entire suite. Two prior findings are disposed `not-addressed`: the plan's Task Files blocks still name files that do not exist, and the atlas sweep BR-18 asked for was not run, leaving a paragraph that states the opposite of what this commit built.

**1. Strengths**

- `vocabularyFor` (`cmd/define/vocab.go:156`) is the right shape for the BR-14 fix: one answer to "loaded, and only with colour", consumed by both `runEditor` and `lookupAndRender`, with `TestEveryEntryPathHighlightsDefinitions` (`vocab_test.go:242`) as a table over *process entry points* driven with a deliberately unloaded `storeVocabulary`. Verified: the mutation the fix targets kills all three rows.
- `tokenStillOpen` (`highlightwriter.go:225`) states its evidence rather than its symptom — three named ways a token can still grow, with the byte-at-a-time table (`highlightwriter_test.go:237`) as the harshest split schedule. Verified by revert.
- Moving the per-region decision into `Render` via `RenderOpts.Vocab` + `admitsHighlight` (`render.go:62`) is the correct structural answer to "a finished string has no structure left to consult", and it keeps `Render` pure over injected data (ARCH-PURE holds — `Load` stays at the boundary).
- `fuzzDeck` (`highlightwriter_test.go:192`) deriving from `TestWordRuns`' table instead of a hand-picked constant is exactly the right generalisation of the round-1 lesson, and the lessons.md entry ("a property is only as wide as its fixtures") states it well.
- `phraseGapOrEmpty` delegating to `phraseGap`, and `stripANSI` extended rather than duplicated, resolve BR-17 properly — no near-duplicate helper pairs remain in `cmd/define` (ARCH-DRY passes).

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/define/highlightwriter.go:187` — `decidedEnd`'s no-token branch releases without the growth check.** When `wordRuns(region)` is empty the function returns `len(region)` on anything that is not a pure phrase gap, never consulting `tokenStillOpen`. Measured with the shipped code: `writeChunks(vocab("über"), "!über")` → `"!\x1b[1;32müber\x1b[0m"`, but `writeChunks(vocab("über"), "!\xc3", "\xbcber")` → `"!über"`, and the same for a three-chunk `"hi!"|" "|"\xc3"|"\xbcber now"`. Bytes survive; the match does not. Fix sketch: make the empty-token branch return 0 when `tokenStillOpen(region)`, so both release paths ask the same question.
- **`cmd/define/render.go:175` — the enclosing-style resume at the one production site that has one is pinned by nothing.** Replacing `p.ex` with `""` in the example region passes the whole suite, yet it changes the bytes: correct output is `\x1b[3;32m“an \x1b[1;32mobsequious\x1b[0m\x1b[3;32m smile followed”\x1b[0m`, the mutant drops the resume and leaves the rest of the example unstyled. That is the M2 Done-when ("the enclosing style resumes after it") with no proof at the surface it applies to. Three siblings measured in the same window: `sgrState.base` dies with it; `maxOpenSGR` (`sgr.go:14`) has no test at all; and dropping `!opt.color` from `vocabularyFor` (`vocab.go:157`) — the M1-round-3 "no IO for a disabled feature" guard — passes the full suite. Fix sketch: assert the production bytes for a highlight inside a styled region, and use a spying double (`countingDeck` already exists at `vocab_test.go:199`) for the guards whose only effect is *absence* of IO.

**4. Minor findings**

- `cmd/define/highlightwriter.go:257` — `highlightText` has zero call sites in production or tests; `RenderOpts.prose` → `highlightRegion` replaced it. Both `atlas/define.md` and the plan still name it as the definition path.
- `cmd/define/render.go:47` — the admit/withhold table lists 10 regions, but `Render` produces more: `HeadHomograph` and `HeadOther` (which carries real prose, e.g. `read verb (past and past participle read | red |)`) and the block label at `render.go:130` have no row. Behaviour is correct for all of them; the enumeration that is supposed to force the decision is a subset of the regions.

**5. Test coverage notes**

- Mutation results this round: 4 killed (`tokenStillOpen` revert, `Load()` deletion, headword re-styling, plus the existing suite), 2 survived (example base-resume, `vocabularyFor` colour gate).
- `TestHighlightingLosesNothing` now runs over the colour-ON render — the substantive half of BR-20 — but still has no "at least one entry highlighted" guard. Measured at HEAD: 6 of 32 corpus entries highlight, so it is not vacuous today, and nothing would say so if a corpus refresh made it vacuous.
- `FuzzHighlightWriterIsChunkIndependent`'s deck reproduces the tokenizer's character *classes* but not their *positions*: `café` is the only multi-byte entry and its multi-byte rune is word-final. Adding one word-initial multi-byte entry makes the Important finding above reachable at low exec counts.

**6. Architectural notes for upcoming work**

- ARCH-DRY: pass. One tokenizer, one matcher, one writer; `phraseGapOrEmpty` delegates; `storeVocabulary` embeds `memVocabulary`; the round-1 duplicate pairs are gone and a sweep of `cmd/define`'s function list turns up no new ones.
- ARCH-PURE: pass. `wordRuns`/`highlightSpans`/`sgrState`/`decidedEnd`/`tokenStillOpen` are pure and unit-tested without IO; `highlightWriter` is the thin shell; `Render` takes the predicate rather than performing the load.
- ARCH-PURPOSE: pass on the milestone's substance (definitions highlight on all three entry paths, verified end to end), flagged on the two enumerations above — the region table and the fuzz deck each name a class and then write a subset of it, which is the axis this issue keeps recurring on.
- ARCH-MOCK: pass. No new external dependency; `memVocabulary` is the double at the same seam production uses, and `store.NewMem()` backs the store-side tests.
- For M3: the writer's `base` field and the flush-on-every-exit-path requirement are the two things with no test pressure today. Land the Important findings first — M3 is where the release gap stops being theoretical.

**7. Plan revision recommendations**

- `## Revisions` entry for the file-layout deviations still outstanding: Task 5 Files (`plan:268-269`) → `cmd/define/sgr.go` / `sgr_test.go`; Task 6 Files (`plan:280-281`) → `highlightwriter.go` / `highlightwriter_test.go`; Task 7 Files (`plan:298-299`) → `cmd/define/render.go` (the region table) plus `main.go:505`, tests in `highlightwriter_test.go`; Task 6 Step 6 (`plan:291`) names `FuzzHighlightWriter`, shipped as `FuzzHighlightWriterIsChunkIndependent`; Task 7 Step 2 (`plan:303`) places the invariant in `invariant_test.go`, shipped as `TestHighlightingLosesNothing` in `highlightwriter_test.go`.
- Core concepts, `plan:70`: `highlightWriter`'s "Injected into: the definition print site (`main.go:488`)" describes the design this round replaced — the injection is now `RenderOpts.Vocab` into `Render`'s admitted regions.
- `atlas/define.md:498-500` needs the same correction in the opposite direction: it claims highlighting "wraps the RENDERED string rather than reaching into `Render`", which is what 5cbcc80 removed.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Verified by revert — stubbing tokenStillOpen to false reddens all four rows of TestHighlightWriterByteAtATimeMatchesOneCall.
  - id: BR-14
    disposition: addressed
    note: |
      Verified by revert — deleting d.vocab.Load() reddens all three entry-path rows plus TestEditorLoopHighlightsADeckWordOnScreen.
  - id: BR-15
    disposition: addressed
    note: |
      Verified by mutation — routing the head token through opt.prose reddens TestTheHeadwordLineIsNotHighlighted.
  - id: BR-16
    disposition: not-addressed
    note: |
      Core-concepts rows and contract rule 4 are corrected; Task 5/6/7 Files blocks, the main.go:488 injection point, the fuzz target name and Task 7 Step 2's invariant location still name what does not exist.
  - id: BR-17
    disposition: addressed
    note: |
      stripEscapes and onlyPhraseGap are gone; phraseGapOrEmpty delegates to phraseGap; a sweep of cmd/define's function list finds no remaining near-duplicate pair.
  - id: BR-18
    disposition: not-addressed
    note: |
      The crlfWriter line was fixed but the window sweep it demanded was not run — atlas/define.md:498 now claims highlighting wraps the rendered string rather than reaching into Render, the opposite of what this commit built.
  - id: BR-19
    disposition: addressed
    note: |
      Working tree is clean; no zz_probe file in cmd/define.
  - id: BR-20
    disposition: not-addressed
    note: |
      The colour-ON half landed; the hits-greater-than-zero vacuity guard did not — measured 6 of 32 corpus entries highlight at HEAD.
  - id: BR-21
    disposition: addressed
    note: |
      highlightwriter_test.go:287 now asserts absence of any escape byte.
  - id: BR-22
    disposition: addressed
    note: |
      newHighlightWriter nils the vocabulary when on is empty, so a direct M3 caller inherits the guard.
findings:
  - id: new
    severity: Important
    family: release-only-what-cannot-change
    title: |
      decidedEnd's no-token branch still releases bytes that can grow, so a chunk splitting a word-initial multi-byte rune loses the match
    detail: |
      This is the 2nd finding in family `release-only-what-cannot-change` — do NOT fix
      only this instance. THE RULE: every release path in decidedEnd must consult the
      same "can the tail still grow?" predicate. There are two, and only one calls
      tokenStillOpen: when wordRuns(region) is empty (highlightwriter.go:187) the
      function returns len(region) for anything that is not a pure phrase gap, so a
      region holding only non-word bytes plus an incomplete rune is released.
      Measured at HEAD: writeChunks(vocab("uber-with-umlaut"), "!<word>") highlights,
      writeChunks(..., "!\xc3", "\xbcber") does not; likewise "hi!"|" "|"\xc3"|"\xbcber now".
      Bytes survive, the match does not, and the property that should quantify over
      this is blind because fuzzDeck reproduces the tokenizer's character CLASSES but
      not their POSITIONS — cafe is the only multi-byte entry and its multi-byte rune
      is word-final, so no exec count reaches a word-initial one. The enumeration to
      write is class x position (initial / medial / final) for each word-character
      class, applied to both the derived deck and the byte-at-a-time table.
  - id: new
    severity: Important
    family: behaviour-claimed-without-a-failing-test
    title: |
      Four behaviours added this window are pinned by nothing, including the M2 Done-when's own enclosing-style resume
    detail: |
      This is the 7th finding in family `behaviour-claimed-without-a-failing-test` —
      do NOT fix only this instance. Measured survivals at HEAD: (1) render.go:175,
      replacing the p.ex base with "" passes the whole suite while changing production
      bytes — correct output is \x1b[3;32m"an \x1b[1;32mobsequious\x1b[0m\x1b[3;32m
      smile followed"\x1b[0m and the mutant drops the resume, leaving the rest of the
      example unstyled; sgrState.base dies with the same mutation. (2) sgr.go:14,
      maxOpenSGR has no test at all. (3) vocab.go:157, deleting !opt.color from
      vocabularyFor passes — the whole deck is read under -no-color, which is exactly
      M1 round 3's io-for-a-disabled-feature finding, now unpinned again after the
      refactor moved it. THE RULE, in the shape this window needs: the prior
      enumerations covered hops that change OUTPUT, and every survivor here is either a
      wiring argument (base) or a guard whose only effect is the ABSENCE of work. So
      the enumeration is: for each behaviour this diff's comments claim, name the
      observation that would falsify it — output bytes at the production site for
      wiring, and a counting/spying double for guards. The package already has
      countingDeck (vocab_test.go:199) doing precisely this for "read the deck once";
      the colour gate can reuse it verbatim.
  - id: new
    severity: Minor
    family: dead-test-scaffolding
    title: |
      highlightText has zero call sites after the per-region refactor, while atlas and plan both name it as the definition path
    detail: |
      highlightwriter.go:257. RenderOpts.prose -> highlightRegion replaced it; go vet
      does not flag unused functions, so it survives silently. Same rule as the earlier
      dead-scaffolding instances, widened to production code: when a refactor replaces a
      helper, the helper goes with it, and so do the docs that name it.
  - id: new
    severity: Minor
    family: feature-leaks-across-namespace
    title: |
      admitsHighlight's region table lists 10 regions; Render produces more
    detail: |
      This is the 3rd finding in family `feature-leaks-across-namespace` — do NOT fix
      only this instance. render.go:47's table omits HeadHomograph, HeadOther (which
      carries real prose, e.g. "read verb (past and past participle read | red |)") and
      the block label at render.go:130. All three are withheld by construction today, so
      behaviour is correct; the enumeration that is supposed to force a decision is a
      subset of the regions, and nothing makes a newly added region declare itself. THE
      RULE: the table is only a decision procedure if it is complete and something
      fails when a region is missing from it — enumerate from Render's emit sites, and
      pin the withholds with one assertion that a highlight appears only inside admitted
      regions.
```

---

## Re-review — 2026-08-26T17:05:08-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 21 — highlight the words you are learning wherever they appear |
| repo | tools |
| issue file | workshop/issues/000021-highlight-learned.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 10de17d99a08858c44f5f9714f3f7c6cf3552675..10de17d99a08858c44f5f9714f3f7c6cf3552675 |
| command | sdlc milestone-close --issue 21 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-26T17:05:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Every code-level finding from round 2 is genuinely fixed, and I verified each by reverting it in a scratch copy and confirming a *named* test goes red — BR-23 (both `decidedEnd` release paths), BR-24 (the `p.ex` base, `maxOpenSGR`, the `!opt.color` gate), BR-26 (the region table, whose leak counts reproduce the Log's 59/29/11 exactly), BR-25 (`highlightText` gone, zero references in the tree). I also fuzzed the shipped property for 2.47M execs clean, ran an independent differential property (byte-at-a-time vs one-call, deck derived from the text's own 1–4-token windows) for 284k execs clean, and ran `go test -race ./cmd/define/` green; `go vet` and `gofmt` are clean. What is left is not correctness: BR-16 is still open with three concrete residues in the plan record (a third round of the same family), and BR-20's second half — the vacuity guard — never landed, so `TestHighlightingLosesNothing` still passes with an unmatchable deck. Neither blocks the boundary; both are one-line fixes and the family escalation is the reason to take them now rather than at close.

### 1. Strengths

- **`decidedEnd`'s two release paths now ask one question, and the test proves it.** `cmd/define/highlightwriter.go:187-197` — reverting to `if phraseGapOrEmpty(region)` reddens `TestHighlightWriterHoldsAWordInitialMultiByteRune` *and* `TestHighlightWriterByteAtATimeMatchesOneCall/!über_and_naïve_café`. The fix is reachable, and the fixture that reaches it (`fuzzDeck`, class × position, `highlightwriter_test.go:207`) is derived rather than hand-picked.
- **`TestHighlightsAppearOnlyInAdmittedRegions` (`highlightwriter_test.go:427`) is a real decision procedure, not a restated table.** It derives admitted text from the parsed `Entry` and normalises through `store.Key`, so it catches regions nobody remembered. I mutated three separate withheld regions; the failure counts were exactly the ones the Log claims — section name 59, block POS 29, `HeadOther` 11 — and `checked=269` across 32 entries with a `checked == 0` fatal guard.
- **The guards whose only effect is absence are now watched by counting doubles.** `TestNoColourReadsNoDeck` (`vocab_test.go:290`) reuses the package's existing `countingDeck` rather than inventing a fixture; deleting `!opt.color` from `vocabularyFor` reddens it on both assertions. This is the right shape for the class the round-2 lesson names.
- **`vocabularyFor` (`vocab.go:157`) is a genuine single answer to "loaded, and only with colour"**, and `TestEveryEntryPathHighlightsDefinitions` drives all three process entry points with a deliberately *unloaded* `storeVocabulary` — the enumeration starts where the process starts, which is the corrected rule.
- **Only one production `Render` call site** (`main.go:505`), so the per-region decision has exactly one place it can be got wrong.

### 2. Critical findings

None.

### 3. Important findings

**BR-16 residue — `workshop/plans/000021-highlight-learned-plan.md:270`, `:282`, `:286`.** The round-2 sweep corrected the `Create:` lines and the Core-concepts rows but not the `Test:` lines or the step text: Task 5 and Task 6 both still say `Test: cmd/define/highlight_test.go` while the tests live in `sgr_test.go` and `highlightwriter_test.go`; Task 6 Step 2 still instructs "assert … that the returned count is in the caller's units", which is the exact contradiction the finding named — contract rule 4 (`:100-110`) and the shipped `TestHighlightWriterPropagatesDownstreamErrors` both say `(0, err)`. (Lines 374 and 498 also say "caller-unit", but those are `## Revisions` history and should stay.) Third round of `plan-record-not-updated`, so the fix is not these three lines: the class is "every path and name the plan asserts, versus the tree at HEAD", and it is mechanically enumerable. `cmd/define/repo_guard_test.go` is in-tree precedent for a guard that shells to git and `Fatal`s rather than `Skip`s.

### 4. Minor findings

- **The region table has 15 rows; three live documents say otherwise.** `render.go:48` ("Render emits thirteen"), `atlas/define.md:498` ("thirteen regions"), `lessons.md:1138` ("the ten-region admit/withhold table"). The table itself is complete and enforced — the count is a second source of truth that nothing checks.
- **A highlight resumes `base` even after a reset that arrived inside the region**, so a deck word changes the colour of its *neighbours*. `highlightRegion(region, vocab("known"), knownOn, "\x1b[3;32m")` over `"foo \x1b[35m/aI/\x1b[0m bar known baz"` yields `… \x1b[1;32mknown\x1b[0m\x1b[3;32m baz"` — ` baz` is italic-green with highlighting on and plain without it. `sgrState.resume()` (`sgr.go:48`) clears `open` on a reset but never `base`. Escape-stripped-equal, so `TestHighlightingLosesNothing` cannot see it, and M3's model output will carry resets.
- **`fuzzDeck` has no phrase longer than two tokens**, so `decidedEnd`'s `k := len(toks) - maxWords` arithmetic is never fuzzed at `maxWords ≥ 3`. I ran that axis separately (text-derived decks up to 4-token windows, byte-at-a-time, 284k execs) and it is clean today — this is coverage, not a bug — but `in spite of` is a realistic deck entry and the axis is one `fuzzDeck` line.
- `decidedEnd` runs `highlightSpans(region, v)` and then `emitText` runs it again on the released prefix. Not hot-path; noted only so M3 doesn't inherit it as a per-delta cost on long streams.

### 5. Test coverage notes

The M2 Done-when rows are each pinned by a named test that dies to a mutation: definition body → `TestDefinitionBodyHighlightsADeckWord` (driven through `lookupAndRender`, not the helper); enclosing-style resume → `TestExampleTextResumesItsStyleAfterAHighlight` (production bytes); `-no-color` → `TestDefinitionHighlightingIsOffWithoutColour` (absence of *any* escape); no-data-loss → `TestHighlightingLosesNothing`, whose byte-drop mutation I confirmed reddens.

The one gap is that last test's vacuity: swapping its deck for `vocab("zzzznotinanycorpusentry")` leaves it green. Measured at HEAD, 6 of 32 corpus entries highlight and 4 exercise a non-empty `resume()`. So the colour-ON half of BR-20 did land (it is `Color: true, Vocab: v` since 5cbcc80, and `sgrState.resume` *is* exercised over the corpus) — only the `hits > 0` guard is missing, and a corpus refresh or a deck typo would silently empty the test.

### 6. Architectural notes

- **ARCH-DRY — pass.** One writer for both styled surfaces; `phraseGapOrEmpty` delegates to `phraseGap` instead of forking it; `stripANSI` now defers to `scanEscape` so production and tests share one escape scanner; `storeVocabulary` embeds `memVocabulary`; the leak test's `wordsOf` calls production `wordRuns`, so it cannot disagree with the tokenizer. `failAfter` next to `shortWriter` is not a copy-paste — it models a different downstream behaviour.
- **ARCH-PURE — pass.** `wordRuns`, `highlightSpans`, `sgrState`, `scanEscape`, `decidedEnd`, `tokenStillOpen` are pure and their tests need no IO. `highlightWriter`'s only IO is the injected `io.Writer`, exercised with `bytes.Buffer` and two doubles. `Render` stays pure by taking `Vocabulary` as injected data; the single IO decision (load + colour) is isolated in `vocabularyFor` and pinned by a counting double.
- **ARCH-PURPOSE — pass on the feature, flag on the finding-answering axis.** Shadow-sweep of the "one predicate" source: `RenderLine → highlightSpans → v.Has` and `Render → prose → highlightRegion → highlightWriter → highlightSpans → v.Has`; every render consumer derives, `ask.go` is M3 and is marked as future tense in both atlas and README rather than claimed. The flag is BR-16: for the third round the class was answered instance-by-instance. Answering it as a class means writing the enumeration, not sweeping harder by hand.
- **ARCH-MOCK — pass.** M2 adds no external dependency. `store.NewMem()` is the portable non-production backend; `countingDeck` wraps it rather than reaching around the seam; the erroring/short doubles sit exactly at the `io.Writer` boundary production uses. The package's live-conformance pattern (`dict_conformance_test.go`, `pty_conformance_test.go`, …) is untouched and still applies.

For M3: the writer's poisoning contract is the piece most likely to bite. `crlfWriter` short-writes with a nil error and `highlightWriter` correctly treats that as failure (`emit`, `highlightwriter.go:152`), but the composed pair means one short write kills highlighting for the rest of the answer with no visible signal. Task 8 Step 4's flush enumeration should include "what does the user see when the writer is already poisoned".

### 7. Plan revision recommendations

- **`## Revisions` — "M2 boundary round 3: the plan-record sweep, written down instead of repeated"**: correct `Test:` on Task 5 (`sgr_test.go`) and Task 6 (`highlightwriter_test.go`); delete "and that the returned count is in the caller's units" from Task 6 Step 2 and point it at contract rule 4; record that the enumeration for this family is now a guard over the plan's own path references rather than a per-round manual sweep.
- **Same entry, region count**: state that `admitsHighlight` enumerates 15 regions and that the number is not to be restated in prose — the derived test is the record. Correct `render.go:48`, `atlas/define.md:498` and `lessons.md:1138` in the same commit.

```findings
dispose:
  - id: BR-16
    disposition: not-addressed
    note: |
      Plan Task 5/6 still say Test: cmd/define/highlight_test.go, and Task 6 Step 2 still promises caller-unit counts.
  - id: BR-18
    disposition: addressed
    note: |
      atlas rule 4 now reads "crlfWriter, which M3 will nest this inside"; swept the rest of this window's atlas/README claims, all true at HEAD except the region count raised below.
  - id: BR-20
    disposition: not-addressed
    note: |
      Colour-ON half fixed (Color:true+Vocab; 4 of 32 entries exercise resume); the hits>0 guard is still absent — an unmatchable deck leaves the test green.
  - id: BR-23
    disposition: addressed
    note: |
      Verified by revert — both the targeted test and the byte-at-a-time table redden; 2.47M-exec fuzz plus an independent 284k-exec differential property clean.
  - id: BR-24
    disposition: addressed
    note: |
      All three mutations verified red — p.ex base to "", maxOpenSGR cap disabled, !opt.color deleted.
  - id: BR-25
    disposition: addressed
    note: |
      highlightText deleted; zero references anywhere in the tree, docs corrected.
  - id: BR-26
    disposition: addressed
    note: |
      Table complete and derived-test enforced; leak mutations reproduce the claimed 59/29/11 exactly.
findings:
  - id: new
    severity: Minor
    family: atlas-claims-unbuilt-surface
    title: |
      admitsHighlight enumerates 15 regions; render.go, atlas and lessons.md say thirteen or ten
    detail: |
      This is the 3rd finding in family `atlas-claims-unbuilt-surface`. Do NOT fix
      only this instance. render.go:48 says "Render emits thirteen", atlas/define.md:498
      says "thirteen regions", lessons.md:1138 says "the ten-region admit/withhold table";
      the table has 15 rows (the round-2 correction also SPLIT two rows, which the
      arithmetic missed). THE RULE: a count written in prose beside an enumeration is a
      second source of truth that nothing checks and that drifts on the next edit — the
      derived test is the record, so the number should be deleted rather than corrected.
  - id: new
    severity: Minor
    family: sgr-resume-outlives-an-inner-reset
    title: |
      A highlight resumes base after a reset that arrived inside the region, restyling neighbouring plain text
    detail: |
      sgr.go:48 clears `open` on a reset but never `base`, so a region carrying its own
      reset (what prettyPronunciations emits inside an example) resumes to the enclosing
      style for text that is unstyled without highlighting. Measured: highlightRegion of
      "foo \x1b[35m/aI/\x1b[0m bar known baz" with base \x1b[3;32m returns
      "... \x1b[1;32mknown\x1b[0m\x1b[3;32m baz", where " baz" is plain in the no-highlight
      render. Escape-stripped-equal, so TestHighlightingLosesNothing cannot see it. Decide
      once whether an inner reset also clears base, and pin it — M3's model output will
      carry resets routinely.
  - id: new
    severity: Minor
    family: fuzz-fixture-axis-missing
    title: |
      fuzzDeck has no phrase longer than two tokens, so decidedEnd's hold arithmetic is never fuzzed at maxWords >= 3
    detail: |
      highlightwriter_test.go:207. The class-x-position table added this round covers
      character classes and their positions but not phrase LENGTH, which is the input
      MaxPhraseWords feeds straight into `k := len(toks) - maxWords` and the straddle
      pull-back. Not a live bug — I ran the axis independently (decks derived from each
      text's own 1..4-token windows, byte-at-a-time vs one-call, 284k execs) and it is
      clean — but it is the same "the deck is input too" rule one notch wider, and
      `in spite of` is a realistic entry. One line in fuzzDeck plus a seed.
```
