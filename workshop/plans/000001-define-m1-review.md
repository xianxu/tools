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
