---
gate: plan-quality
issue: 31
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-29T11:42:24-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Plan claims the it corpus auto-enters suites that are hardcoded to English, and D3/Done-when-6 rest on it
          detail: |-
            Only capturedLanguages-driven checks generalise: TestEveryCapturedLanguageLoads,
            TestRenderLosesNothingInEveryCapturedLanguage (dict_fake_test.go:297) and
            TestFixturesMatchLiveDictionary. These take testDict(t) and stay English:
            render_test.go:122, 154, 230, 302; invariant_test.go:161, 205;
            highlightwriter_test.go:326, 445; parse_test.go:11. The raw-notation corpus
            sweep IS render_test.go:230, so Italian gets no notation coverage at any
            width, not "corpus width" as D3 states. Enumerate the class and decide per
            check; the cheap Italian closure is the sibling of
            TestSpanishEntriesCarryNoPronunciationNotation (render_test.go:335) asserting
            ParseEntry(raw).IPA == "" over the it corpus, which also pins the measured
            fact that isPronunciation declines the syllabification form.
          family: corpus-check-language-blind
          round: 1
        - id: PQ-2
          severity: Important
          title: Task 1 Step 3's confirmation command cannot produce the it subtest it tells the implementer to look for
          detail: |-
            TestRenderLosesNothing (invariant_test.go:161) walks testDict(t), so its
            subtests are English words only; the language-general variant
            (dict_fake_test.go:299) is a flat loop with no t.Run, so no it subtest exists
            under either name. Done-when coverage row 3 names the same wrong test. Point
            both at TestEveryCapturedLanguageLoads and
            TestRenderLosesNothingInEveryCapturedLanguage.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-3
          severity: Important
          title: Done-when row 4 claims a doc-sync pin that does not exist, while Task 4 adds a third language to an unpinned restatement of curated
          detail: |-
            doc_sync_test.go pins the per-cause raw-notation counts, pronHelp, localeHelp
            and the command table; nothing pins a notation table or the curated language
            list. README.md:350-354 hand-restates curated's books and Task 4 extends it
            to Italian. Either generate a marked span from curated on the
            TestDocsQuoteTheCommandList pattern, or drop the "pinned by doc-sync" cell so
            the close review is not handed a pin that isn't there.
          family: doc-sweep-incomplete
          round: 1
        - id: PQ-4
          severity: Minor
          title: Task 1 omits capture.sh's closing summary, which will under-report the corpus
          detail: |-
            The trailing echo counts English and Spanish only, and the wc -c line globs
            entries/en and entries/es. Neither is in Task 1's step list.
          family: capture-script-summary-stale
          round: 1
        - id: PQ-5
          severity: Minor
          title: Done-when row 3's removal test is silent when entries/it is deleted rather than emptied
          detail: |-
            capturedLanguages derives from the directory listing and its guard only
            requires len(out) >= 2 (dict_fake_test.go:274), so deleting entries/it leaves
            en+es and reddens nothing — the plan's own "red when its wiring is removed"
            rule is satisfied only under the emptied reading.
          family: derived-coverage-silent-on-removal
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-29T11:46:05-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Names fixed in the Architecture paragraph; D3 and Done-when row 6 still rest on the false claim and the it notation closure is still absent.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Right tests, wrong probe — measured `grep -c '/es'` = 0 today, so `grep -c '/it'` cannot be non-zero.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 5's TestDocsQuoteTheCuratedLanguages replaces the pin that did not exist; see the new finding on its source data.
          round: 2
        - id: PQ-4
          disposition: addressed
          round: 2
        - id: PQ-5
          disposition: addressed
          note: TestEveryCuratedLanguageHasACorpus plus the Done-when row 3 asymmetry note.
          round: 2
      findings:
        - id: PQ-6
          severity: Important
          title: Task 5's generated doc span has no source — `curated` holds identifiers, the README prose names book titles
          detail: |-
            `TestDocsQuoteTheCommandList` works because `commands` owns `name` and
            `summary`, the strings the table prints. `curated` (dictselect.go:96) is
            language to bundle identifiers only, and nothing maps
            `com.apple.dictionary.es.DGLEV` to the "Larousse Diccionario General" the
            README names at README.md:350-354. The plan must choose: add a title field
            to `curated`'s value (a shape change to the single source all consumers
            read), emit an identifier table and change what that paragraph is, or
            weaken the check to language-name containment.
          family: source-lacks-the-fact-consumer-renders
          round: 2
        - id: PQ-7
          severity: Minor
          title: The issue file's `## Plan` still says "4 tasks" and omits Task 5's two mechanisms
          detail: |-
            workshop/issues 000031 lists four checkboxes and the summary line "(4 tasks,
            single pass, no Mx)"; the plan now has five tasks, and the two Task 5
            mechanisms are the ones PQ-3 and PQ-5 turn on. The close gate's
            plan-unchecked guard will read the stale list.
          family: doc-sweep-incomplete
          round: 2
      blocked: true
---

# Gate ledger — tools#31 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T11:42:24-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `corpus-check-language-blind` Plan claims the it corpus auto-enters suites that are hardcoded to English, and D3/Done-when-6 rest on it
  Only capturedLanguages-driven checks generalise: TestEveryCapturedLanguageLoads,
  TestRenderLosesNothingInEveryCapturedLanguage (dict_fake_test.go:297) and
  TestFixturesMatchLiveDictionary. These take testDict(t) and stay English:
  render_test.go:122, 154, 230, 302; invariant_test.go:161, 205;
  highlightwriter_test.go:326, 445; parse_test.go:11. The raw-notation corpus
  sweep IS render_test.go:230, so Italian gets no notation coverage at any
  width, not "corpus width" as D3 states. Enumerate the class and decide per
  check; the cheap Italian closure is the sibling of
  TestSpanishEntriesCarryNoPronunciationNotation (render_test.go:335) asserting
  ParseEntry(raw).IPA == "" over the it corpus, which also pins the measured
  fact that isPronunciation declines the syllabification form.
- **PQ-2** [Important] `unbacked-existing-behavior-claim` Task 1 Step 3's confirmation command cannot produce the it subtest it tells the implementer to look for
  TestRenderLosesNothing (invariant_test.go:161) walks testDict(t), so its
  subtests are English words only; the language-general variant
  (dict_fake_test.go:299) is a flat loop with no t.Run, so no it subtest exists
  under either name. Done-when coverage row 3 names the same wrong test. Point
  both at TestEveryCapturedLanguageLoads and
  TestRenderLosesNothingInEveryCapturedLanguage.
- **PQ-3** [Important] `doc-sweep-incomplete` Done-when row 4 claims a doc-sync pin that does not exist, while Task 4 adds a third language to an unpinned restatement of curated
  doc_sync_test.go pins the per-cause raw-notation counts, pronHelp, localeHelp
  and the command table; nothing pins a notation table or the curated language
  list. README.md:350-354 hand-restates curated's books and Task 4 extends it
  to Italian. Either generate a marked span from curated on the
  TestDocsQuoteTheCommandList pattern, or drop the "pinned by doc-sync" cell so
  the close review is not handed a pin that isn't there.
- **PQ-4** [Minor] `capture-script-summary-stale` Task 1 omits capture.sh's closing summary, which will under-report the corpus
  The trailing echo counts English and Spanish only, and the wc -c line globs
  entries/en and entries/es. Neither is in Task 1's step list.
- **PQ-5** [Minor] `derived-coverage-silent-on-removal` Done-when row 3's removal test is silent when entries/it is deleted rather than emptied
  capturedLanguages derives from the directory listing and its guard only
  requires len(out) >= 2 (dict_fake_test.go:274), so deleting entries/it leaves
  en+es and reddens nothing — the plan's own "red when its wiring is removed"
  rule is satisfied only under the emptied reading.

## Round 2 — 2026-08-29T11:46:05-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Names fixed in the Architecture paragraph; D3 and Done-when row 6 still rest on the false claim and the it notation closure is still absent.
- PQ-2 — not-addressed — Right tests, wrong probe — measured `grep -c '/es'` = 0 today, so `grep -c '/it'` cannot be non-zero.
- PQ-3 — addressed — Task 5's TestDocsQuoteTheCuratedLanguages replaces the pin that did not exist; see the new finding on its source data.
- PQ-4 — addressed
- PQ-5 — addressed — TestEveryCuratedLanguageHasACorpus plus the Done-when row 3 asymmetry note.

### Raised

- **PQ-6** [Important] `source-lacks-the-fact-consumer-renders` Task 5's generated doc span has no source — `curated` holds identifiers, the README prose names book titles
  `TestDocsQuoteTheCommandList` works because `commands` owns `name` and
  `summary`, the strings the table prints. `curated` (dictselect.go:96) is
  language to bundle identifiers only, and nothing maps
  `com.apple.dictionary.es.DGLEV` to the "Larousse Diccionario General" the
  README names at README.md:350-354. The plan must choose: add a title field
  to `curated`'s value (a shape change to the single source all consumers
  read), emit an identifier table and change what that paragraph is, or
  weaken the check to language-name containment.
- **PQ-7** [Minor] `doc-sweep-incomplete` The issue file's `## Plan` still says "4 tasks" and omits Task 5's two mechanisms
  workshop/issues 000031 lists four checkboxes and the summary line "(4 tasks,
  single pass, no Mx)"; the plan now has five tasks, and the two Task 5
  mechanisms are the ones PQ-3 and PQ-5 turn on. The close gate's
  plan-unchecked guard will read the stale list.

## Open findings

- **PQ-1** [Critical] `corpus-check-language-blind` Plan claims the it corpus auto-enters suites that are hardcoded to English, and D3/Done-when-6 rest on it
- **PQ-2** [Important] `unbacked-existing-behavior-claim` Task 1 Step 3's confirmation command cannot produce the it subtest it tells the implementer to look for
- **PQ-6** [Important] `source-lacks-the-fact-consumer-renders` Task 5's generated doc span has no source — `curated` holds identifiers, the README prose names book titles
- **PQ-7** [Minor] `doc-sweep-incomplete` The issue file's `## Plan` still says "4 tasks" and omits Task 5's two mechanisms
