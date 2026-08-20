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
