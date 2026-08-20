---
gate: plan-quality
issue: 1
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-20T10:03:59-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Block has no IPA field, but record's verb block carries its own pronunciation
          detail: 'Live DCSCopyTextDefinition("record") returns "record rec·ordnoun | ˈrekərd | 1 a thing … verb [with object] | rəˈkôrd | 1 set down …" — the verb block has a distinct IPA the model cannot hold. Same case: the head rule strips the glued "noun" out of "rec·ordnoun" but the model has no slot for it and record''s body has no POS token left, so the first Block gets an empty POS. record is a named Done-when fixture; fix the model before Entry has consumers.'
          round: 1
        - id: PQ-2
          severity: Important
          title: Parse rules do not disambiguate IPA pipes from example-separator pipes
          detail: In record, " | " delimits IPA (| rəˈkôrd |, | rəˈkôrdəb(ə)l |) and also separates multiple examples ("you should keep a written record | identification was made through dental records | a record of meter readings."). Parsing order step (1) only covers "the first |…| pair" in the header and is silent about the body, yet Sense.Examples depends on splitting the same delimiter. State one rule (e.g. a pipe span is IPA only when it contains no ASCII letters).
          round: 1
        - id: PQ-3
          severity: Important
          title: bank returns homograph 1 only; the limitation is unstated and the invariant masks it
          detail: The live call returns only "bank 1 | baNGk | noun 1 the land alongside…" — the financial-institution sense is unreachable through this API. Google's panel shows both, so this is an ARCH-PURPOSE gap against the issue's stated purpose. The plan states no non-goals of its own, and the no-data-loss invariant guarantees fidelity to the returned string, not entry completeness. Say so explicitly, and render Homograph so the user can see "bank 1".
          round: 1
        - id: PQ-4
          severity: Important
          title: capture.sh can emit zero-byte fixtures, making the invariant test vacuously green
          detail: '"go run ../ --raw $w > entries/$w.txt || echo …" creates the file by redirection before the command runs and swallows the failure, so a sandboxed run (already documented in the issue Log as returning nothing) yields empty fixtures and TestRenderLosesNothing passes trivially; wc -c is an eyeball, not an assertion. Hard-fail on a short capture and unlink the file. This also resolves the Task 1 to Task 6 circularity (Step 2 misreferences Task 5, and "the verified spike" is not in the repo) — the corpus can be captured with a self-contained python3 ctypes call into CoreServices, needing neither cgo nor --raw.'
          round: 1
        - id: PQ-5
          severity: Minor
          title: Compress the enumerated test cases and inline test bodies to strategy lines
          detail: 'Tasks 2, 4, 5, 7, 8, 9 reproduce full Go test functions and Task 10 Step 1 enumerates four cases in prose. Every one will be rewritten as code within the hour. One strategy line per risky function is the obligation — e.g. "ParseEntry: schema-less flat text, guarded by an alnum-subsequence invariant over the whole corpus, parser yields to the test." Keep the cgo body (hard-won, with the do-not-redeclare trap) and the awkward-header table (that is the spec, not a test case).'
          round: 1
        - id: PQ-6
          severity: Minor
          title: Conformance checks have no scheduled cadence
          detail: ARCH-MOCK at-plan asks for a live conformance cadence; both checks are "run on demand" behind a build tag. The repo already ships .github/workflows/merge-check.yml — name where these run, or state that on-demand after a macOS upgrade is the deliberate cadence and why.
          round: 1
      blocked: true
---

# Gate ledger — tools#1 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-20T10:03:59-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] Block has no IPA field, but record's verb block carries its own pronunciation
  Live DCSCopyTextDefinition("record") returns "record rec·ordnoun | ˈrekərd | 1 a thing … verb [with object] | rəˈkôrd | 1 set down …" — the verb block has a distinct IPA the model cannot hold. Same case: the head rule strips the glued "noun" out of "rec·ordnoun" but the model has no slot for it and record's body has no POS token left, so the first Block gets an empty POS. record is a named Done-when fixture; fix the model before Entry has consumers.
- **PQ-2** [Important] Parse rules do not disambiguate IPA pipes from example-separator pipes
  In record, " | " delimits IPA (| rəˈkôrd |, | rəˈkôrdəb(ə)l |) and also separates multiple examples ("you should keep a written record | identification was made through dental records | a record of meter readings."). Parsing order step (1) only covers "the first |…| pair" in the header and is silent about the body, yet Sense.Examples depends on splitting the same delimiter. State one rule (e.g. a pipe span is IPA only when it contains no ASCII letters).
- **PQ-3** [Important] bank returns homograph 1 only; the limitation is unstated and the invariant masks it
  The live call returns only "bank 1 | baNGk | noun 1 the land alongside…" — the financial-institution sense is unreachable through this API. Google's panel shows both, so this is an ARCH-PURPOSE gap against the issue's stated purpose. The plan states no non-goals of its own, and the no-data-loss invariant guarantees fidelity to the returned string, not entry completeness. Say so explicitly, and render Homograph so the user can see "bank 1".
- **PQ-4** [Important] capture.sh can emit zero-byte fixtures, making the invariant test vacuously green
  "go run ../ --raw $w > entries/$w.txt || echo …" creates the file by redirection before the command runs and swallows the failure, so a sandboxed run (already documented in the issue Log as returning nothing) yields empty fixtures and TestRenderLosesNothing passes trivially; wc -c is an eyeball, not an assertion. Hard-fail on a short capture and unlink the file. This also resolves the Task 1 to Task 6 circularity (Step 2 misreferences Task 5, and "the verified spike" is not in the repo) — the corpus can be captured with a self-contained python3 ctypes call into CoreServices, needing neither cgo nor --raw.
- **PQ-5** [Minor] Compress the enumerated test cases and inline test bodies to strategy lines
  Tasks 2, 4, 5, 7, 8, 9 reproduce full Go test functions and Task 10 Step 1 enumerates four cases in prose. Every one will be rewritten as code within the hour. One strategy line per risky function is the obligation — e.g. "ParseEntry: schema-less flat text, guarded by an alnum-subsequence invariant over the whole corpus, parser yields to the test." Keep the cgo body (hard-won, with the do-not-redeclare trap) and the awkward-header table (that is the spec, not a test case).
- **PQ-6** [Minor] Conformance checks have no scheduled cadence
  ARCH-MOCK at-plan asks for a live conformance cadence; both checks are "run on demand" behind a build tag. The repo already ships .github/workflows/merge-check.yml — name where these run, or state that on-demand after a macOS upgrade is the deliberate cadence and why.

## Open findings

- **PQ-1** [Important] Block has no IPA field, but record's verb block carries its own pronunciation
- **PQ-2** [Important] Parse rules do not disambiguate IPA pipes from example-separator pipes
- **PQ-3** [Important] bank returns homograph 1 only; the limitation is unstated and the invariant masks it
- **PQ-4** [Important] capture.sh can emit zero-byte fixtures, making the invariant test vacuously green
- **PQ-5** [Minor] Compress the enumerated test cases and inline test bodies to strategy lines
- **PQ-6** [Minor] Conformance checks have no scheduled cadence
