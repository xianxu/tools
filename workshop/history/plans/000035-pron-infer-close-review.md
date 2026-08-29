# Boundary Review — tools#35 (whole-issue close)

| field | value |
|-------|-------|
| issue | 35 — /pron with no language: infer the origin from ORIGIN, error only when it cannot be determined |
| repo | tools |
| issue file | workshop/issues/000035-pron-infer.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1db0ee55f820a3c4f754fbf262bc5a2be28fd20..13defbf44825c5ece7d7c0286254ba7594c53b3b |
| command | sdlc close --issue 35 |
| reviewer | claude |
| timestamp | 2026-08-29T15:57:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is real and the hard part is genuinely pinned: `go test ./...` is green, and I verified by reverting in scratch copies that the cognate cut (7 corpus subtests + the rules row + `TestPronReportsWhyItCannotInfer/read` go red) and `historicalStages` (10+ subtests go red) are load-bearing and tested. What blocks a clean SHIP is a cluster of record-vs-code gaps that this issue's own standard rejects: the word-boundary guard the code comment and the closing Log both promote to "the real guard, pinned by its own case" is **not** pinned — I removed `\b` and all 47 subtests stayed green; the historical-stage exclusion is claimed "BY CATEGORY" (D4) but is an enumerated list that misses its own siblings, so `from Old Italian`/`Middle French`/`Old Spanish`/`Low German` all infer a live language and report `ORIGIN says Italian` when ORIGIN says *Old* Italian; the doc sweep fixed the three prose sites the plan named while four more members of the same class stayed false; and the atlas record that the issue's Plan checkbox and the plan's Done-when row 5 both promise ("D5 is dated prose in the atlas") does not exist — the atlas gained one clause at line 1150 and nothing else.

## 1. Strengths

- **`TestOriginLanguageOverTheCorpus` (origin_test.go:22) is the right shape.** Ranging over every `en` fixture with an unlisted-fixture → `t.Errorf` rule genuinely forces a decision when a fixture is added, and the `len(d.entries)==0` guard keeps it from going vacuous. This is the PQ-6 remediation done properly.
- **The cognate cut is measurably the guard it claims to be.** Deleting the loop at origin.go:120-124 reddens `even`, `man`, `thing`, `read`, `bank`, `bargainer`, `set`, the rules row, and the integration test — with `read` replaying `read_nl_nl`, exactly the `#29` D1 failure the cut exists to stop.
- **`TestAnOrdinaryLookupNeverInfersTheOrigin` (main_test.go:449) is non-vacuous** — I confirmed the run makes exactly one request, `jalapeño_en_us`, so the assertion loop has something to inspect.
- **`pronCommandHelp` was placed correctly.** Not in `pronHelp` (D6: the flag doesn't infer), not in the `commands` summary (`#31`'s recorded reason) — its own const with a marked span. The reasoning at pron_cmd.go:75-86 is the kind of decision record that pays off later.
- **First-named-wins is deterministic despite the map range** — the strict `loc[0] < best` tie-break plus word-boundaried, non-overlapping names means iteration order can't change the answer.

## 2. Critical findings

None.

## 3. Important findings

**(a) `cmd/define/origin.go:59-61` — the word boundary is asserted to be pinned, and it is not.** The comment says "What actually protects `run` is the WORD BOUNDARY … and that is pinned by its own case in `TestOriginLanguageRules`", and the issue Log says "dropping the boundary reddens five subtests." Measured: replacing `` `\b` + QuoteMeta(name) + `\b` `` with `QuoteMeta(name)` at origin.go:134 leaves **all 47 subtests green**. The two rows meant to pin it (origin_test.go:113-115, `"of Germanic origin."`) are held green by the `Germanic` mask, which the same comment demotes to "redundancy". The two mechanisms are mutually redundant and each hides the other's removal. *Fix:* add a rules row the mask cannot save — e.g. `{"a country is not a language", "named after a town in Germany.", ""}` — which reddens only when `\b` is dropped; then the comment's claim is true.

**(b) `cmd/define/origin.go:66-71` — `historicalStages` enumerates instances, not the category D4 promises.** `Old French`/`Old English`/`Middle Dutch` are masked, but the identical construction over other mapped languages is not. Measured against HEAD: `from Old Italian mezzo` → `it/Italian`, `from Middle French bureau` → `fr/French`, `from Old Spanish casco` → `es/Spanish`, `from Low German bugsēren` → `de/German`. Each plays a modern recording for an explicitly superseded stage — the failure D2/D4 exist to prevent — and each prints `ORIGIN says Italian` when ORIGIN says *Old* Italian, so the record is untrue too. *Fix:* derive the mask from the map (for every `name` in `originLanguages`, mask `Old <name>`, `Middle <name>`, `Old High <name>`, `Low <name>` before searching), keeping the hand list only for stages with no modern member (`Latin`, `Greek`, `Sanskrit`, `Old Norse`, `Frankish`, `Germanic`); add rows for the four above.

**(c) `cmd/define/pron_cmd.go:58-70` — bare `/pron` before any lookup blames an entry that does not exist.** The inference runs before the `c.replay == nil` check, so `/pron` as the first line of a session prints `define: /pron: no source language named: this entry has no ORIGIN. Name one: /pron fr` (measured). `command.go:191` claims the opposite — "Empty when nothing has been looked up, which /pron reports rather than inferring from nothing" — and the correct message at pron_cmd.go:68 is now unreachable for the bare form. `TestPronWithNothingLookedUpSaysSo` uses `/pron fr`, so nothing pins the bare path. *Fix:* move the `c.replay == nil` block above the inference (`replay` is nil exactly when `!sess.hasCurrent()`, repl.go:364), and add the bare-`/pron`-with-nothing-looked-up case to that test.

**(d) `cmd/define/pron_cmd_test.go:116-135` — the test's `because` field is never asserted.** The doc comment says "it declines with the reason, on the two shapes that differ", but the body only checks that stderr contains `/pron`; `tc.because` is dead. The uncovered bug is already shipped: `gaslighting` ("1960s: see gaslight (verb)") names no language at all yet gets `its ORIGIN names only historical stages or cognates` — the wrong one of the two messages. *Fix:* assert `strings.Contains(errb.String(), tc.because)`, and give the no-language-named case its own wording distinct from the stages/cognates one.

**(e) Doc sweep fixed three named sites; the class has at least four more members (ARCH-PURPOSE).** Still false or unpinned after this range:
- `README.md:207` — "You name the language; the tool never guesses it." — two lines below a span that now says `/pron` alone reads it for you.
- `atlas/define.md:1326` — "**The language is DECLARED, never inferred**" — same contradiction, five lines below the same span.
- `cmd/define/pron_cmd.go:12-16` — the function's own doc comment still reads "A language is REQUIRED … a bare one is a half-typed command", contradicted by the inline comment three lines below it.
- `README.md:340-345` — hand-restates `pronCommandHelp` in near-identical words while `TestDocsQuoteThePronCommandHelp` (doc_sync_test.go:200) hardcodes only the atlas, breaking the `for _, doc := range derivedDocs` pattern its two sibling tests use. `derivedDocs` already includes README (dictselect_test.go:487). This is precisely the "fixing the README alone left atlas/define.md as the next copy to go stale" half-fix that doc_sync_test.go:135-137 warns about, run in the other direction. *Fix:* range the new test over `derivedDocs` and make README consume the span; sweep the three prose/comment sites, qualifying "never inferred" as "never inferred on an ordinary lookup (`#29` D1); `/pron` alone infers on request and reports (`#35`)".

**(f) The atlas record promised by a checked plan item does not exist (AGENTS.md §8).** The issue's Plan says `- [x] Docs: … plus the atlas record of why bare Greek is ancient and why this table is accepted where ParseLang refuses one`, and the plan's Done-when row 5 pins D5 on "dated prose in the atlas". `grep -n "ancient\|OriginLanguage\|originLanguages\|Greek" atlas/define.md` returns nothing; the only `#35` mention is one clause at atlas/define.md:1150. A new file, a new pure entity, a new closed table, and a reversal of a documented stance went in with no atlas surface record. *Fix:* add a short `Origin inference (#35)` block to `atlas/define.md` carrying D0/D3/D4/D5, and tick the checkbox only then.

## 4. Minor findings

- `cmd/define/origin.go:134` — a fresh `regexp.MustCompile` per language per call (22 compiles per `/pron`); precompile at package level or build one alternation. ARCH-DRY.
- `cmd/define/origin.go:87-90` — `"compare "` subsumes `"compare with"`, and `"cognate with"` has no capitalized twin while every other marker does; the enumeration is inconsistent.
- `cmd/define/origin.go:93` — `fmt.Errorf` with no verbs for a sentinel; `errors.New`. No caller uses `errors.Is(err, ErrNoOriginLanguage)` yet.
- Plan Core-concepts table omits `cognateMarkers` (the D0 entity) and `ErrNoOriginLanguage`.
- Task 4 checkbox says "**`pronHelp` is left alone**", but voice.go:139-141 edits it. The added clause is true where it is written, so this is a record fix, not a code fix.
- `README.md:344-346` — "`/pron fr` names it explicitly. Either way it / leaves nothing switched on" reads as a wrap artifact of the original sentence.

## 5. Test coverage notes

- Verified red-on-revert: cognate cut ✓, `historicalStages` ✓. Verified **not** red-on-revert: the word boundary (finding a).
- No test covers `/pron` in a non-English session, where the entry has `ETIMOLOGÍA` rather than `ORIGIN` (`testdata/entries/es/*`); today it declines with "this entry has no ORIGIN", which is the right outcome but unpinned.
- `TestAnOrdinaryLookupNeverInfersTheOrigin` would pass if the request list were empty; a `len(rig.cdn.Requested()) == 0 → t.Fatal` guard would make it as robust as the corpus test's own vacuity guard.
- Nothing checks the inference at dictionary width. The Spec's 220-word measurement ("no ORIGIN 171 / historical-only 31 / determinate 15") is exactly the ratchet that would have caught finding (b); `live_property_test.go` already walks `/usr/share/dict/words` and is the natural home.

## 6. Architectural notes

- **ARCH-DRY — flag.** README restates `pronCommandHelp` by hand though `derivedDocs` makes it derivable (e); per-call regex construction and the overlapping cognate markers are the smaller instances.
- **ARCH-PURE — pass.** `OriginLanguage` is a pure function over a parsed `Entry` in its own file; `TestOriginLanguageRules` and the fuzz target run with zero IO; the loops only copy `sess.entry` into `commandCtx`, which stays data rather than a capability. Putting the inference in the command that owns the message is the right seam.
- **ARCH-PURPOSE — flag.** Two instance-not-class fixes: the doc sweep (e) and the stage mask (b). Both are enumerable, and the enumeration was not written.
- **ARCH-MOCK — pass.** No new external dependency; the fake dictionary corpus is the fixture set behind the same seam and `TestFixturesMatchLiveDictionary` is its live half. Gap noted above: no live conformance row for the inference's own outcome distribution.
- For `#30`: `OriginLanguage` returning `(code, name, err)` is the right surface for a click that supplies the name directly — but the name→code half is currently unreachable without the whole entry. Exporting a tiny `languageNamed(string) (store.Lang, bool)` over `originLanguages` when `#30` lands avoids a second table.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000035-pron-infer-plan.md`:

- **Correcting the implementation note on the `Germanic` mask.** The note claims "dropping the boundary reddens five subtests"; measured at `13defbf`, dropping `\b` reddens **nothing**. Record that the mask and the boundary are mutually redundant, that neither is isolated by a test today, and name the row added to isolate the boundary.
- **D4 narrowed to what the code does.** `historicalStages` is an enumerated list, not a category rule; state which stage constructions are covered and either commit to deriving the mask from `originLanguages` or record the uncovered residue (`Old Italian`, `Middle French`, `Old Spanish`, `Low German`) as known.
- **Task 4's site table was incomplete.** Three sites listed, at least six in the class; add `README.md:207`, `atlas/define.md:1326` and `parsePronArgs`'s own doc comment, and record that `pronHelp` *was* edited despite the "left alone" checkbox.
- **Done-when row 5 and the issue Plan's docs checkbox claim an atlas record that does not exist** — either write it or untick both.
