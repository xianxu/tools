# Boundary Review — tools#1 (milestone M1)

| field | value |
|-------|-------|
| issue | 1 — define — NOAD word lookup with Google-style IPA and spoken pronunciation |
| repo | tools |
| issue file | workshop/issues/000001-define.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e8386b80feb8986ae8a2deb3e85d02012c0ee4a^..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-20T10:30:22-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

M1's architecture is genuinely good — the `Dictionary` seam is clean, `Render` is pure with the TTY probe correctly pushed to `main.go`, the fixture corpus is real captured output with a live conformance check, and `isPronunciation` is a well-derived rule that correctly overrode the plan gate's wrong suggestion. What blocks SHIP is the one thing M1 stakes its safety on: the no-data-loss invariant. It is a *corpus* test, not a *property* test, so it protects exactly the nine shapes that were already sampled — the opposite of its stated purpose ("safe against entries nobody sampled"). I captured 170 live NOAD entries and ran the milestone's own `TestRenderLosesNothing` predicate over them: **12 fail (7%)**, on ordinary words — `present`, `minute`, `content`, `desert`, `even`, `read`, `use`, `iPhone`, `iPad`, `Amazon`, `subject`, `MacBook`. `define iPhone` today prints "combination mobile phone…", silently dropping the leading "A". Separately, `define bank` — one of the two fixtures the issue's Done-when names by hand — renders a bogus `adjective` block whose entire body is `)`. All four root causes are one-to-three-line fixes, and a 12-line `FuzzRenderLosesNothing` seeded from the existing corpus reproduces three of them in under two seconds. Fix, add the property test, re-run.

## 1. Strengths

- **`cmd/define/dict.go` + `dict_darwin.go` / `dict_stub.go` / `dict_fake.go`** — a textbook ARCH-PURE seam. Verified `go build ./...`, `go vet ./...`, and `GOOS=linux CGO_ENABLED=0 go build ./...` all green; the parser genuinely never sees CoreServices.
- **`main.go:60-64`** — `isTerminal` kept out of `Render` so rendering stays pure. Verified: zero ANSI escapes in non-TTY output.
- **`parse.go:71-86` `isPronunciation`** — the word-shape rule is correct where the plan gate's suggested "no ASCII letters" rule was wrong. Re-measured against every pipe span in 170 live entries: no misclassification.
- **`invariant_test.go:31-45` `subsequenceGap`** — reporting the *position* with context, not a bare bool, is what made root-causing all four bugs below fast. That design choice paid for itself during this review.
- **Layered defence against a vacuous green** — `capture.sh`'s `MIN_BYTES` floor plus `dict_fake.go:20-24`'s empty-corpus rejection plus `TestLoadFakeDictionaryRejectsEmptyCorpus`. PQ-4 was addressed properly, not nominally.
- **Plan-gate discipline** — PQ-2's *suggested* rule was re-measured and rejected rather than accepted. That is the right way to receive review.

## 2. Critical findings

### C1 — `Render` prints head fields in fixed order, not source order → reordering loss on any homograph-with-syllabification entry
`cmd/define/render.go:42-51`

`Syllables` is printed before `Homograph`, and `Homograph` before `HeadExtra`. NOAD emits `even 1 e·ven | ˈēvən |` — homograph *then* syllabification. The corpus never exercises both together (`bank 1` has no syllabification; `record rec·ord` has no homograph), so the ordering is unverified.

Failure scenario (verified live): `define present` → raw `present 1 pres·ent | ˈpreznt |`, rendered `present  pres·ent 1`. The invariant predicate reports a gap at rune 24/1335. Same for `even`, `content`, `desert`, `minute`, `Amazon`, and `use` (`use verb 1 [with object]` → rendered `use 1 verb [with object]`).

Fix sketch: emit `Homograph` before `Syllables`. More robustly — since `use` shows `HeadExtra` can legitimately precede the homograph — have `parseHead` record head tokens in source order (e.g. `HeadTokens []headTok{kind, text}`) and let `Render` walk that slice, so order is carried by the data rather than re-guessed by the renderer. This is the same failure class the issue Log credits the invariant with catching twice on `record`.

### C2 — `firstToken` and `trimFirstToken` disagree on what a token boundary is → silent drop on every no-pronunciation entry with a newline
`cmd/define/parse.go:363-376`, consumed at `parse.go:113-116`

`firstToken` uses `strings.Fields` (any whitespace); `trimFirstToken` uses `strings.IndexByte(s, ' ')` (literal space only). For `"iPhone\nA combination mobile phone…"`, `firstToken` consumes `iPhone` but `trimFirstToken` consumes `iPhone\nA`.

Failure scenario (verified live, real NOAD output): `define iPhone` prints
```
iPhone

    combination mobile phone, multimedia player, and wireless internet device from Apple. …
```
The word "A" is gone. Reproduced on `iPad` and `MacBook` too. Note the entire `!ok` branch at `parse.go:112-117` has **no test at all** — every fixture has a pronunciation span.

Fix sketch (also the ARCH-DRY consolidation): replace both with one function.
```go
func splitFirstToken(s string) (head, rest string) {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}
```
Two functions independently re-deriving the same boundary, and disagreeing, *is* this bug (ARCH-DRY).

### C3 — `Render` discards a non-empty `Syllables` when it equals `Headword`
`cmd/define/render.go:42`

The `e.Syllables != e.Headword` guard is the only place `Render` deliberately drops a populated field, and `Render`'s own doc comment says it "must not change case, abbreviate, truncate, or reorder."

Failure scenario (verified live): `read` returns `read verb (past and past participle read | red |) [with object] | rēd | 1 look at…`. `parseHead` classifies the second `read` as `Syllables` (it equals the bare headword), and `Render` then suppresses it. Invariant gap at rune 141/4507. Minimal repro: `ParseEntry("in in | ˈin | noun a thing.")` renders `in`, losing the second `in`.

Fix sketch: drop the equality guard from `Render` and instead refuse the classification in `parseHead:143-144` — a token identical to the headword with no `·` is not a syllabification; route it to `HeadExtra`, which is already the "kept, never dropped" bucket. Then `Render` never has to decide what to hide.

### C4 — `posAt` treats `]` and `)` as trailing token boundaries → spurious top-level blocks on `bank` and `run`
`cmd/define/parse.go:327-345` (`isBoundary` / `posAt`)

`isBoundary` includes `]` and `)`, so `adjective]` and `adjective)` inside grammar labels match as part-of-speech tokens and open a new block mid-entry.

Failure scenario (verified, `define bank` — a fixture named in the issue's Done-when): `(banked as adjective) : a banked racetrack` splits the verb block, producing
```
  adjective
    )
      "a banked racetrack"
    3. British English (of a locomotive) …
    4. North American English (in pool and other games) …
```
Verb senses 3 and 4 are orphaned under a fake `adjective` heading whose body is a lone `)`. Same on `run`: `6 (the run) [usually with adjective] the average…` splits the noun block, moving noun senses 6–14 into a phantom `adjective` block. The alnum invariant cannot see this because order is preserved.

Fix sketch: use a *whitespace-or-EOS* trailing boundary in `posAt:340` (keep `isBoundary` for the leading check at `:334`). I verified this over all nine fixtures: it removes exactly the two false positives and keeps every true positive —

| fixture | before | after |
|---|---|---|
| `bank` | `[noun verb adjective]` | `[noun verb]` |
| `run` | `[verb noun adjective]` | `[verb noun]` |
| other 7 | unchanged | unchanged |

The only live `verb)` / `adjective]` occurrences that *should* match sit inside `ORIGIN`/`PHRASAL VERBS`, which `splitSections` already peels off — so the `]`/`)` trailing boundary buys nothing and costs two bugs.

## 3. Important findings

### I1 — The invariant is corpus-scoped, so it delivers none of its stated purpose (ARCH-PURPOSE)
`cmd/define/invariant_test.go:47-65`

`atlas/define.md` and the plan both claim the invariant is "what makes a best-effort parser safe against entries nobody sampled." It runs over exactly the entries someone sampled. Failure scenario: any of C1–C4 — all four ship green.

Fix sketch: add a fuzz target seeded from the corpus. This found C2, C3, and the I3 overwrite in under 2 s each when I ran it:
```go
func FuzzRenderLosesNothing(f *testing.F) {
	d, err := loadFakeDictionary("testdata/entries")
	if err != nil { f.Fatal(err) }
	for _, raw := range d.entries { f.Add(raw) }
	f.Fuzz(func(t *testing.T, raw string) {
		out := Render(ParseEntry(raw), RenderOpts{Color: false})
		if i := subsequenceGap(alnum(raw), alnum(out)); i >= 0 {
			t.Fatalf("alnum loss at rune %d for %q", i, raw)
		}
	})
}
```
Commit the minimized crashers under `testdata/fuzz/` as regressions.

### I2 — Fixture corpus misses three structural shapes that all ship bugs
`cmd/define/testdata/capture.sh:19`

`words=(sycophantic quokka ephemeral defenestrate bank record run gaslighting set)` has no entry with (a) homograph **and** syllabification (C1), (b) no pronunciation span at all (C2), (c) a headword-equal syllabification token (C3). Add `present` (or `content`), `iPhone`, and `read`. The bracket cases (C4) are already present in `bank`/`run` — they were simply never asserted.

### I3 — `parseHead` overwrites single-valued fields instead of preserving, contradicting its own contract
`cmd/define/parse.go:139-153`

`ParseEntry`'s doc comment (`parse.go:104-106`) promises "It never discards input: any token it cannot classify is preserved." But `case isDigits(tok): e.Homograph = tok` and `e.Syllables = tok` overwrite on a second match. Failure scenario: a head with two digit-only tokens keeps only the last; found by fuzz on `"|0 0 0 ||0|0"`. Fix: guard each assignment with `&& e.Homograph == ""` / `&& e.Syllables == ""` and fall through to `HeadExtra`.

### I4 — A block pronunciation equal to `Entry.IPA` is consumed but never stored
`cmd/define/parse.go:216-221`

`if ipa != entryIPA { b.IPA = ipa }` is followed unconditionally by `text = after` — when the two match, the span is dropped outright. `Block.IPA`'s comment says "Empty means same as Entry.IPA," but `Render` never falls back, so nothing is printed. Failure scenario (constructed; not observed in the 170 live entries, hence Important not Critical): `ParseEntry("wug | wʌg | noun a thing. verb | wʌg | 1 to wug.")` loses the second `wʌg`. Fix: always `b.IPA = ipa`; if the duplicate is visually noisy, record `b.IPADup = true` and still emit it — suppression is a rendering choice that costs fidelity.

### I5 — `senseSplit` splits on any bare numeral in prose
`cmd/define/parse.go:247`

`(?:^|\s)(\d+)\s|•` matches numerals inside example sentences. Verified on `run`: the verb block gets fake senses `200` ("she ran in the 200 meters"), `42` ("Dave has run 42 marathons"), `11` ("inflation is running at 11 percent"); the noun block gets `398` and `150`. Rendered sense numbering for `define run` reads `1 200 42 2 3 4 11 5 6 …`. Order is preserved so the invariant is blind to it. Fix sketch: accept a numbered split only when the number opens the block (`1`) or continues the sequence (`prev+1`) — that rejects all five false positives above and keeps every real one.

### I6 — No structural assertion over the corpus; no `render_test.go` at all
Plan Task 5 lists `cmd/define/render_test.go`; it was never created. `Render` has **no** direct unit test — `Color: true` is entirely uncovered (the whole `palette` path), as are `prettyPronunciations`, homograph rendering, and the `FromHead` header-suppression branch. Those last two were added *specifically* to fix invariant failures during M1 and are pinned only indirectly.

Equally, nothing asserts per-fixture *structure*, which is why C4 and I5 ship silently. Add a golden table (or golden rendered-output files) covering all nine fixtures: block POS list, sense-number sequence, section names. That is precisely the class of bug this diff shipped.

### I7 — Issue Spec drift: the stated invariant is not the implemented one
`workshop/issues/000001-define.md` `## Spec`

The Spec says "**every non-whitespace character** of the NOAD entry must survive into the rendered output." The implementation asserts letters+digits only. I ran the Spec-level predicate over the corpus: it fails on **all nine** fixtures, at the header's first `|`. The plan's Task 5 documents and justifies the relaxation ("punctuation may be restructured"), but the issue's Spec — the artifact the Done-when list points at — still asserts the stronger contract. Revise the Spec text to the letters+digits rule, or add a `## Revisions` note recording the narrowing and why.

### I8 — `atlas/define.md` states a guarantee the code does not provide
`atlas/define.md:44-48` — "This is what makes a best-effort parser safe against entries nobody sampled." False as written (7% live failure rate). Correct the claim, or land I1 first and then the claim becomes true.

## 4. Minor findings

- `go.mod:6-8` — `golang.org/x/term` is marked `// indirect` but imported directly by `main.go:9`; `go mod tidy` rewrites 3 lines.
- `dict_darwin.go:38-46` — `noad_lookup` returns `NULL` for three distinct conditions (CFString creation failure, no definition, UTF-8 conversion failure); all surface as `ErrNoEntry`, so a genuine failure reports "no dictionary entry".
- `dict_fake.go` is a non-test file, so `fakeDictionary`/`loadFakeDictionary` link into the shipped binary. Both consumers are `_test.go`; renaming it `dict_fake_test.go` costs nothing.
- `splitLeadingPronunciation` (`parse.go:96`) scans the whole string for the *first pronunciation-shaped* span, not a leading one — the name overpromises, and `newBlock` relies on the scanning behaviour.
- `define -h` exits 2; conventionally `-h` is 0 (`flag.ErrHelp` deserves its own branch in `run`).
- README documents the `define` binary but not `-raw` / `-no-color`. The incorrect `make install` claim is knowingly deferred to M2 Task 10 Step 5 — fine, but it is now user-visible for a milestone.
- Commit `f993dfd` untracked 5.6 MB of build artifacts; the blobs remain in history. Cheap to drop while the branch is unmerged.

## 5. Test coverage notes

Verified green: `go test ./...`, `go vet ./...`, `GOOS=linux CGO_ENABLED=0 go build ./...`. `TestRunUnknownWordExitsOne`, `TestRunNoColorWhenNotATerminal`, and `TestLoadFakeDictionaryRejectsEmptyCorpus` are real assertions on real logic, not mocks restating the implementation — good.

Gaps, in payoff order: (1) no property/fuzz test, so the invariant covers 9 strings out of an open-ended input space (I1); (2) no structural golden over the corpus (I6, catches C4 + I5); (3) `ParseEntry`'s no-pronunciation branch (`parse.go:112-117`) has zero coverage (C2 lives there); (4) `Render` has no direct test, so the entire colour path and `prettyPronunciations` are unexercised (I6); (5) no fixture has homograph + syllabification (C1) or a headword-equal syllabification (C3).

The conformance test (`dict_conformance_test.go`) is well-built and correctly gated; the on-demand cadence is stated with a reason in both the test and `atlas/define.md`, which satisfies ARCH-MOCK.

## 6. Architectural notes

- **ARCH-DRY — flag.** `firstToken` (`parse.go:363`) and `trimFirstToken` (`parse.go:370`) are near-identical helpers that re-derive the same token boundary and disagree; that disagreement is C2. Consolidate into one `splitFirstToken(s) (head, rest)`. Separately, `capture.py`'s duplication of the cgo call is deliberate, one-directional, and documented at `capture.py:4-7` — correct call, no action.
- **ARCH-PURE — pass.** `ParseEntry`, `Render`, `isPronunciation`, and `subsequenceGap` are pure string→value functions with no injected doubles; `TestIsPronunciation` and `TestParseHeader` run on literals with zero IO. `isTerminal` is correctly parked at the boundary. `parse_test.go`'s `fixture()` reads `testdata/` through the fake, which is read-only idiomatic Go, not a mock.
- **ARCH-PURPOSE — flag.** The Non-goals section is exemplary about the single-homograph limit and about fidelity-vs-completeness. But the invariant, which the whole best-effort-parser design rests on, is delivered as a nine-entry corpus check rather than as the property its own documentation claims. That is the cheap win standing in for the purpose (I1).
- **ARCH-MOCK — pass.** `Dictionary` has a fixture-backed fake behind the same seam, production and test flow share that boundary, and a live conformance check with a stated cadence and trigger exists. For M2, mirror `dict_fake.go`'s shape for `fakeCDN`/`fakePlayer` but put them in `_test.go` files.
- **For M2:** `deps` will gain `audio` and `player`; keep `playN`'s loop in the shell as planned so the fake can count. Extend the I1 fuzz harness to `AudioCandidates` (the plan already flags the one-letter-word shard case). And treat the `Entry` head model — see C1's `HeadTokens` sketch — as the place to fix ordering once, before the audio path adds another consumer of `Entry`.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000001-define-plan.md`:

1. **Core concepts drift (M1 implementation).** `Entry` gained `HeadPOS`; `Block` gained `FromHead bool` and `Label string`. Both were added to fix invariant reordering failures on `record` (recorded in the issue `## Log`, absent from the plan). The Chunk 1 bullets for `Entry` and `Block` still list the pre-M1 field sets.
2. **Task 5 file list.** `cmd/define/render_test.go` is listed but was never created. Either create it (see I6) or strike it from the Files line.
3. **Invariant scope.** Task 5 and the Non-goals should state that the invariant as written is corpus-scoped, and add the property/fuzz obligation (I1) plus the three missing corpus shapes (I2) as explicit M1 steps.
4. **Task 6 / Task 5 commit merge.** Tasks 5 and 6 landed as one commit (`21ea7b3 #1 M1: render Entry + CLI wiring`), as did Tasks 2 and 3. Cosmetic, but worth noting so the plan's step list stays readable as a record.

Separately, for `workshop/issues/000001-define.md`: revise the `## Spec` invariant wording per I7, and carry the two estimate-quality INFO observations recorded in the `## Log` ("routed to the close review via the gate ledger") forward explicitly — they are not in `000001-define-plan-gate.md`, whose `## Open findings` is empty, so nothing currently carries them to the M2 close.

---

## Re-review — 2026-08-20T10:48:07-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 1 — define — NOAD word lookup with Google-style IPA and spoken pronunciation |
| repo | tools |
| issue file | workshop/issues/000001-define.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e8386b80feb8986ae8a2deb3e85d02012c0ee4a^..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-20T10:48:07-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

The rework is substantially good and I verified its central claim independently: I ran the milestone's own predicate over **2,749 live NOAD entries and got 0 losses** (the pre-rework state was 7%), ran `FuzzRenderLosesNothing` for 16.3M executions with no crasher, and confirmed all four prior Criticals are genuinely fixed in rendered output (`record` head order, `present 1 pres·ent`, `read`'s parenthetical, `iPhone`'s restored leading "A"). Coverage is 90.6%, `go vet` clean, `go mod tidy` a no-op. What blocks the boundary is that the *class* of bug the prior review called C4 is only half-fixed: a part-of-speech word inside a bracketed grammar label still opens a phantom top-level block, so `define man` and `define thing` — two of the most common words in English — print invented `adjective`/`noun` headings and tear a sub-sense in half. The alnum invariant is blind to it by construction (order is preserved) and the 8-word structural golden doesn't sample a bracketed label. I reproduced it live, wrote the 8-line fix, and re-verified: all existing tests pass and the live sweep stays 0/2749. Second blocker: the plan's Revisions section states "Chunk 1 above is updated in place," but Chunk 1 still describes the pre-rework model, including a `Block.IPA` contract that the code now deliberately inverts.

## 1. Strengths

- **The invariant is now the property it claimed to be.** Three widths — 21 fixtures, corpus-seeded fuzz, and `live_property_test.go:34` walking a stride sample of `/usr/share/dict/words` through the real dictionary — sharing *one* predicate (`subsequenceGap`/`alnum` in `invariant_test.go:32,42`). That is the correct ARCH-DRY shape, and it closes the prior I1 honestly rather than nominally. Verified: 2749 checked, 0 failed.
- **`parse.go:131` `findPronunciation` tracking paren depth** is the right root-cause fix for `read`, not a special case. Confirmed on live output.
- **`parse.go:196` `parseHead`'s `add` closure** — demoting a repeated single-valued kind to `HeadOther` instead of overwriting is exactly the "kept, never dropped" contract the doc comment promises, and it's the structural fix for I3 rather than a guard bolted on.
- **`parse.go:387` `splitFirstToken`** consolidating the two disagreeing helpers, with the comment naming the bug that motivated it. This is how an ARCH-DRY fix should read.
- **`live_property_test.go:63` `if checked < 500 { t.Fatalf(...) }`** — the sandbox guard means this test cannot pass vacuously, which is the same discipline `capture.sh`'s `MIN_BYTES` applies to the corpus. Layered, not decorative.

## 2. Critical findings

### C1 — A part-of-speech word inside a bracketed grammar label still opens a phantom top-level block
`cmd/define/parse.go:267-276` (the `posAt` scan in `parseBlocks`); the stale rationale is at `cmd/define/parse.go:449-453`

The rework removed `]` and `)` as *trailing* boundaries, which fixed `(banked as adjective)`. It did not handle the case where the POS sits **inside** a bracket followed by a space: `[with adjective or noun modifier]`. Both `adjective` and `noun` there have whitespace on each side, so `posAt` matches and `parseBlocks` opens two new top-level blocks.

The comment at `parse.go:449-453` asserts the reason this is safe — "Real POS tokens followed by a bracket only occur inside ORIGIN / PHRASAL VERBS, which `splitSections` has already peeled off." That reasoning covers POS-then-bracket; it does not cover bracket-then-POS, which is where the remaining bug lives.

Failure scenario (verified live, real NOAD output). `define thing`:

```
      • (things) [with

  adjective
    or

  noun
    modifier] equipment, utensils, or other objects used for a particular purpose
```

`define man` is the same, splitting `[with adjective or noun modifier]` and orphaning the Cro-Magnon sense under a phantom `noun` heading whose body begins `modifier] a type of prehistoric human…`. Block lists measured live: `man` → `[noun adjective noun verb exclamation]` (should be `[noun verb exclamation]`), `thing` → `[noun adjective noun]` (should be `[noun]`).

Prevalence is 2/2749 live entries (0.1%), but both are top-100 English words — `man` and `thing` are plausible first inputs. The alnum invariant cannot see this (letter order is preserved), and `render_test.go:120` `TestCorpusBlockStructure` covers 8 words, none containing a bracketed POS label.

Fix sketch — track bracket depth in the same loop that already scans for POS marks, mirroring what `findPronunciation` does for parens:

```go
var marks []mark
depth := 0
for i := 0; i < len(body); {
	switch body[i] {
	case '[':
		depth++
	case ']':
		if depth > 0 {
			depth--
		}
	}
	pos, width := posAt(body, i)
	if pos == "" || depth != 0 {
		i++
		continue
	}
	marks = append(marks, mark{i, pos})
	i += width
}
```

I applied this in a scratch copy and re-verified: `go test ./...` passes (including the 8-word golden and the 21-fixture invariant), `man` → `[noun verb exclamation]`, `thing` → `[noun]`, and the live property check stays **0 failures / 2749 checked**. Also update the `posAt` comment, which currently documents only half the rule.

Add a golden row for `man` (or `thing`) to `TestCorpusBlockStructure` and a fixture to `capture.sh` — this is the third boundary-visible bug in the "POS token in a grammatical aside" family, and the corpus still has no entry that exercises it.

### C2 — The plan's Core-concepts section contradicts the code, while its Revisions section claims it was reconciled
`workshop/plans/000001-define-plan.md:130-160`

The Revisions entry says "**Core concepts drift, now reconciled (Chunk 1 above is updated in place)**". Chunk 1 was not updated. Four contradictions, checked against the code:

| Plan says | Code has |
|---|---|
| `Entry` fields `Headword`, `Syllables`, `Homograph`, `HeadExtra []string` (line 144) | `Head []HeadTok` (`parse.go:40`); those are *accessors* (`parse.go:58-61`); `HeadExtra` does not exist — it is `HeadOther` |
| `Block.IPA`: "Empty means 'same as `Entry.IPA`'; `Render` … prints a block-level pronunciation **only when it differs**" (line 149) | Stored unconditionally, "even when it equals the entry's: suppressing it would silently drop input" (`parse.go:73-76`); `Render` prints whenever non-empty (`render.go:85`) — the **inverse** contract, and the I4 fix |
| `fakeDictionary` → `cmd/define/dict_fake.go` (lines 142, 214) | `cmd/define/dict_fake_test.go` |
| no row for `HeadTok` / `HeadKind` | new pure entities, `parse.go:11-32` |

The `Block.IPA` row is the dangerous one: M2 adds a second `Entry` consumer, and the plan currently instructs that consumer to expect fallback semantics the parser no longer has. Fix: update the Chunk 1 tables and bullets in place (which is what the Revisions entry already promises), or change the Revisions wording to point at the delta instead of claiming an in-place edit that didn't happen.

## 3. Important findings

### I1 — `loadFakeDictionary` doesn't normalize keys, so 4 of 21 fixtures are unreachable through the seam
`cmd/define/dict_fake_test.go:41` vs `:47`

Keys are the raw filename stem (`Amazon`, `MacBook`, `iPad`, `iPhone`); `Lookup` looks up `strings.ToLower(word)`. So `fake.Lookup("iPhone")` returns `ErrNoEntry`. I verified the real dependency is case-**insensitive** (`capture.py amazon` and `capture.py Amazon` both return the entry), so the lowercasing policy is right and the load side is wrong.

Failure scenario: `run([]string{"iPhone"}, testDeps(t), …)` exits 1 in a test while the real binary exits 0. That means the no-pronunciation path — the exact path C2 shipped a silent word-drop in, and the reason `iPhone`/`iPad`/`MacBook` were added to the corpus — **cannot be exercised end-to-end through `run()`**. It is reachable only by iterating `d.entries` directly, which is what `invariant_test.go:52` does.

This is an ARCH-MOCK finding: the fake diverges from the modeled dependency's behavior at the seam. It is also why the divergence is invisible — `dict_conformance_test.go:25` iterates `fake.entries` directly rather than calling `fake.Lookup`, so the conformance check bypasses the boundary it exists to validate (ARCH-DRY: two readers of the fake, one bypassing its accessor).

Fix: lowercase at load — `d.entries[strings.ToLower(strings.TrimSuffix(filepath.Base(p), ".txt"))]` — and add an assertion that every fixture on disk is reachable via `Lookup`.

### I2 — `prettyPronunciations` is applied only to sections, so ~10% of entries render raw `| … |` in the body
`cmd/define/render.go:116` (only call site)

The tool's stated purpose is Google-style `/…/` notation. `prettyPronunciations` is applied to `Section.Text` and nowhere else, so pronunciations appearing in `Block.Label`, sense glosses, and examples keep NOAD's raw pipes. The same screen shows both notations.

Measured live: **273 of 2749 entries (9.9%)** render at least one raw `|` in the body. Verified example — `define alewife`:

```
  noun
    (plural alewives | ˈālˌwīvz |) a northwestern Atlantic fish …

  ORIGIN
    mid 17th century: possibly from …
```

`define man` shows `(plural men | men |)` the same way. Fix: run `Block.Label`, `Sense.Gloss`, and `Sense.Examples` through `prettyPronunciations` too. It is already punctuation-only and invariant-safe by construction (`render.go:124-127`), and `isPronunciation` correctly leaves prose spans alone (`render_test.go:70-74` pins that). ARCH-PURPOSE: the helper exists but is wired to one of the three places the problem occurs.

### I3 — Sense numbering collapses entirely when the head swallows sense "1"
`cmd/define/parse.go:320-336`

The `want`-sequence guard is the right fix for I5's prose numerals, but it anchors at 1 unconditionally. When `parseHead` classifies a leading sense number as a homograph (`use verb 1 [with object] | yo͞oz |` → `1` becomes `HeadHomograph`), the body starts at sense **2**, `want` is still 1, and every numbered sense in that block is rejected. They merge into the preceding sense's text.

Failure scenario (verified live). `define use`, verb block:

```
        "yo͞ost"
        "be or become familiar with someone or something through experience: she was used to getting what she wanted"
        "you just have to get used to him. 5 (one could use) informal"
        "yo͞oz"
```

Sense numbers 2–5 are gone, per-sense pronunciations render as quoted "examples", and a literal `5` appears mid-quote. Same on `bases`. Measured: **12 of 2749 live entries** have a block where the raw shows numbering the parser produced no sense for.

`TestCorpusSenseNumbersAreSequential` (`render_test.go:96`) cannot catch this — it validates numbers that *were* assigned, not numbering that was never assigned. Fix sketch: seed `want` from the entry's `HeadHomograph` when it was taken from the position immediately preceding the body (i.e. treat `use … 1 …` as sense 1 opening block 0), or accept the first numbered split when it is ≤ 2 and no earlier number was seen. Add a coverage assertion of the shape "if the raw body contains a ` 2 ` split candidate and the block has no sense 1, flag it."

### I4 — 30 of 30 M1 task checkboxes in the durable plan are unticked at milestone close
`workshop/plans/000001-define-plan.md`, Chunk 2 (Tasks 1–6)

`sed -n '/## Chunk 2: M1/,/## Chunk 3/p' | grep -c '^- \[ \]'` → 30; `'^- \[x\]'` → 0. The issue's `## Plan` correctly shows `- [x] M1`, but the durable plan — the artifact AGENTS.md §8 says to tick per milestone — records nothing as delivered. It is the record of truth for this boundary and currently reads as though M1 never started. (Note Task 5 Step 1 and Task 6 Step 3 *were* delivered; this is a bookkeeping gap, not a delivery gap.)

## 4. Minor findings

- `main.go:52-56` collapses `ErrNoEntry`, `ErrLookupFailed`, and the non-darwin stub's error into exit 1, discarding the three-way `status` distinction `dict_darwin.go:20-24` went to trouble to preserve; README documents exit 1 as "no dictionary entry" specifically. The message still differs, only the code is merged.
- `parse.go:214-219` can't segment a multiword glued POS: `bases ba·sesplural noun` leaves `ba·sesplural` as one `HeadOther` token because `matchPOS("plural")` requires the full `"plural noun"`.
- `parse.go:363` splits gloss/examples at the first `:`, so a grammar label lands inside a quoted example — `define bank` sense 4 renders `"[as modifier] : a bank shot"`.
- `parse.go:463` re-inlines the whitespace test byte-by-byte instead of a named helper sitting beside `isBoundary` two lines up.
- `invariant_test.go:98` `len([]rune(alnum(raw)))` — `alnum` already returns `[]rune`; the conversion is a no-op.
- `atlas/define.md` says crashers land in `testdata/fuzz/`; no such directory exists. Accurate today (none found in 16.3M execs) — worth a word so a reader doesn't think it was pruned.
- README says `go test -tags conformance ./...`; `atlas/define.md` says `./cmd/define/`. Both work; pick one.

## 5. Test coverage notes

90.6% of statements, `go vet` clean, `go build` green on darwin and under `GOOS=linux CGO_ENABLED=0`. `TestRunNoColorWhenNotATerminal`, `TestLoadFakeDictionaryRejectsEmptyCorpus`, and `TestRenderColorOnlyWhenAsked`'s strip-and-compare (`render_test.go:60-62`) are real assertions on real logic. The prior I6 gap is genuinely closed: `render_test.go` exists and pins the colour path, `prettyPronunciations`, homograph rendering, and `FromHead` suppression.

Remaining gaps, in payoff order:

1. **No golden covers a bracketed grammar label** — `TestCorpusBlockStructure` samples 8 words, none with `[with adjective or noun modifier]`. That is C1's hiding place, and the second time this family has reached a boundary undetected.
2. **No `run()`-level test reaches the no-pronunciation branch** (`parse.go:164-174`), because I1 makes its three fixtures unreachable through `Lookup`. `TestRenderLosesNothing` covers them only via direct map access.
3. **Sense-number *absence* is unasserted** (I3) — the existing golden validates assigned numbers only.
4. **The fake's own contract is untested** — nothing asserts that every file in `testdata/entries` is retrievable through `Lookup`, which is the one property the fake exists to provide.
5. `realDeps` and `main` are at 0% — correct and not worth chasing.
6. No committed fuzz corpus under `testdata/fuzz/`; the seed set in `invariant_test.go:86-88` is doing that job today.

## 6. Architectural notes

- **ARCH-DRY — flag (minor).** The three-width invariant sharing one predicate is a genuine win. Two remaining duplications: `dict_conformance_test.go:25` reaches into `fake.entries` rather than going through `Lookup`, which is precisely why I1 hides; and `posAt:463` re-inlines the whitespace test beside the existing `isBoundary`. `capture.py`'s duplication of the cgo call stays correct — deliberate, one-directional, documented at `capture.py:4-7`.
- **ARCH-PURE — pass.** `ParseEntry`, `Render`, `isPronunciation`, `alnum`, `subsequenceGap` are pure string→value functions; their tests run on literals with no IO. `isTerminal` is correctly parked at the boundary (`main.go:64`) and `Render` never probes the terminal — I confirmed zero ANSI escapes on a non-TTY writer. cgo is confined to `dict_darwin.go`. No "pure" entity needs a mock to run.
- **ARCH-PURPOSE — flag.** The invariant now fulfils its stated purpose; I verified the atlas's "0%" claim independently. But the issue's purpose is Google's *notation*, and `prettyPronunciations` — the function that delivers it — is wired to one of the three places pronunciations appear, leaving ~10% of entries mixing `| … |` and `/…/` on one screen (I2). The helper exists; only one consumer derives from it.
- **ARCH-MOCK — flag.** Seam, fixture-backed fake from real captured output, and a live conformance check with a stated cadence and trigger: the shape is right and `TestFixturesMatchLiveDictionary` passes against live NOAD. The flag is that the fake's behavior diverges from the modeled dependency on case (I1) and the conformance check bypasses the seam, so production flow and test flow do **not** share the boundary for those four fixtures.
- **For M2:** fix the `Entry` head model's remaining consumers *before* audio lands — `AudioCandidates` will be the second `Entry` reader and the plan currently documents a `Block.IPA` contract the code inverted. Put `fakeCDN`/`fakePlayer` in `_test.go` files as `fakeDictionary` now is, and give them the same reachability assertion I1 is missing. Extend the fuzz harness to `AudioCandidates` (the plan already names the one-letter-word shard case).

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000001-define-plan.md` — and reconcile Chunk 1 itself, which the existing entry already claims to have done:

1. **Chunk 1 Core-concepts reconciliation (C2).** Update the `Entry` bullet to `Head []HeadTok` with `Headword`/`Syllables`/`Homograph`/`HeadPOS` as accessors and `HeadOther` replacing `HeadExtra`; add table rows for `HeadTok` and `HeadKind` (`cmd/define/parse.go`, new); correct the `fakeDictionary` path to `cmd/define/dict_fake_test.go` in both the Integration table (line 142) and Task 2's Files line (line 214).
2. **`Block.IPA` contract inversion (C2).** The bullet at line 149 still specifies "empty means same as `Entry.IPA`, printed only when it differs." Replace with the implemented contract: always stored when present, always rendered when non-empty, because suppression drops input. Record why — this was I4 from the first boundary review.
3. **Tick the M1 task steps (I4 above).** All 30 Chunk 2 checkboxes are `- [ ]` at milestone close.
4. **Amend the existing Revisions entry.** It closes with "Deferred to M2 with reason: nothing. All Critical and Important findings are addressed in this milestone." C1 is a surviving edge of that entry's own `posAt` fix; note it rather than leaving a claim the code contradicts.

Separately for `workshop/issues/000001-define.md`: the `## Log` says the two estimate-quality INFO observations "reach the close review via the gate ledger," but `000001-define-plan-gate.md`'s `## Open findings` is empty, so nothing carries them to the M2 close. The prior review raised this and it is still open — record them in the ledger or in the issue's `## Plan` for M2.

---

## Re-review — 2026-08-20T11:08:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 1 — define — NOAD word lookup with Google-style IPA and spoken pronunciation |
| repo | tools |
| issue file | workshop/issues/000001-define.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e8386b80feb8986ae8a2deb3e85d02012c0ee4a^..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-20T11:08:24-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

Round 2's fixes hold: I re-verified every one of them live rather than taking the commit message's word for it — block structure is correct across 2,749 live entries (zero phantom blocks, zero non-consecutive or high-starting sense numbers), `use`'s senses 2–5 are back, the fixture corpus matches live NOAD byte-for-byte, and `FuzzRenderLosesNothing` survived 7.9M executions. What blocks the boundary is that the milestone's headline claim — `atlas/define.md`'s "Before the review, 7% of live entries lost content… it is 0% now" — is measured by an oracle that **is the code under test**. Both raw-pipe checks (`live_property_test.go:57`, `render_test.go:208`) ask `isPronunciation` whether a surviving `|…|` span is a pronunciation, so the rule's own false negatives are invisible to them by construction. Under an independent oracle (a NOAD stress mark appearing outside a `/…/` span), **61 of the same 2,749 entries (2.2%) render raw, unconverted NOAD pronunciation notation** — not 0. Two root causes: `isPronunciation` rejects every multi-word pronunciation (`| ˈhät ˌdäɡ |`), and `findPronunciation` searches the whole entry unbounded, so an entry lacking its own pronunciation silently adopts a *derivative's*. Together they make `define concrete`, `define "hot dog"`, `define methodological`, `define bucktooth`, and `define apriori` collapse the entire definition into one unreadable head line and display **a different word's pronunciation** as the entry's. I reproduced all of it live at HEAD, trialled a fix in a scratch copy, and measured it down to 8/2,749 with every existing test still green. Fix, replace the self-referential oracle, re-run.

## 1. Strengths

- **Round 2's block-opener rule is genuinely right, and I proved it at a width the goldens can't reach.** I swept 2,749 live entries for duplicate-POS blocks, non-consecutive sense numbers, sequences starting above 2, and empty senses: **zero** of each. `opensBlock` (`parse.go:549`) retired that whole bug family, not just its three known shapes. This is confirmed-good ground — don't re-litigate it.
- **The three-width invariant shares one predicate** (`alnum`/`subsequenceGap`, `invariant_test.go:32,42`) across fixtures, fuzz, and live. Re-measured: 2,749 live entries, 0 content loss. That claim is true and the ARCH-DRY shape is correct.
- **ARCH-MOCK is properly closed.** `dict_conformance_test.go:28` reads through `fake.Lookup`, not `fake.entries` — the round-2 I1 fix landed as designed. I independently verified the case-insensitivity the fake models: `capture.py Amazon`, `amazon`, and `AMAZON` all return the same entry, so lowercasing at load matches the real dependency.
- **`parse.go:131` `findPronunciation`'s paren-depth tracking** is the right root-cause fix for `read`, and `parse.go:463` `splitFirstToken`'s comment naming the bug that motivated it is how a DRY fix should read.
- **Bookkeeping is clean this round.** All 30 Chunk 2 checkboxes ticked, `go mod tidy` a no-op, no build artifacts tracked or in history (`git log --all -- bin define` is empty), coverage 91.0%, `go vet` clean, `GOOS=linux CGO_ENABLED=0 go build && go vet` green.

## 2. Critical findings

### C1 — `isPronunciation` rejects every multi-word pronunciation, so 2.2% of entries render raw NOAD pipes and some collapse entirely
`cmd/define/parse.go:111`

The rule is "a span is a pronunciation iff every comma-separated part is a single space-free token." NOAD writes multi-word headwords' pronunciations *with interior spaces*: `hot dog | ˈhät ˌdäɡ |`, `a priori | ˌā prīˈôrī |`, `above board | əˈbəv ˌbôrd |`, `on behalf of | ˌän bəˈhaf əv, ˌôn bəˈhaf əv |`. All are rejected.

Two consequences, both verified against live NOAD at HEAD:

1. **Raw notation on screen.** Over all 71,427 reachable entries, 1,062 render a surviving multi-token `|…|` span. Under the independent stress-mark oracle over the standard 2,749-entry sample: **61 entries (2.2%)**. Examples: `define article` → `the genuine article | T͟Hə ˌjenyəwən ˈärdək(ə)l |`; `define block` → `block out | ˌbläk ˈout |`; `define agrammatical` → `| BrE ˌeɪɡrəˈmatɪk(ə)l, AmE ˌeɪɡrəˈmædɪk(ə)l |`.
2. **Total collapse.** When the entry's *own* pronunciation is the rejected one, `findPronunciation` walks past it and adopts a later one, dumping everything before it into the head. `define "hot dog"` at HEAD:
```
hot dog | ˈhät ˌdäɡ |  noun 1 a frankfurter, especially one served hot in a long, soft roll and
topped with various condiments: he's ordering a hot dog | a package of hot dogs. 2 North American
English informal a person who shows off … DERIVATIVES hotdogger
/ˈhätˌdäɡər/

  noun
```
`/ˈhätˌdäɡər/` is *hot dogger*. Same for `define bucktooth` → `/ˌbəkˈto͞oTHt/` (buck-toothed) and `define apriori` → `/āˈprīəˌrizəm/` (apriorism).

Fix sketch — keep the single-token rule as sufficient, and admit a multi-token span when it carries NOAD stress marks, which prose never does:
```go
if strings.ContainsAny(inner, "[]:;.") { return false }
for _, part := range strings.Split(inner, ",") {
	fields := strings.Fields(part)
	if len(fields) == 0 || len(fields) > 8 { return false }
	marked := false
	for _, f := range fields {
		if strings.ContainsAny(f, "ˈˌ") { marked = true }
	}
	if !marked { return false }
}
return true
```
I measured this against every distinct multi-token span in the live dictionary: **1,025 accepted** (all genuine — `ˌsānt ˈjəstən`, `ˌslīd əv ˈhand`, `BrE ˌʌndɪsˈtʃɑːdʒd, AmE ˌəndɪsˈtʃɑrdʒd`), **9,697 rejected** (all prose — "she ran her fingers through her hair…", "[as modifier] : a pirate ship…"). Applied in a scratch copy: every existing test passes, the live property check stays 0 loss / 2,749, stray stress drops 61 → 8, and `hot dog`, `bucktooth`, and `apriori` render fully structured with the correct `/ˈhät ˌdäɡ/`, `/ˌbək ˈto͞oTH/`, `/ˌā prīˈôrī/`. Raise the token cap past 6 to clear the residual 8 (5–6-token phrase pronunciations).

### C2 — `findPronunciation` is unbounded, so an entry with no pronunciation adopts a derivative's and loses its whole structure
`cmd/define/parse.go:131`, consumed at `parse.go:174-176`

`findPronunciation` scans the *entire* raw entry for the first pronunciation-shaped span. NOAD run-on entries carry no pronunciation of their own but do carry one under `DERIVATIVES`, so the parser reaches past every block and section boundary and takes it — treating all the intervening text as head tokens.

Failure scenario (verified live, and **not** fixed by C1's patch). `define concrete`:
```
concrete  con·crete  adjective existing in a material or physical form; not abstract: concrete
objects like stones | … noun a heavy, rough building material … verb [with object] 1 cover (an
area) with concrete … PHRASES be set in concrete … DERIVATIVES concreteness
/känˈkrētnəs, kənˈkrētnəs, ˈkänˌkrētnəs/

  noun

  ORIGIN
    late Middle English (in the sense ‘solidified’): …
```
The adjective, noun, and verb blocks and PHRASES are all gone into a single head line, and the displayed pronunciation is *concreteness*. Affected, measured over 52,721 distinct live entries: **8** — `concrete`, `maniacal`, `melancholic`, `meteorological`, `methodological`, `smudgily`, `stinging`, `transect`. Small in count, but they are ordinary words and the per-entry output is unusable with no signal to the user that anything went wrong.

Fix sketch: bound the search to the head region — stop at the first structural block opener (`opensBlock` + `posAt`) or the first section word, whichever comes first. If no pronunciation is found there, take the existing `!ok` branch (`parse.go:167-174`) so the body parses normally with an empty `Entry.IPA`. That also converts any future `isPronunciation` false negative from "catastrophic collapse" into "missing IPA," which is the failure mode you want.

## 3. Important findings

### I1 — Both raw-pipe checks use `isPronunciation` as their own oracle, which is why C1 and C2 shipped green through two boundary reviews
`cmd/define/live_property_test.go:57`, `cmd/define/render_test.go:208`

Both are `for _, m := range pipeSpanRe.FindAllStringSubmatch(out, -1) { if isPronunciation(m[1]) { …fail… } }`. A test that asks the function under test whether its own output was correct can detect false *positives* only. Every span C1 misses is, by definition, a span `isPronunciation` says is not a pronunciation — so both checks report 0 while 61/2,749 entries show raw notation. The atlas's "0%" is this measurement.

Fix: add an oracle that does not call `isPronunciation`. Verified to work — a NOAD stress mark (`ˈ` or `ˌ`) may appear only inside a `/…/` span:
```go
var slashSpan = regexp.MustCompile(`/[^/\n]*/`)
func strayStress(out string) int {
	return strings.IndexAny(slashSpan.ReplaceAllString(out, ""), "ˈˌ")
}
```
Fires 61 times on HEAD, 8 after C1's fix, 0 once the token cap is raised. Also ARCH-DRY: extract the one check into a shared helper instead of the two copies.

### I2 — No fixture, and no structural golden, is a multi-word headword
`cmd/define/testdata/capture.sh:34`, `cmd/define/render_test.go:120`

All 25 fixtures are single-word headwords, and `TestCorpusBlockStructure` samples 11 of them. C1's entire class is therefore unreachable from the fixture-backed suite — the fake models the dependency's single-word output only. Add `hot dog` (multi-word pronunciation), `concrete` (no entry pronunciation, derivative has one), and `a priori`, with block-structure golden rows. Note `capture.sh` takes the word as an argv element, so multi-word entries capture fine; the filename will need the space handled.

### I3 — Grammar labels are swallowed into quoted examples on 13.8% of entries
`cmd/define/parse.go:413` `newSense`

The split is at the *first* `:`, so `4 the cushion of a pool table: [as modifier] : a bank shot.` yields the example `"[as modifier] : a bank shot"` — label and a stray colon inside the quotes. Measured: **378 of 2,749 live entries**, and it is visible on `bank`, a fixture the issue's Done-when names by hand. `Block.Label` already models exactly this concept at block level (`parse.go:70`); `Sense` has no counterpart. Fix: peel a leading bracketed label off each example segment into a `Sense.Label` (or per-example label) and render it outside the quotes, mirroring what `newBlock` does with `isGrammarLabelOnly`.

### I4 — README documents audio playback that M1 does not ship
`README.md:25,30`

`| define | Print a word's NOAD definition with Google-style IPA, and play its pronunciation. |` and `define sycophantic # definition + /ˌsikəˈfan(t)ik/, pronunciation played 3x`. There is no audio path in the binary at this boundary — `deps` holds only `dict` (`main.go:16`). The README is honest about `make install` landing with M2; it should be equally honest about playback. A reader who runs the documented command at this commit hears nothing and has no way to know that is expected.

### I5 — `atlas/define.md` describes three IO seams when one exists
`atlas/define.md:13-24`

"A pure core with three thin IO seams," a table listing `AudioSource`/`fakeCDN` and `Player`/`fakePlayer`, and "Pure: `ParseEntry`, `Render`, `AudioCandidates`." `grep` finds no `AudioSource`, `Player`, `fakeCDN`, `fakePlayer`, or `AudioCandidates` in the tree — all are M2. AGENTS.md §8 makes the atlas the current-state map; a reader (or agent) navigating by it will look for `fetch.go` and `player.go` and find nothing. Mark the two unbuilt seams as M2, or drop them until they land.

## 4. Minor findings

- Seven of round 2's Minors are still open with no note recording the decision: exit code collapses `ErrNoEntry`/`ErrLookupFailed` to 1 (`main.go:52`); `bases ba·sesplural` can't segment a multiword glued POS (`parse.go:214`); `posAt` re-inlines the whitespace test beside `isBoundary` (`parse.go:463` region); `invariant_test.go:98` `len([]rune(alnum(raw)))` is a no-op conversion; `testdata/fuzz/` still doesn't exist though the atlas points at it; README says `-tags conformance ./...` while the atlas says `./cmd/define/`. Leaving Minors is fine — leaving them *silently* means round 4 rediscovers them.
- `live_property_test.go:24` `liveSampleSize = 9000` with stride 26 covers 2,749 of 71,427 reachable entries (3.8%). A full sweep runs in ~35 s; the `checked < 500` guard would still protect it. Cheap width for the one check that earns the "entries nobody sampled" claim.
- `define parrot` renders `(, parroting "ˈperədiNG")` — an inflection-list pronunciation landing inside a quoted example because the enclosing sense already split on a `:`. Same family as I3.
- `TestRenderGluedPOSNotPrintedTwice` (`render_test.go:34`) asserts only `strings.Count(out, "noun") >= 1`; it would pass if the head POS vanished and the block heading appeared instead — the exact swap it exists to prevent.

## 5. Test coverage notes

91.0% of statements, vet clean, darwin and `GOOS=linux CGO_ENABLED=0` both green, live fixture conformance passes, fuzz clean at 7.9M executions. The structural goldens added in round 2 are real assertions on real logic and I confirmed at live width that the properties they pin actually hold.

Gaps, in payoff order:

1. **The raw-pipe oracle is the code under test** (I1). This is the single highest-value fix in the review: it is why two prior boundaries reported 0% on a 2.2% defect. Every other gap below is downstream of it.
2. **No multi-word fixture** (I2), so C1 is unreachable from the fake — the fixture-backed suite cannot model a shape of the dependency's output that it has never seen.
3. **No test asserts the head stays small.** Both C1 and C2 manifest as "the definition became head tokens." A one-line property — a parsed head over ~8 tokens is a parse failure — fires on 136 live entries at HEAD and would have caught both classes without any pronunciation-specific reasoning.
4. **No test asserts a block has senses.** 37 live entries have a block with zero senses; 24 of those are faithful (`behalf`, `cahoot`, `inasmuch` genuinely have no gloss outside PHRASES), 13 were bugs. Worth a golden with the known-faithful set allowlisted.
5. `realDeps` and `main` at 0% — correct, not worth chasing.

## 6. Architectural notes

- **ARCH-DRY — flag.** The raw-pipe check is written twice (`render_test.go:208`, `live_property_test.go:57`) and both copies inherit the same blind oracle; extract one helper and give it an independent predicate. Residual: `posAt` re-inlines the whitespace test beside `isBoundary`. Positives confirmed: `alnum`/`subsequenceGap` shared across all three widths, `rewritePronunciations` one function with two call sites (`parse.go:176`, `render.go:123`), `splitFirstToken` consolidated. `capture.py`'s duplication of the cgo call remains correct — deliberate, one-directional, documented.
- **ARCH-PURE — pass.** `ParseEntry`, `Render`, `isPronunciation`, `alnum`, `subsequenceGap` are pure string→value functions tested on literals with no IO; `isTerminal` is parked at the boundary (`main.go:62`) and I confirmed zero ANSI escapes on a non-TTY writer; cgo is confined to `dict_darwin.go`. No "pure" entity needs a mock to run.
- **ARCH-PURPOSE — flag.** Run the shadow-sweep on `isPronunciation`, the single source of truth for "what is a pronunciation." It has four consumers: `findPronunciation` (parser), `rewritePronunciations` (renderer), and **both test oracles**. Because the oracles derive from the source rather than from an independent statement of the property, the source cannot be caught being wrong — the guard and the guarded are the same claim. That is the structural reason a 2.2% defect reads as 0%, and it is worth a `workshop/lessons.md` rule of its own: *a property test must not use the function under test as its oracle.*
- **ARCH-MOCK — pass.** Seam, fixture-backed fake from real captures, byte floor, empty-corpus rejection, conformance reading through `Lookup`, documented on-demand cadence with a trigger. Verified live that the real dependency is case-insensitive and the fake models it. The one note is I2: the fake models only single-word headword output, so a shape of the dependency's real behavior is unmodeled — that is a corpus gap, not a seam defect.
- **For M2:** the head model is about to get its second consumer. Fix C2's bound *before* that, so `Entry.IPA` is trustworthy when audio code reads it. And note that `AudioCandidates` derives from the *word*, not from `Entry.IPA`, so C1/C2 don't propagate into the CDN path — but a user seeing a collapsed entry plus correct audio will report it as an audio bug.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000001-define-plan.md`:

1. **Rule B is incomplete and is specification, so it must be corrected in Chunk 1** (lines ~103–118). The stated rule — "a `|…|` span is a pronunciation iff every comma-separated part of its trimmed content is a single token" — is false for multi-word headwords. Extend the verdict table with the rows that break it (`ˈhät ˌdäɡ`, `ˌā prīˈôrī`, `ˌän bəˈhaf əv, ˌôn bəˈhaf əv` → pronunciation) and state the stress-mark discriminator, with the measurement that backs it (1,025 accept / 9,697 reject over the live dictionary).
2. **Record the bound on `findPronunciation`.** Chunk 1 does not say where the entry pronunciation may legally appear; C2 is the cost of that omission. State that it must lie in the head region, before the first block opener or section word.
3. **Task 1's inline `capture.sh` is stale** (lines ~205–225). It reproduces the original 9-word list (`words=(sycophantic quokka … set)`) while the real script carries 25. Round 2's prose says "corpus grew 9 → 25" but the script block was never updated, so the plan again documents something the code contradicts — the same defect as round 2's C2. Replace the inline block with a pointer to the file, or re-sync it, and add the I2 words.
4. **The Integration table will repeat a fixed mistake in M2** (lines ~142-143): `fakeCDN` → `cmd/define/fetch_fake.go` and `fakePlayer` → `cmd/define/player_fake.go` are non-test files. `fakeDictionary` was moved to `dict_fake_test.go` precisely so it would not link into the shipped binary; correct these two rows to `_test.go` before Tasks 8 and 9 are executed.
5. **Amend round 2's closing claim.** It ends "All Critical and Important findings are addressed in this milestone." C1 and C2 here are surviving edges of that entry's own pronunciation-rule work; note them rather than leaving a claim the code contradicts — this is now the second round in which that sentence was written and then falsified.

For `workshop/issues/000001-define.md`: the estimate-quality INFO carry-forward is resolved — it is recorded in the plan's round-2 Revisions (line 577), so nothing is stranded. The plan-gate ledger's `## Open findings` is empty, so there is nothing deferred from the plan gate for this boundary to pick up.

For `atlas/define.md`: the "0% now" claim under **The invariant** must be re-measured after C1/C2, and the sentence describing what the property is blind to should name the oracle limitation, not just letter-order preservation.

---

## Re-review — 2026-08-20T11:38:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 1 — define — NOAD word lookup with Google-style IPA and spoken pronunciation |
| repo | tools |
| issue file | workshop/issues/000001-define.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 1e8386b80feb8986ae8a2deb3e85d02012c0ee4a^..HEAD |
| command | sdlc milestone-close --issue 1 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-20T11:38:28-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

Round 3's fixes hold, and I confirmed that independently rather than from the commit messages: the live property check reads **0 content loss and 0 stress-marked notation over 2,749 live entries**, `FuzzRenderLosesNothing` survived **30.3M executions** with no crasher, `TestFixturesMatchLiveDictionary` passes against live NOAD, and a phantom-block sweep I wrote from scratch found **0 duplicate-POS blocks and 0 empty blocks** — round 2's `opensBlock` rule is genuinely sound. I also checked the invariant's own one-directional blind spot (it proves nothing is *lost*, not that nothing is *duplicated*) and found zero bloat, so that gap is clean. What blocks the boundary is that `opensBlock` accepts `.`, `)`, `:`, `;` but **not `]`** — so a part-of-speech that follows a bracketed editorial note never opens a block, and the entire block is swallowed into the preceding block's *quoted example*. `define complete` renders one block (`adjective`) and buries the whole verb entry — sense numbering and all — inside a quotation; `pulp`, `blaze`, `beetle` are the same. Full-dictionary sweep: **57 entries hit the rejected-opener shape, 22 of them lose a whole part of speech into an example.** Every existing guard is blind to it by construction — the alnum property (order preserved), `strayStress` (no stress marks involved), the 11-word block golden (no fixture has `] verb`). I applied the one-token fix in a scratch copy and re-verified: all unit tests pass, live property stays 0/0, phantom sweep stays 0/0, and `complete`/`pulp` render both blocks. Second blocker: the plan's Chunk 1 still *specifies* the pronunciation rule round 3 proved false, along with three other code-contradicting entries — all four were round-3 recommendations that did not land, and M2 designs against that chapter.

## 1. Strengths

- **The three-width invariant is real, and it survives adversarial probing.** I re-ran it (2,749 live entries, 0 loss) and then attacked its known blind spot: rendered alnum length never exceeded raw across the whole sample, so nothing is silently duplicated either. `alnum`/`subsequenceGap` shared across fixtures, fuzz, and live (`invariant_test.go:42,55`) is the right ARCH-DRY shape.
- **Round 2's block-opener rule is confirmed-good ground — don't re-litigate it.** `opensBlock` (`parse.go:733`) retired the phantom-block family: 3,547 blocks across 2,749 entries, zero duplicate-POS, zero senseless blocks. The bug below is a *missing* member of its accept set, not a flaw in the rule.
- **`strayStress` (`invariant_test.go:40`) is the correct response to round 3's I1** — an oracle resting on a fact about the notation instead of on the parser. It reads a true 0 for what it measures.
- **ARCH-MOCK closure held up under inspection.** `dict_conformance_test.go:28` reads through `fake.Lookup` rather than around it, and the live fixture-conformance check passes byte-for-byte.
- **Bookkeeping is clean and I verified each claim:** 30/30 M1 checkboxes ticked, `go mod tidy` a no-op, no build artifacts anywhere in history, 91.2% coverage, `go vet` clean, `GOOS=linux CGO_ENABLED=0` build **and** vet green via the stub, and CLI edge cases (empty arg, unknown flag, 5000-char word, emoji, two args, `-h`) all exit sanely with no panic.

## 2. Critical findings

### C1 — `opensBlock` rejects `]`, so a part-of-speech after a bracketed note never opens a block and the whole block is swallowed into a quoted example
`cmd/define/parse.go:738`

The accept set is `'.', ')', ':', ';'`. NOAD ends editorial notes with `]` and then opens the next part-of-speech directly:

```
… his range of skills made him the complete footballer | these articles are for the
compleat mathematician. [the spelling compleat is a revival of the 17th century use
as in Walton's The Compleat Angler] verb [with object] 1 finish making or doing: …
```

`verb` is preceded by `]`, so `opensBlock` returns false, no mark is recorded, and everything after it stays inside the adjective block's sub-sense. Verified live at HEAD — `define complete` renders exactly one block heading (`adjective`), and the verb entry appears *inside quotation marks*:

```
      • (also compleat) skilled at every aspect of a particular activity; consummate
        "his range of skills made him the complete footballer"
        "these articles are for the compleat mathematician. [the spelling compleat …
         Angler] verb [with object] 1 finish making or doing: he completed his Ph.D. in 1983"
      …
        "he completed 12 of 16 passes for 128 yards. 2 make (something) whole or perfect: …"
```

Verb sense 2 is inside a quoted string. `define pulp` is identical (only `noun` renders).

Prevalence, full sweep over all **71,427** reachable entries: **57** entries contain a strong-shaped POS token at depth 0 that `opensBlock` rejects solely because its predecessor is `]`; **22** of them lose a part of speech into a quoted example (`complete`, `pulp`, `blaze`, …). This is not the ambiguity the atlas accepts under *Limits* — that clause is about NOAD *not writing* punctuation (`parrot`). Here NOAD writes a delimiter and the accept set omits it.

Why nothing caught it: the alnum property preserves order; `strayStress` sees no stress marks; `TestCorpusBlockStructure` (`render_test.go:120`) samples 11 fixtures, none with `] verb`.

Fix — one token:

```go
case '.', ')', ':', ';', ']':
```

I applied exactly this in a scratch copy and re-measured: `go test ./...` passes (including the block golden and all 29 fixtures), live property stays **0 loss / 0 stray notation**, the phantom sweep stays **0 dupPOS / 0 empty blocks**, example-swallows drop **54 → 32**, and `complete` and `pulp` both render their verb blocks. Add `complete` to `capture.sh` and a `{"complete": {"adjective","verb"}}` row to `TestCorpusBlockStructure` — this is the fourth boundary at which this family has arrived undetected, and the corpus still has no entry that exercises it.

### C2 — Plan Chunk 1 still specifies the pronunciation rule the code replaced, plus three further code-contradicting entries; all four were round-3 recommendations that did not land
`workshop/plans/000001-define-plan.md:86`, `:130`, `:155`, `:158`, `:201`

Chunk 1 is specification — it is what M2 will be designed against — and it currently contradicts the code in four places:

| Plan says | Code has |
|---|---|
| `:86` "A `\|…\|` span is a **pronunciation** iff every comma-separated part of its trimmed content is a single token" | The stress-mark rule (`parse.go:111-155`): a multi-word span qualifies when a word carries `ˈ`/`ˌ`, capped at `maxPronunciationWords`. The plan's rule **is** the one that shipped round 3's C1 (`define "hot dog"` → `/ˈhätˌdäɡər/`) |
| `:130` `Sense … Examples []string` | `Examples []Example` (`parse.go:85`), where `Example{Label, Text}` (`parse.go:95`) is a **new pure entity with no row in the Core-concepts table** |
| `:155`, `:158` `fakeCDN` → `fetch_fake.go`, `fakePlayer` → `player_fake.go` | Non-test files. `fakeDictionary` was moved to `dict_fake_test.go` precisely so fakes don't link into the shipped binary; Tasks 8–9 will re-make the fixed mistake |
| `:201` Task 1's inline `capture.sh` shows `words=(sycophantic quokka … set)` — 9 words | 29 fixtures on disk |

Round 3's plan-revision recommendations #1, #3 and #4 named the first, fourth and third of these; none was applied, and round 2's C2 was this same defect one level up. Fix Chunk 1 in place (the Rule B block and its verdict table, the `Sense` bullet, an `Example` table row, the two fake paths), and replace Task 1's inline script with a pointer to the file so it cannot drift again.

## 3. Important findings

### I1 — The seam does not read NOAD; it reads whatever dictionaries are active, and every artifact claims otherwise
`cmd/define/dict_darwin.go:44`, `atlas/define.md:1-8`, `README.md:25`, issue `## Spec`

`DCSCopyTextDefinition(NULL, s, r)` (`dict_darwin.go:31`) passes a NULL `DCSDictionaryRef`, which means *search all active dictionaries*. The code comment says "noadDictionary reads the New Oxford American Dictionary bundled with macOS — the same dictionary Google licenses … which is why the notation matches character-for-character." That guarantee does not hold.

Measured over the full 71,427 reachable entries: **530 contain Han script** — they come from a Chinese dictionary, not NOAD. `define anda` prints `谙达 āndá 动 （对人情世故等）熟悉通达。…`; `define Ao` likewise. The English case is present too: `iPhone`, `iPad` and `MacBook` — three of the 29 committed fixtures — are Apple Dictionary entries ("A line of notebook computers from Apple that was discontinued in 2019"), not NOAD. The `splitHeadByShape` no-pronunciation path (`parse.go:186`) was built for a shape that belongs to a *different dictionary* than the one the design documents.

I checked whether this is fixable in code: the SDK header exports only `DCSGetTermRangeInString` and `DCSCopyTextDefinition`, and `DCSDictionaryRef` has no public constructor — so passing NULL is forced and the behavior is inherent. That is why I rate this Important rather than Critical: the deliverable is a documentation correction, not a code change. State in `atlas/define.md` *Limits*, `README.md`, and the `noadDictionary` doc comment that results come from the host's active dictionary set, NOAD-first in the common case; note that the conformance tests' pass/fail therefore depends on undocumented host dictionary configuration; and mark the three Apple Dictionary fixtures as such in `capture.sh`.

### I2 — 2.0% of entries still render raw NOAD `|` delimiters, and the atlas publishes "0%"
`atlas/define.md:87`, `cmd/define/render.go:116`

The atlas states: "Independently measured, unconverted notation went 2.2% → **0%** over 2749 live entries." `strayStress` only sees notation carrying a stress mark. Example-separator pipes carry none, so they are invisible to it.

Full-sweep measurement with a one-line oracle that consults nothing (`strings.Contains(out, "|")`): **1,461 of 71,427 entries (2.0%)** render at least one raw `|` — 1,370 in section text, 92 in a sense gloss, 0 in examples. Verified live, `define bargainer`:

```
  PHRASES
    drive a hard bargain … into the bargain (North American English in the bargain) in
    addition to what was expected; moreover: they've exceeded expectations and played some
    great football into the bargain | save yourself money and keep warm and cozy in the bargain.
```

Round 3's lesson was that a narrow oracle published a false 0. The oracle is honest now but the *claim* is broader than the measurement. Fix: re-word the atlas to say what was measured ("0% stress-marked notation"), and add the pipe check to `TestNoRawPronunciationNotationSurvives` and the live sweep — it is one line and independent of every function under test. Separately, section text is rendered as one undifferentiated paragraph (`render.go:116-121`); splitting it on the same example-separator rule the sense path already uses would remove the pipes at the source.

### I3 — The register-label opener family is larger than *Limits* implies, and 32 entries still lose a block after C1's fix
`cmd/define/parse.go:733`, `atlas/define.md` *Limits*

`opensBlock` also rejects an opener preceded by a register or domain label rather than punctuation — `mainly British English verb`, `rare adjective`, `vulgar slang noun`, `see Coleoptera verb`, `chemical formula: CHCl3 verb`. Two distinct severities, both verified live:

- **Block swallowed** (second or later block): after C1's fix, **32 of 71,427** entries still show a POS-plus-grammar-label inside a *quoted example*. Confirmed by hand: `define shuttle` and `define chloroform` each render only `noun`; the verb block is inside a quotation.
- **First block unlabeled**: `define backheel` and `define Barmecide` render a leading block with no POS heading whose gloss opens "mainly British English verb [with object] kick (something)…". By the same signature this shape reaches ~597 entries; I confirmed two by hand, so treat that number as an upper bound.

The atlas's *Limits* entry covers this in principle ("NOAD does not always write one — `parrot` …"), but "some block boundaries" reads far smaller than 600+, and the swallow-into-example variant is materially worse than `parrot`'s nesting. At minimum, quantify it in the atlas. The register-label sub-case also looks tractable: `isGrammarLabelOnly` (`parse.go:376`) already models "text that may sit between a POS and its pronunciation"; a mirror predicate for a short leading register label would fix `backheel`/`Barmecide` without loosening `opensBlock` generally.

### I4 — `TestRenderGluedPOSNotPrintedTwice` asserts nothing
`cmd/define/render_test.go:36`

Both checks are dead:

```go
if n := strings.Count(out, "noun"); n < 1 { t.Fatal(...) }        // "record" always contains "noun"
for _, line := range strings.Split(out, "\n") {
    if strings.TrimSpace(line) == "" && strings.HasPrefix(line, "  ") { t.Error(...) }
}
```

The second condition requires a line that is entirely whitespace *and* starts with two spaces. `Render` writes blank lines as a bare `"\n"` (`render.go:70`) and never emits an all-space line — every indented write is guarded by a non-empty payload — so the loop body is unreachable. This is the named guard for `FromHead` suppression, which was a Critical in round 1. Replace it with a positive assertion: the head line contains `noun`, and no block heading line equals `noun` for `record`.

## 4. Minor findings

- `parse.go:581` `strings.TrimRight(seg, ".")` strips *all* trailing periods, so an example ending in an abbreviation or ellipsis loses them (`…in Washington, D.C.` → `D.C`). Invisible to the invariant, which excludes punctuation by design.
- `invariant_test.go:44` `rest[lo:hi]` byte-slices a string that contains multibyte runes; the failure message can carry invalid UTF-8. Test-only.
- `invariant_test.go:127` `len([]rune(alnum(raw)))` — `alnum` already returns `[]rune`; the conversion is a no-op. Open since round 2.
- `main.go:49-53` collapses `ErrNoEntry`, `ErrLookupFailed` and the non-darwin stub error to exit 1, while `README.md:31` documents 1 as "no dictionary entry". Open since round 2.
- `README.md:52` says `go test -tags conformance ./...`; `atlas/define.md:112` says `./cmd/define/`. Open since round 3.
- `atlas/define.md` heading "## Three parsing rules worth knowing" is followed by "Two shapes are non-obvious" and then three numbered items.
- `atlas/define.md:56` points at `testdata/fuzz/` for minimized crashers; the directory does not exist. Accurate today (30.3M execs found none) but worth a word.
- `live_property_test.go:24` `liveSampleSize = 9000` samples 2,749 of 71,427 (3.8%). My full sweeps took ~34 s each; the `checked < 500` guard would still protect it. The stride sample did surface `complete`, so this is not why C1 shipped — but the 26× width is nearly free.
- `parse.go:214` still cannot segment a multiword glued POS: `define bases` renders the head as `bases ba·sesplural  noun 1`. Open since round 2.

## 5. Test coverage notes

91.2% of statements, `go vet` clean, darwin and `GOOS=linux CGO_ENABLED=0` build **and** vet green, live fixture conformance passing, fuzz clean at 30.3M executions. `TestRenderColorOnlyWhenAsked`'s strip-and-compare (`render_test.go:60`), `TestLoadFakeDictionaryRejectsEmptyCorpus`, and `TestEveryFixtureIsReachableViaLookup` are real assertions on real logic.

Gaps, in payoff order:

1. **No golden exercises a POS opener after `]`** — C1's hiding place. `TestCorpusBlockStructure` covers 11 words; none has the shape.
2. **The raw-notation checks measure only stress-marked notation** (I2). A `strings.Contains(out, "|")` assertion is one line, needs no knowledge of the parser, and closes the residual.
3. **`TestRenderGluedPOSNotPrintedTwice` is a no-op** (I4) — the one test named for a round-1 Critical.
4. **Nothing asserts a block is not swallowed.** The signature I used — a POS word followed by `" ["` or `" ("` inside a `Sense.Gloss` or `Example.Text` — found C1 and I3 in one pass and is cheap enough to run over the fixture corpus as a golden.
5. **No fixture comes with a documented non-NOAD provenance** (I1), so nothing pins which of the 29 are NOAD and which are Apple Dictionary.
6. `realDeps` and `main` at 0% — correct, not worth chasing.

## 6. Architectural notes

- **ARCH-DRY — pass, one Minor.** Round 3's duplicate raw-pipe oracle is genuinely consolidated: `strayStress` lives once in `invariant_test.go:40` and is called from both `render_test.go:208` and `live_property_test.go:57`. `alnum`/`subsequenceGap` shared across all three widths; `rewritePronunciations` one function with two call sites (`parse.go:174`, `render.go:127`); `splitFirstToken` still the single token-boundary definition; `capture.py`'s duplication of the cgo call remains correct — deliberate, one-directional, documented at `capture.py:4-7`. Residual: `posAt` (`parse.go:734` region) re-inlines the whitespace test two lines below `isBoundary`.
- **ARCH-PURE — pass.** `ParseEntry`, `Render`, `isPronunciation`, `opensBlock`, `rewritePronunciations`, `alnum`, `subsequenceGap` are pure string→value functions; `TestIsPronunciation` and `TestParseHeader` run on literals with zero IO. `isTerminal` is parked at the boundary (`main.go:62`) and I confirmed zero ANSI escapes on a non-TTY writer. cgo is confined to `dict_darwin.go`. No entity marked PURE needs a mock to run — reading `testdata/` through the fake is idiomatic fixture IO, not mocking.
- **ARCH-PURPOSE — flag.** Run the shadow-sweep on the issue's stated purpose ("reproduce Google's NOAD panel"). Two consumers do not derive from it: the dictionary seam does not actually select NOAD and no artifact says so (I1), and the pronunciation-notation conversion reaches senses and sections' pronunciations but not their example separators, leaving 2.0% of entries mixing notations while the atlas publishes 0% (I2). Both are the pattern of a claim measured by a check narrower than the claim — the same shape as round 3's finding, one level out.
- **ARCH-MOCK — pass, with a documentation flag.** Seam, fixture-backed fake from real captures, byte floor, empty-corpus rejection, conformance reading *through* `Lookup`, on-demand cadence with a stated trigger. The flag is I1: the fake is named and documented for NOAD while the real dependency is the host's active dictionary set, so `TestFixturesMatchLiveDictionary` will report "NOAD drifted" on a host whose active dictionaries differ — a false drift signal the test cannot distinguish from a real one.
- **For M2:** fix C1 before the second `Entry` consumer lands, and correct Chunk 1's Rule B (C2) *first* — `AudioCandidates` derives from the word rather than from `Entry.IPA`, so C1/C2 do not propagate into the audio path, but a user seeing a collapsed entry alongside correct audio will file it as an audio bug. Put `fakeCDN`/`fakePlayer` in `_test.go` files as `fakeDictionary` now is, and give them the reachability assertion `TestEveryFixtureIsReachableViaLookup` provides for the dictionary fake.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000001-define-plan.md`, and this time make the Chunk 1 edits the entry claims:

1. **Rule B is false as written (C2), and it is specification.** Replace the block quote at `:86` with the implemented rule — single-token *or* a multi-word span carrying `ˈ`/`ˌ`, bounded by `maxPronunciationWords` — and extend the verdict table at `:92` with the rows that break the old rule (`ˈhät ˌdäɡ`, `ˌā prīˈôrī`, `ˌän bəˈhaf əv, ˌôn bəˈhaf əv` → pronunciation).
2. **`Example` is an undocumented entity (C2).** Correct the `Sense` bullet at `:130` to `Examples []Example` and add an `Example` row to the Pure-entities table at `:101`, noting it mirrors `Block.Label` one level down.
3. **Fake file paths (C2).** `:155` and `:158` must read `cmd/define/fetch_fake_test.go` and `cmd/define/player_fake_test.go`, matching Task 8/Task 9's Files lines at `:378` and `:389`. Round 3 recommended this and it did not land; Tasks 8–9 are next.
4. **Task 1's inline `capture.sh` (`:201`) is stale** — 9 words versus 29 on disk. Replace the code block with a pointer to the file so it cannot drift a fourth time.
5. **Record the `]` opener rule (C1)** alongside the `opensBlock` description, so M2 and any future block-boundary work inherit the corrected accept set rather than rediscovering it.
6. **Amend round 3's closing framing.** The round-3 entry documents `parrot` as the accepted block-boundary ambiguity; C1 is a *punctuated* delimiter that was simply omitted, and I3 shows the accepted class reaches ~600 entries. Distinguish the two so the limitation is not read as covering both.

For `atlas/define.md`: re-word the "0%" claim under **The invariant** to state what `strayStress` measures (I2), quantify the *Limits* block-boundary entry (I3), and add the active-dictionary limitation (I1). For `workshop/issues/000001-define.md`: the `## Spec`'s "NOAD only" should note that the local seam cannot select NOAD specifically. The plan-gate ledger's `## Open findings` is empty, so nothing was deferred from the plan gate for this boundary to pick up.
