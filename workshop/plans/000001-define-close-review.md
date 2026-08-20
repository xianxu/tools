# Boundary Review — tools#1 (whole-issue close)

| field | value |
|-------|-------|
| issue | 1 — define — NOAD word lookup with Google-style IPA and spoken pronunciation |
| repo | tools |
| issue file | workshop/issues/000001-define.md |
| boundary | whole-issue close |
| milestone | — |
| window | b1d312830a59cd9a8ed6589e9b68a4556ba31b04..HEAD |
| command | sdlc close --issue 1 |
| reviewer | claude |
| timestamp | 2026-08-20T11:58:42-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

I verified this boundary live rather than from the commit log: unsandboxed, `TestFixturesMatchLiveDictionary` passes byte-for-byte, the repo's live property check reads **0 content loss / 0 raw notation over 2,728 entries**, the fuzz target survived **4.9M executions**, `go vet` and `GOOS=linux CGO_ENABLED=0 build && vet` are green, and I confirmed the tool's actual purpose at 8× the repo's own width — of **21,640 live entries where NOAD supplies a pronunciation, all 21,640 render a `/…/` span**. That is real, confirmed-good ground. What blocks the close is a structural blind spot shared by all three invariant widths: `subsequenceGap`, `strayStress`, and the bare `|` check each detect **removal or leftovers**, and none detects **insertion**. Two correctness bugs live in that gap. `Render` prints examples with `%q`, so Go's string escaping reaches the screen — `define blow` shows `"\"This blows,\" she sighs"` and `define otherwhere` shows `other\u00adwhere`, the latter injecting five alphanumeric runes NOAD never returned. And `parseSenses` has no paren-depth guard, so NOAD's routine cross-reference `(sense 1 of the noun)` manufactures a sense number and cuts the gloss mid-parenthetical: `define bashaw` renders `another term for pasha (sense` / `1. of the noun)`. I measured both across 23,632 live entries (0.055% and 0.18%, ≈40 and ≈130 entries dictionary-wide), confirmed all 43 instances of the second against the raw text, and reduced each to a three-line offline repro. Both fixes are small and local — `%q` → explicit quoting, and a depth counter that `parseBlocks` already implements for the identical rule. Everything else is bounded documentation drift.

## 1. Strengths

- **The purpose is delivered, and I measured it with an oracle the repo doesn't run.** 21,640/21,640 live entries carrying a NOAD pronunciation render a `/…/` span — zero silent IPA drops. The round-3 `headLimit` bound (`parse.go`) degrades cleanly rather than adopting a derivative's pronunciation, exactly as designed.
- **The colour path is genuinely presentation-only, across the whole corpus.** `render_test.go:60` proves it for one fixture; I ran strip-and-compare over all 29 and it holds, and the alnum invariant holds on the colour path too. Colour is not a second renderer.
- **`opensBlock` (`parse.go:733`) is sound.** Round 4's one-token `]` addition closed the `complete`/`pulp` family; my independent sweep found block-swallowing at 0.051% of 23,632 entries, matching the atlas's documented `~32/71,427` Limits figure. The documented limitation is honestly quantified.
- **The two-oracle notation check works.** `strayStress` + `IndexByte(out, '|')` (`invariant_test.go:40`, `live_property_test.go:57`) read a true 0 over the sampled width; the residual I found at 8× width is a *symptom of C2*, not an oracle failure.
- **The fake/seam discipline is real.** Every fake is in a `_test.go` file, `dict_conformance_test.go:28` reads *through* `fake.Lookup` rather than around it, and `TestEveryFixtureIsReachableViaLookup` pins the fake's own contract. `make install` resolves against the inherited `build` target; nothing untracked; `go mod tidy` a no-op.

## 2. Critical findings

### C1 — `Render` prints examples with `%q`, so Go escape sequences reach the screen and alphanumeric runes are *inserted*
`cmd/define/render.go:111`

```go
fmt.Fprintf(&b, "%s%q%s\n", p.ex, prettyPronunciations(ex.Text, p), p.off)
```

`%q` is `strconv.Quote`. It escapes `"` and every rune failing `unicode.IsPrint` — and NOAD examples routinely contain both quoted speech and U+00AD soft hyphens.

Offline repro, no dictionary needed:

```go
Render(ParseEntry(`blow | blō | verb 1 make a sound: "This blows," she sighs.`), RenderOpts{})
//   1. make a sound
//     "\"This blows,\" she sighs"          <- literal backslashes on screen

Render(ParseEntry("otherwhere | ˈəT͟H | adverb elsewhere: a secret other\u00adwhere."), RenderOpts{})
//     "a secret other\u00adwhere"          <- 5 alnum runes NOAD never returned
```

Live-confirmed on real entries: `define blow`, `define briskly`, `define glowingly`, `define otherwhere`. Measured **13 of 23,632** sampled live entries (0.055%, ≈40 dictionary-wide); the `\"` variant dominates, the soft-hyphen variant is what also inflates alnum.

This is drift from a contract the code states about itself — `render.go:28` says Render "must not change case, abbreviate, truncate, or reorder" as "a CORRECTNESS CONSTRAINT, not a style preference" — and from the issue Spec's framing that "words may not vanish or move." A word gaining `\u00ad` in its middle does both. It survived five review rounds because every oracle is one-directional.

Fix: quote explicitly instead of delegating to `%q`.

```go
fmt.Fprintf(&b, "%s%q%s\n", ...)                  // before
fmt.Fprintf(&b, "%s\"%s\"%s\n", p.ex, prettyPronunciations(ex.Text, p), p.off)  // after
```

Then add the insertion oracle in §5 — I verified `len(alnum(out)) == len(alnum(raw))` holds **exactly** on all 29 fixtures and on 23,630/23,632 live entries, failing precisely on this bug.

### C2 — `parseSenses` splits on numerals inside parentheses, manufacturing a sense from NOAD's `(sense 1 of the noun)` cross-reference
`cmd/define/parse.go:487` (`parseSenses`), regex at `parse.go:480`

`senseSplit` scans the block text with no delimiter-depth tracking, and the sequence guard accepts a numeral that *opens* a sequence — so a lone prose `1` is always accepted. `parseBlocks` (`parse.go:~267`) already implements the exact rule this needs, counting `paren` and `bracket` depth so a POS word inside a delimiter cannot open a block. That rule was never applied one level down (ARCH-DRY).

Offline repro:

```go
ParseEntry(`bashaw ba·shaw | bəˈSHô | noun another term for pasha (sense 1 of the noun) ORIGIN mid 16th century.`)
// renders:
//   noun
//     another term for pasha (sense
//     1. of the noun)
```

The gloss is cut mid-parenthetical, the paren is left unbalanced across two lines, and a fake sense number appears. Live-confirmed on `assign`, `aureole`, `bashaw`, `bonito`, `channelize`, `corm`, `crummiest`, `dendron`, `endoplasm`, `exothermal`, `flittermouse`, `hough`, `landslip`, `lentoid`, `loxodromic` — and on the non-cross-reference variants `born` ("on January **1** 1992"), `deuterium` ("present to about **1** part in 6,000"), `kilocalorie` ("equal to **1** large calorie"), `liveborn` ("affects **1** in 3,600").

Measured: I flagged every block whose only numbered sense is `1` following a lead sense, then checked each against the **raw** text to see whether that numeral follows a sentence end. **43 of 43 confirmed manufactured**, out of 23,632 sampled (0.18%, ≈130 dictionary-wide). `TestCorpusSenseNumbersAreSequential` (`render_test.go:96`) cannot see this — a sequence of exactly `1` is both sequential and correctly anchored.

`define born` is the worst case, because the false split also lets a raw delimiter escape:

```
      • (be born) (of an organization, movement, or idea) be brought into existence
        "on January"
    1. 1992 the new company was born | the sound bite was born in the TV newsroom.
```

The text after an accepted split with no `:` becomes a *gloss*, and gloss text is never split on `|`. That is the entire residual my wide sweep found (5 of 23,632 entries render a raw `|`) — one root cause, two symptoms. Fixing C2 removes both.

Fix: track paren/bracket depth in `parseSenses` and reject any split at depth > 0, mirroring `parseBlocks`. Consider also requiring that an accepted numeral follow a sentence end or open the block — that additionally rejects `on January 1 1992`.

## 3. Important findings

### I1 — `atlas/define.md` states the pronunciation rule the plan records as having shipped a Critical
`atlas/define.md:60` (parsing rule 3)

> A span is a pronunciation iff every comma-separated part is a single space-free token (`isPronunciation`).

That is the single-token-only rule. `isPronunciation` (`parse.go:135`) also accepts a short multi-word span carrying a stress mark, and the plan says so explicitly at `workshop/plans/000001-define-plan.md:102`: *"The single-token-only rule was wrong and shipped a Critical… `define "hot dog"` displayed `/ˈhätˌdäɡər/`."* The plan was corrected; the atlas — the current-state map agents navigate by, per AGENTS.md §8 — still publishes the superseded rule. Anyone deriving from it reintroduces the bug.

### I2 — `define -h` still claims NOAD-only, the one claim round 4 retracted everywhere else
`cmd/define/main.go:47`

```
"Looks the word up in the New Oxford American Dictionary bundled with
macOS — the same dictionary Google licenses — and plays its recorded
pronunciation."
```

The issue `## Log` says the NOAD-only claim was "Corrected in all four" artifacts (code comment, atlas, README, Spec). The CLI help text is a fifth surface carrying it, and it is the most user-facing one. One string.

### I3 — README is stale for surface this very window shipped
`README.md:52`, `README.md:31-35`

- `` `bin/` is not on `$PATH` by default; a `make install` target lands with M2. `` — M2 landed in this window and `Makefile.local` ships `install`. I confirmed `make -n install` resolves. The Build block should document it.
- The `define` usage block documents `-raw` and `-no-color` but not `-no-audio`, `-times N`, or `-locale` — three new user-facing flags from M2. This is the README docs gate.

### I4 — the `Player`/`afplay` seam has a fake but no live conformance check, and zero coverage
`cmd/define/player.go:31`, `atlas/define.md:170`

`afplayPlayer.Play` is **0.0%** covered and no conformance test touches `afplay`. `Dictionary` and `AudioSource` each have one; the third seam does not, and the atlas's Conformance section doesn't mention it. The dependency assumption that matters is that `afplay` **blocks until playback completes** — that is what makes `playN`'s 250 ms gap and the "3×" acceptance criterion meaningful. If it ever returned immediately, three overlapping sounds would still pass every test in the repo (ARCH-MOCK: missing live conformance for behavior we depend on). A conformance-tagged test that plays a short fixture and asserts wall-clock ≥ duration would close it.

### I5 — a transport failure is reported as "no recorded pronunciation", with the cause dropped from the error chain
`cmd/define/fetch.go:67`

```go
return nil, "", fmt.Errorf("%w: %v", ErrNoAudio, firstErr)
```

`%v`, not `%w`, so `errors.Is(err, context.Canceled)` is false — I verified this against the fake CDN. A caller cannot distinguish "the CDN has no recording for this word" (a normal outcome) from "the network is down". The text carries the cause, so it isn't silent, but the sentinel is wrong. The codebase states the opposite principle one seam over, in `dict_darwin.go:15`: *"status: 0 = found, 1 = no entry, 2 = internal failure. Collapsing these into a bare NULL would report a genuine CoreFoundation failure as 'no entry'."* Same discipline, opposite implementation. Fix: `%w` for both, or a distinct `ErrFetchFailed`. (`player.go:34` uses the same `%w`/`%v` shape but there `ErrNoPlayer` *is* the accurate sentinel — leave it.)

### I6 — the plan has no Revisions entry for round 4 or M2, and its latest entry publishes a figure round 4 disproved
`workshop/plans/000001-define-plan.md:580` (last entry)

The plan logs a Revisions entry per review round for rounds 1–3, then stops. Round 4 (2 Criticals) and all of M2 are recorded only in the issue `## Log`. Two concrete consequences:

- The round-3 entry closes with *"Live measurements after round 3: 0 lost content, **0 unconverted notation**"* — round 4 found 2.0% of entries still rendering raw `|` under a broader oracle. The durable plan's most recent claim is one the next commit falsified, with nothing after it correcting the record. That is the third round in a row this pattern has been flagged.
- The plan's only normative statement of the block-opener rule (`plan.md:537`, a block quote — i.e. specification) reads *"immediately after a sentence end (`.`, `)`, `:`, `;`)"* and **omits `]`**, which `opensBlock` now accepts. Round 4's review recommended recording it; it wasn't.

The Core-concepts tables themselves are accurate — I verified every entity in both the Pure and Integration tables exists at its stated path, including the `_test.go` corrections for `fakeCDN`/`fakePlayer`.

## 4. Minor findings

- `atlas/define.md:26` — Seam table declares four columns (`Seam | Wraps | Fake | Status`) but every row has three cells; `Status` renders empty.
- `atlas/define.md:41` — heading says "Three parsing rules worth knowing", the sentence under it says "Two shapes are non-obvious", three items follow. Open since round 3.
- `url.PathEscape("..")` is a no-op, so `define ..word` puts a literal `..` segment in the CDN path (`audiourl.go:40`). Harmless GET to a fixed host, but the shard should be escaped as a path *segment*.
- No overall deadline on the audio phase: four candidates × a 10 s per-request timeout means up to ~40 s of hang *after* the definition has printed. A `context.WithTimeout` in `run()` bounds it.
- `Fetch` doesn't check `ctx.Err()` between candidates, so a cancelled context still walks all four (each failing instantly).
- A 200 with an empty or non-audio body is accepted as a recording — verified: exit 0, `♫ playing 3×` announced, silence, no diagnostic. `fetch_conformance_test.go` already knows the shape of a valid recording (≥1000 bytes, ID3/frame-sync); production checks neither.
- `fetch_conformance_test.go:66` indexes `data[0]`/`data[1]` after a non-fatal `len(data) < 1000` check — panics on a short body instead of failing.
- `player.go:32` runs `exec.LookPath("afplay")` on every play, three times per invocation.
- `-locale` is unvalidated though documented "us or gb"; a typo silently degrades to "no recorded pronunciation".
- README documents exit 1 as "no dictionary entry", but `ErrLookupFailed` (a genuine CoreFoundation malfunction) also exits 1. Open since round 2.
- `render.go:123` — a section whose text begins with `|` yields an empty first segment, so its first visible line gets the 6-space continuation indent instead of 4.
- `invariant_test.go:127` `len([]rune(alnum(raw)))` is a no-op conversion; `alnum` already returns `[]rune`. Open since round 2.
- `atlas/define.md:56` points at `testdata/fuzz/` for minimized crashers; the directory does not exist (accurate today — 4.9M execs found none).
- README says `go test -tags conformance ./...`; the atlas says `./cmd/define/`. Open since round 3.

## 5. Test coverage notes

88.9% of statements, vet clean on darwin and `GOOS=linux CGO_ENABLED=0`, fuzz clean at 4.9M executions, live fixture conformance passing. `TestRunNoAudioMakesNoRequests`, `TestFetchWalksCandidatesInOrderAndStopsAtFirstHit`, and `TestPlayNStopsOnError` are real assertions on real logic — the fake CDN's ordered request log is doing genuine work, not restating the implementation.

The gap that matters is structural, and it is the reason both Criticals survived five rounds:

1. **Every oracle is one-directional.** `subsequenceGap` proves nothing was *removed*; `strayStress` and `IndexByte(out,'|')` prove nothing was *left over*. Nothing proves nothing was *added*. Add the bound — I verified it holds exactly on all 29 fixtures and on 23,630/23,632 live entries, failing only on C1:
   ```go
   if a, b := len(alnum(raw)), len(alnum(out)); a != b {
       t.Errorf("render changed alnum count: raw=%d rendered=%d", a, b)
   }
   ```
   Combined with the existing subsequence check this upgrades the invariant from "nothing lost" to "exact ordered multiset" — a materially stronger property for one line. Add a companion escape oracle (`strings.Contains(out, "\\\"")`) since the `\"` variant adds no alnum runes.
2. **No structural assertion covers a numeral inside a delimiter** (C2). A golden with `bashaw` or `aureole` asserting zero numbered senses would pin it; neither is in the 29-fixture corpus.
3. **The legacy-path fallback is untested end-to-end.** `newAudioRig` only ever populates `AudioCandidates(word)[0]`, so the composition of the 4-element list with `Fetch`'s walk is exercised at index 0 only — despite that fallback being the documented reason the list exists (`defenestrate` needs `_2` on the legacy path). I wrote the test in a scratch copy and it passes, so this is a coverage gap, not a bug.
4. **`afplayPlayer.Play` at 0% with no conformance check** (I4).
5. `realDeps` and `main` at 0% — correct, not worth chasing.

## 6. Architectural notes

- **ARCH-DRY — flag.** C2 is the finding: `parseBlocks` tracks paren/bracket depth so a token inside a delimiter cannot open a block; `parseSenses` re-derives "where may I split?" without that guard. One rule, two sites, one of them missing it. Secondary: `fetch.go`'s error collapse contradicts `dict_darwin.go`'s explicit malfunction/absence split (I5). Confirmed-good: `alnum`/`subsequenceGap` shared across all three widths, `strayStress` defined once and called from two, `rewritePronunciations` one function with two call sites, `splitFirstToken` still the single token-boundary definition, and `capture.py`'s duplication of the cgo call correctly deliberate and documented.
- **ARCH-PURE — pass.** I proved this rather than inspected it: both my Critical repros run `ParseEntry` + `Render` on string literals with no fixture, no fake, no IO. `isTerminal` is parked at the boundary (`main.go:110`) and produces zero ANSI on a non-TTY writer; cgo is confined to `dict_darwin.go`; `deps` is explicit and injected. The M2 additions hold the line — `AudioCandidates` is pure and offline, and the repeat loop deliberately lives in `playN` rather than behind `Player.Play(n)` so the count is assertable.
- **ARCH-PURPOSE — mostly pass, one flag.** The shadow-sweep on the issue's purpose comes back clean at 8× the repo's width: 21,640/21,640 entries with a pronunciation show it. Both milestones' Done-when items are delivered and test-backed. The flag is I1 — the atlas, which is the artifact every future consumer derives its model of the parser from, restates a rule the code deliberately replaced after that rule shipped a Critical. A hand-maintained restatement that has drifted from the source is a deferred consumer.
- **ARCH-MOCK — flag.** Two of three seams are exemplary: fixture-backed fake from real captures, a byte floor, empty-corpus rejection, conformance reading *through* the seam, on-demand cadence with a stated trigger. The third (`Player`/`afplay`) has the fake but no live conformance check and no coverage at all (I4), leaving the one behavioral assumption the headline feature rests on — that `afplay` blocks — unasserted anywhere.
- **For future work in this repo:** the insertion oracle in §5 is the durable lesson. The atlas already carries the "an honest oracle can still be a narrow one" note from round 3; the generalization it hasn't yet made is *direction* — a subsequence check is structurally incapable of seeing interpolation, so a fidelity claim needs both bounds. That belongs in `workshop/lessons.md` alongside the existing "a check must not consult the function it is checking" rule. Secondarily: when `internal/` is finally earned by a second consumer, `ParseEntry`/`Render`/`AudioCandidates` are the stable surface and none of them needs to move.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000001-define-plan.md`:

1. **Round 4 + M2 has no entry at all** (I6). Rounds 1–3 each have one; the convention stops before the largest commit in the window. Record round 4's two Criticals (the NULL `DCSDictionaryRef` / not-NOAD-only correction, and `opensBlock` accepting `]`) and M2's Tasks 7–10 as delivered.
2. **Correct the round-3 closing measurement.** That entry states "0 unconverted notation"; round 4 measured 2.0% under a broader oracle and added the second check. Leaving it unamended is the third instance of the pattern earlier rounds flagged.
3. **The `opensBlock` rule block-quote at `:537` is specification and contradicts the code** — it lists `.`, `)`, `:`, `;` and omits `]`. Add `]` with the `complete`/`pulp` rationale, so future block-boundary work inherits the corrected accept set.
4. **Record the delimiter-depth rule for sense splitting** (C2) next to the block-opener rule, since it is the same rule one level down and its absence is a shipped Critical.
5. The Core-concepts tables need no change — every Pure and Integration row was verified against the filesystem, including the `fetch_fake_test.go` / `player_fake_test.go` corrections.

For `atlas/define.md`: replace parsing rule 3 with the implemented stress-mark rule (I1), fix the Seam table's column count and the "Three rules / Two shapes" mismatch, and — after C1 and C2 are fixed — re-measure before restating the notation figure.

For `workshop/issues/000001-define.md`: the estimate-quality INFO carry-forward is resolved (recorded in the plan's round-2 Revisions), and `000001-define-plan-gate.md`'s `## Open findings` is empty with all six PQ items disposed `addressed` — nothing was deferred from the plan gate for this boundary to pick up.

---

## Re-review — 2026-08-20T12:18:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 1 — define — NOAD word lookup with Google-style IPA and spoken pronunciation |
| repo | tools |
| issue file | workshop/issues/000001-define.md |
| boundary | whole-issue close |
| milestone | — |
| window | b1d312830a59cd9a8ed6589e9b68a4556ba31b04..HEAD |
| command | sdlc close --issue 1 |
| reviewer | claude |
| timestamp | 2026-08-20T12:18:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

I verified this boundary live rather than from the commit log, and at 25× the width the repo's own checks run. Both prior Criticals are genuinely fixed and I confirmed it at **full dictionary width — 70,897 live entries, 0 content loss and 0 insertion**, which is the strongest evidence this issue has produced: the `%q` escaping is gone and the `(sense 1 of the noun)` cross-reference no longer manufactures a sense. All three conformance checks pass live (including the new `afplay`-blocks check that closes the last ARCH-MOCK gap), `go vet` and `GOOS=linux CGO_ENABLED=0 build && vet` are green, `go mod tidy` is a no-op, the tree carries no build artifacts, and every entity in the plan's Core-concepts tables exists at its stated path. What I found is a residual of the same family, one notch narrower: the C2 fix guarded only a sequence-*opening* `1`, and its comment claims a continuing `2, 3, …` "needs no such proof". That is false — `define charge`, `define just`, `define ratio`, `define depth`, `define shortness`, `define glop`, `define logarithmic` all manufacture a sense from a prose numeral, burying a real NOAD sense inside a quoted gloss and leaking raw `|`. I confirmed ≥10 distinct entries by hand out of 52,204 (~0.02%). I rate this Important rather than Critical because it sits an order of magnitude *below* limitations the atlas already documents and accepts (~600 entries, 0.8%), and because I tried and failed to find a clean discriminator — `absolute`, `bases`, `ambrosia` and `bind` all carry *real* unplaced continuing senses, so the obvious fix regresses them. The deliverable is a fix **or** an honest `Limits` entry with the measured number. The one thing that should not ship as-is is the atlas's "**0** over the live sample" claim: at full width 8 distinct entries still render a raw `|` — the atlas has now recorded that exact lesson twice.

## 1. Strengths

- **The invariant is now bidirectional, and it holds at full dictionary width.** I ran the repo's own predicate plus the new count-equality bound over all 70,897 reachable live entries: **0 loss, 0 insertion**. The `%q` → explicit-quoting fix at `cmd/define/render.go:111` is confirmed on `blow` (no `\"`) and `otherwhere` (the U+00AD survives as one rune, not five). This is confirmed-good ground.
- **The delimiter-depth guard in `parseSenses` works.** `ParseEntry("...another term for pasha (sense 1 of the noun)...")` now renders the gloss intact with a balanced paren, and `on January 1 1992` no longer splits. Both prior-round repros pass.
- **ARCH-MOCK is fully closed and I verified it live.** All three seams have a conformance check and all three pass unsandboxed: fixtures byte-match live NOAD, the CDN survey still holds (`gaslighting` 200 on 2022 / 404 on legacy), and `TestAfplayBlocksUntilPlaybackCompletes` takes 7.59s — genuinely proving `afplay` blocks, which is what makes "plays 3×" mean anything.
- **Docs gate satisfied.** `README.md:29-45` now documents `-times`, `-no-audio`, `-locale`, `make install`, exit codes, and the active-dictionary limitation; `atlas/index.md` links `atlas/define.md`. `make -n install` resolves against the inherited `build` target.
- **Plan/code agreement on entities.** I checked all 23 rows of both Core-concepts tables against the filesystem — every one exists at its stated path, including the `_test.go` corrections for `fakeDictionary`/`fakeCDN`/`fakePlayer`.

## 2. Critical findings

None.

## 3. Important findings

### I1 — `parseSenses` still manufactures a sense from a prose numeral once the sequence is running; the code's stated justification for exempting continuing numbers is false
`cmd/define/parse.go:518-522`

```go
// A continuing number (2, 3, …) needs no such proof — the sequence it
// continues is the evidence, and NOAD does not always write punctuation
// before it ("plural form of base1 2 …" in `bases`).
if n == 1 && !opensBlock(text, loc[2]) {
    continue
}
```

The sequence is not evidence: any prose numeral that happens to equal `want` is accepted. Verified live at HEAD —

```
$ define charge
    1. demand (an amount) as a price ... for a service rendered or goods supplied
      [with two objects] "he charged me"                       <- example cut mid-sentence
    2. euros for the postcard | the restaurant charged $15 for dinner | [no object]
      • (charge something to) record the cost ...
        "they charge the calls to their credit card accounts. 2 accuse (someone) of
         something, especially an offense under law: they were charged with assault"
```

NOAD's real sense 2 ("accuse (someone) of something") is buried inside a quoted example, and two raw `|` reach the screen. Same shape on `just` (`4. percent | I only just caught the train | ... 4 simply; only; no more than` — the real sense 4's heading swallowed into a fake gloss), `ratio` (`2. to 1.`), `glop` (`2. gallons of premixed compound into a bucket.`), `logarithmic` (`2. is ten times more acidic than pH 3.`), `depth`, `shortness` (NOAD literally writes "question 4 remained unanswered"), plus `fugu`, `hyperemesis`, `put`.

Measured: ≥10 distinct entries hand-confirmed out of 52,204 distinct live entries (~0.02%). `TestCorpusSenseNumbersAreSequential` (`render_test.go:96`) cannot see it — a manufactured continuing number is by construction sequential.

Fix is not one token. I checked the obvious rule (require `opensBlock` for every number) and it regresses real senses: `absolute` ("Law see absolute title **2** viewed or existing…"), `bases`, `ambrosia` ("another term for bee bread **4** North American English…"), `bind` ("binding the variable x **8** Linguistics…") are all genuinely unplaced. An example-run mask (numeral after a `:` with no intervening sentence end) catches `charge`/`glop`/`logarithmic`/`ratio` but false-positives on taxonomic notes (`Family Cepolidae: several genera and species **2** a large edible fish`). Given that, the acceptable outcomes are (a) a discriminator that survives both sets, or (b) a `Limits` entry in `atlas/define.md` with the measured 0.02% and the four named examples, alongside the block-boundary limitations already recorded there. At minimum, correct the comment — it currently asserts a safety property that does not hold.

### I2 — `isPronunciation` requires a stress mark on every multi-word span, so unstressed and dialect-labelled pronunciations are rejected outright and the entry loses its IPA entirely
`cmd/define/parse.go:155-167`

Rule 2 demands at least one `ˈ`/`ˌ` in any multi-word part. NOAD writes plenty without one — monosyllabic phrases and `BrE`/`AmE` variant pairs:

```
$ define Jan                        raw: Jan | BrE dʒan, AmE dʒæn | abbreviation = January
Jan
    | BrE dʒan, AmE dʒæn | abbreviation = January        <- raw pipes, no /…/ at all

$ define greenroom                  raw: green room | ɡrēn ro͞om, ro͝om | noun a room in a theater…
green
    room | ɡrēn ro͞om, ro͝om | noun a room in a theater or studio in which…
```

`greenroom` is worse than a missing IPA — the multi-word headword splits, so "room" becomes body text and the entry has no POS heading. `vol` is the third case.

Measured with a purpose-level oracle: of 67,252 live entries whose raw carries a pronunciation span, **3 render no `/…/` at all** (0.004%). Small, but it is a total loss of the tool's headline feature for those words, and it is the same false-negative direction that shipped round 3's Critical (`define "hot dog"` → `/ˈhätˌdäɡər/`). A plausible extension: accept a multi-word span when every word carries a NOAD phonetic diacritic (`ē ō ə ͞ ͝ ʒ ɒ æ`), which `ɡrēn ro͞om` and `dʒan` satisfy and prose ("he charged me") does not. Fix or document; do not leave it silently outside the atlas's `Limits`.

### I3 — the insertion oracle added for the last round's C1 exists only at the narrowest of the three widths
`cmd/define/invariant_test.go:91` (present) vs `:134` (fuzz) and `cmd/define/live_property_test.go:82` (live)

`TestRenderLosesNothing` now asserts `len(want) != len(got)`. `FuzzRenderLosesNothing` and `TestRenderLosesNothingOverLiveEntries` still call `subsequenceGap` only — so the property the last commit strengthened is checked against 29 strings and nothing else. This is precisely the pattern the atlas's own three-width table exists to prevent ("the boundary reviews found bugs that each narrower check had shipped green").

I applied the two-line change to the fuzz target and it found a real insertion bug in **91 seconds**:

```
alnum count raw=4 rendered=6 for "· noun·"   ->  "·  no  noun"
```

Root cause, `cmd/define/parse.go:349`:

```go
suffix := stripped[len(bare):]                      // length measured on the ·-STRIPPED token
add(HeadSyllables, tok[:len(tok)-len(suffix)])      // sliced off the UN-stripped token
```

When the token's suffix region contains an interpunct the two lengths disagree and the head duplicates text — `ParseEntry("record rec·ordno·un | ˈrekərd | 1 a thing.")` renders `rec·ordno  noun` (30 alnum in, 32 out). Not reachable from real NOAD (my full-width sweep measured INSERT=0), so the bug itself is Minor; the finding is that the oracle which would have caught it isn't wired to the widths that explore. Fix: add the count bound to both, and take the derived `suffix` length from `tok` rather than `stripped`.

### I4 — `ErrFetchFailed`, introduced by this window's I5 fix, has no test at all
`cmd/define/fetch.go:21,75`

`grep ErrFetchFailed cmd/` returns three hits, all in `fetch.go`. Nothing asserts `errors.Is(err, ErrFetchFailed)`, nothing asserts that a transport failure is no longer reported as `ErrNoAudio`, and nothing pins the `%w: %w` chain that lets `errors.Is` reach `context.Canceled`. That chain is the entire point of the fix — a regression to `%v`, or a swap back to `ErrNoAudio`, passes every test in the repo. `fakeCDN` can produce it cheaply (close the server before `Fetch`, or cancel the context) and `TestFetchAllMissingTriesEveryCandidate` is the natural sibling.

### I5 — `atlas/define.md` publishes "0" for raw notation on the strength of a 3.8% sample, and the claim is measurably wrong at full width
`atlas/define.md:110-111` ("Both now measure **0** over the live sample"), `:84` ("7% → **0%**"), `:80` (the three-width table, "~2750 real NOAD entries")

The live sweep strides 2,749 of 71,427 reachable entries. At full width: **8 distinct entries render a raw `|`** (`charge`, `depth`, `just`, `shortness`, `greenroom`, `Jan`, `vol`, plus `pipe` — which is a true negative, NOAD genuinely defines "the symbol |") and **5 more carry a stress mark outside a `/…/` span** (`hundred`, `million`, `thousand`, `short`, `shapable`). The atlas devotes a whole section to the lesson that "a claim must not outrun what was actually measured," and it has now happened a third time at the next order of magnitude. Two cheap fixes: state the sample scope explicitly, and raise `liveSampleSize` (`live_property_test.go:24`) — the full sweep runs in **37 seconds** on this machine, and the `checked < 500` guard still protects it.

### I6 — the plan's normative block-opener rule still omits `]`, after two reviews asked for it
`workshop/plans/000001-define-plan.md:534-536`

```
> A part-of-speech word opens a block only when it is **structurally** placed —
> at the start of the body, or immediately after a sentence end (`.`, `)`, `:`,
> `;`) — and never inside a bracket or paren.
```

`opensBlock` (`cmd/define/parse.go:738`) accepts `]`. The round-4 prose entry at `:660` says so, but the block quote is the *specification* — it is what the next block-boundary change will be designed against. Both the round-4 review (rec. #5) and the close review (rec. #3) asked for this edit specifically; neither landed.

### I7 (ARCH-DRY) — delimiter-nesting depth is implemented three times, and the C2 fix added the third rather than reusing the second
`cmd/define/parse.go:233` (`findPronunciationIn`, paren only, inline), `:416` (`parseBlocks`, paren+bracket tracked separately, inline), `:554` (`delimiterDepths`, paren+bracket merged, a function)

The close review's C2 wrote "`parseBlocks` already implements the exact rule this needs" — the fix then wrote a fourth-of-a-kind helper beside it instead of extracting one. The three also disagree in detail (`parseBlocks` keeps paren and bracket in separate counters; `delimiterDepths` merges them, so `([)` behaves differently). `parseBlocks` can call `delimiterDepths(body)` and test `depths[i] > 0`, which is the same predicate its comment already describes.

## 4. Minor findings

- Examples containing quoted speech render doubled quotes: `""This blows," she sighs"` (`render.go:111`). Faithful, but a distinguishable inner quote style would read better.
- `fetch_conformance_test.go:66` indexes `data[0]`/`data[1]` after a **non-fatal** `len(data) < 1000` check — panics on a 1-byte body instead of failing. Open since the close review.
- `Fetch` never checks `ctx.Err()` between candidates, so a cancelled context still walks all four (`fetch.go:57`). Open.
- No overall deadline on the audio phase: 4 candidates × 10 s = up to ~40 s of hang *after* the definition printed. A `context.WithTimeout` in `run()` bounds it. Open.
- A 200 with an empty or non-audio body is accepted as a recording — exit 0, `♫ playing 3×`, silence. `fetch_conformance_test.go` already knows what a valid recording looks like; production checks neither. Open.
- `-locale zz` is unvalidated and degrades silently to "no recorded pronunciation" (verified). Open.
- `player.go:32` runs `exec.LookPath("afplay")` on every play — three times per invocation. Open.
- `url.PathEscape("..")` is a no-op, so `define ..x` puts a literal `..` segment in the CDN path (`audiourl.go:40`). Open.
- README documents exit 1 as "no dictionary entry", but `ErrLookupFailed` (a genuine CoreFoundation malfunction) also exits 1. Open since round 2.
- `invariant_test.go:135` `len([]rune(alnum(raw)))` is a no-op conversion — `alnum` already returns `[]rune`. Open since round 2.
- `atlas/define.md:26` — the Seam table declares four columns (`Seam | Wraps | Fake | Status`) but every row has three cells; `Status` renders empty. Open.
- `atlas/define.md:41` — "## Three parsing rules worth knowing" followed by "Two shapes are non-obvious" and three items. Open since round 3.
- `atlas/define.md:56` points at `testdata/fuzz/` for minimized crashers; the directory does not exist (accurate — 6.2M execs in my run found none, and the one I induced was via a modified oracle). Open.
- README says `go test -tags conformance ./...`; the atlas says `./cmd/define/`. Open since round 3.
- "cardinal number" is not in `posWords`, so `define hundred` renders a first block with no POS heading. Same family as the documented `backheel` limitation.
- Eleven Minors from the previous close review are carried forward unchanged with no recorded disposition. Leaving Minors open is fine; leaving them undisposed means the next reviewer re-derives them.

## 5. Test coverage notes

88.9% of statements, vet clean on darwin and `GOOS=linux CGO_ENABLED=0`, all three live conformance checks passing, `go mod tidy` a no-op, no build artifacts tracked or in history. `TestRunNoAudioMakesNoRequests`, `TestFetchWalksCandidatesInOrderAndStopsAtFirstHit`, `TestAfplayBlocksUntilPlaybackCompletes`, and the strip-and-compare in `TestRenderColorOnlyWhenAsked` are real assertions on real behaviour, not mocks restating the implementation. CLI edge cases (empty word, 5000 chars, emoji, path-like input, `-times 0`, `-h`) all exit cleanly with no panic.

Gaps, in payoff order:

1. **The insertion oracle is wired to 29 strings only** (I3). Two lines each in `invariant_test.go:134` and `live_property_test.go:82`; the fuzz version found a real bug in 91 s.
2. **`ErrFetchFailed` is untested** (I4) — a brand-new sentinel with no pin.
3. **No structural golden covers a numeral inside an example run** (I1). `charge` or `ratio` as a fixture with a "no sense number above N" assertion would pin it; neither is in the 29.
4. **No test covers a multi-word pronunciation without a stress mark** (I2). `Jan` or `green room` as a fixture, asserting `Entry.IPA != ""`.
5. **The legacy CDN fallback is untested end-to-end** — `newAudioRig` only ever populates `AudioCandidates(word)[0]`, so the composition of the 4-element list with `Fetch`'s walk is exercised at index 0 only, despite that fallback being the documented reason the list exists. Carried from the close review.
6. `realDeps` and `main` at 0% — correct, not worth chasing.

## 6. Architectural notes

- **ARCH-DRY — flag.** I7: three implementations of delimiter-nesting depth (`parse.go:233`, `:416`, `:554`), the newest added by the very fix whose finding said to reuse the existing one. Confirmed-good elsewhere: `alnum`/`subsequenceGap` shared across all three widths, `strayStress` defined once and called from two, `rewritePronunciations` one function with two call sites, `splitFirstToken` still the single token-boundary definition, and `capture.py`'s duplication of the cgo call correctly deliberate and documented.
- **ARCH-PURE — pass.** I proved this rather than inspected it: every repro in this review runs `ParseEntry` + `Render` on string literals with no fixture, no fake, no IO. `isTerminal` is parked at `main.go:110` and produces zero ANSI on a non-TTY writer; cgo is confined to `dict_darwin.go`; `deps` is explicit and injected; `playN` keeps the repeat loop in the shell so the count is assertable.
- **ARCH-PURPOSE — flag, narrow.** The shadow-sweep on the issue's purpose is near-clean at full width: of 67,252 entries whose raw carries a pronunciation, 67,249 render a `/…/`. The three exceptions (I2) plus the ~10 manufactured-sense corruptions (I1) are the residual, and neither is documented in the atlas's `Limits` — which is where this project has (correctly) chosen to record accepted parser gaps. The purpose is delivered; the honest accounting of what it does *not* cover has drifted behind the code.
- **ARCH-MOCK — pass.** This is the strongest dimension in the diff and I verified all of it live. Three seams, three fixture- or state-backed fakes, all in `_test.go` files, all with a live conformance check that passes unsandboxed. `dict_conformance_test.go:28` reads *through* `fake.Lookup` rather than around it; `capture.sh` enforces a byte floor; `loadFakeDictionary` rejects an empty corpus; the cadence is stated on-demand with a documented trigger. The `afplay`-blocks check is genuinely load-bearing and it earns its 7.6 s.
- **For future work here:** the durable lesson from this window is *direction* — a subsequence oracle cannot see interpolation, so a fidelity claim needs both bounds. That belongs in `workshop/lessons.md` next to the existing "a check must not consult the function it is checking" rule, with the corollary I3 demonstrates: **a strengthened oracle must be wired to every width, or it protects only the sample.** When `internal/` is finally earned, `ParseEntry`/`Render`/`AudioCandidates` are the stable surface and none of them needs to move.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000001-define-plan.md`:

1. **Correct the block-opener block quote at `:534-536`** to include `]` (I6). Asked for by two prior reviews; it is specification, and the code has diverged from it since round 4.
2. **Amend the close-review entry's C2 paragraph.** It states that "a continuing `2` … needs no such proof — the sequence it continues is the evidence." That is falsified by `charge`, `just`, `ratio`, `depth`, `shortness`, `glop`, `logarithmic` (I1). This is the fourth consecutive round in which a Revisions entry's closing claim was disproved by the next measurement; recording the correction here is what stops the plan from teaching the next change something untrue.
3. **Record the delimiter-depth rule for sense splitting** as a normative statement in Chunk 1 beside the block-opener rule, noting it is the same rule one level down — and that it should share one implementation (I7), not three.
4. **State `isPronunciation`'s known false-negative direction** (I2) next to Rule B's verdict table: a multi-word span with no stress mark is rejected, which costs the IPA entirely on `Jan`/`vol`/`green room`.

The Core-concepts tables need no change — I verified all 23 rows against the filesystem.

For `atlas/define.md`: scope the "0" claims to the sample actually measured and give the full-width numbers (I5); add `Limits` entries for I1 and I2 with their measured frequencies; fix the Seam table's column count and the "Three rules / Two shapes" mismatch; and add **insertion** to "Two things the property cannot see" — it was the last round's Critical and the section still lists only two blind spots.

`workshop/plans/000001-define-plan-gate.md`'s `## Open findings` is empty with all six PQ items disposed `addressed` — nothing was deferred from the plan gate for this boundary to pick up, and I confirmed each disposition against the code.
