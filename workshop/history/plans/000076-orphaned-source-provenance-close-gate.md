---
gate: boundary-review
issue: 76
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T13:59:31-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: dictionary_language_test.go is named for a production file this window deleted
          detail: |-
            dictionary_language.go was deleted because it "would hold nothing
            dictionary-related", but its test file keeps the name while holding only
            definition-tint tests (assertDictionaryTint and three TestDictionary*
            cases). Fold into definitions_output_test.go or rename.
          family: name-outlives-referent
          round: 1
        - id: BR-2
          severity: Minor
          title: TestDictionaryMonolingualOriginAndDisabledTint asserts neither origin nor tint
          detail: |-
            cmd/define/dictionary_language_test.go:45 now only checks that Color:false
            emits no ANSI. #70 removed the tint half; this window removed the last
            Language: literal that gave "MonolingualOrigin" any meaning. Rename to what
            it checks, or delete it if another no-color assertion already covers it.
          family: name-outlives-referent
          round: 1
        - id: BR-3
          severity: Minor
          title: atlas/define.md:2350 still describes the deleted ro.Language write
          detail: |-
            "renderDefinitionOutput retains section language ownership through
            rendering and region offsets" was the prose for `ro.Language =
            section.language`. What remains is `ro.Vocab = nil` for non-primary
            sections plus role-based row paint; a one-clause edit keeps the sentence
            true without naming a field that no longer exists.
          family: atlas-describes-removed-surface
          round: 1
        - id: BR-4
          severity: Minor
          title: projectDisplayText requires ascending non-overlapping spans, stated nowhere and never fuzzed
          detail: |-
            owner() in cmd/define/language_text.go:32 advances spanIndex monotonically
            and never rewinds, so out-of-order or overlapping source.spans silently
            lose ownership rather than failing visibly. languageText's doc comment
            (language_text.go:17) does not state the invariant, and
            FuzzDisplayProjection only builds a sorted two-span partition, so it never
            explores a violation. Both current producers do append in order; this diff
            made projectDisplayText the type's only consumer and moved it beside the
            type, which is the moment to write the precondition down.
          family: unwritten-precondition
          round: 1
        - id: BR-5
          severity: Minor
          title: Working tree carries uncommitted changes unrelated to this issue
          detail: |-
            .github/workflows/merge-check.yml, a Makefile type change, bootstrap.sh, a
            deleted scripts/issue-sync.sh and an untracked
            scripts/merge-checks.d/40-duplicate-issue-id.sh are modified but belong to
            no #76 commit. Keep them out of the close commit.
          family: unrelated-changes-in-close-window
          round: 1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — tools#76 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T13:59:31-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `name-outlives-referent` dictionary_language_test.go is named for a production file this window deleted
  dictionary_language.go was deleted because it "would hold nothing
  dictionary-related", but its test file keeps the name while holding only
  definition-tint tests (assertDictionaryTint and three TestDictionary*
  cases). Fold into definitions_output_test.go or rename.
- **BR-2** [Minor] `name-outlives-referent` TestDictionaryMonolingualOriginAndDisabledTint asserts neither origin nor tint
  cmd/define/dictionary_language_test.go:45 now only checks that Color:false
  emits no ANSI. #70 removed the tint half; this window removed the last
  Language: literal that gave "MonolingualOrigin" any meaning. Rename to what
  it checks, or delete it if another no-color assertion already covers it.
- **BR-3** [Minor] `atlas-describes-removed-surface` atlas/define.md:2350 still describes the deleted ro.Language write
  "renderDefinitionOutput retains section language ownership through
  rendering and region offsets" was the prose for `ro.Language =
  section.language`. What remains is `ro.Vocab = nil` for non-primary
  sections plus role-based row paint; a one-clause edit keeps the sentence
  true without naming a field that no longer exists.
- **BR-4** [Minor] `unwritten-precondition` projectDisplayText requires ascending non-overlapping spans, stated nowhere and never fuzzed
  owner() in cmd/define/language_text.go:32 advances spanIndex monotonically
  and never rewinds, so out-of-order or overlapping source.spans silently
  lose ownership rather than failing visibly. languageText's doc comment
  (language_text.go:17) does not state the invariant, and
  FuzzDisplayProjection only builds a sorted two-span partition, so it never
  explores a violation. Both current producers do append in order; this diff
  made projectDisplayText the type's only consumer and moved it beside the
  type, which is the moment to write the precondition down.
- **BR-5** [Minor] `unrelated-changes-in-close-window` Working tree carries uncommitted changes unrelated to this issue
  .github/workflows/merge-check.yml, a Makefile type change, bootstrap.sh, a
  deleted scripts/issue-sync.sh and an untracked
  scripts/merge-checks.d/40-duplicate-issue-id.sh are modified but belong to
  no #76 commit. Keep them out of the close commit.

## Open findings

- **BR-1** [Minor] `name-outlives-referent` dictionary_language_test.go is named for a production file this window deleted
- **BR-2** [Minor] `name-outlives-referent` TestDictionaryMonolingualOriginAndDisabledTint asserts neither origin nor tint
- **BR-3** [Minor] `atlas-describes-removed-surface` atlas/define.md:2350 still describes the deleted ro.Language write
- **BR-4** [Minor] `unwritten-precondition` projectDisplayText requires ascending non-overlapping spans, stated nowhere and never fuzzed
- **BR-5** [Minor] `unrelated-changes-in-close-window` Working tree carries uncommitted changes unrelated to this issue
