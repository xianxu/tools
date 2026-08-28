# Repo-wide guards

Invariants that belong to the repository rather than to any one binary, enforced
as ordinary Go tests.

They physically live in `cmd/define/repo_guard_test.go` because `define` is
currently the only package, and a test needs a package to live in. **They are not
about `define`** — a contributor who adds `cmd/foo` and leaves a build artifact
there will see `cmd/define`'s suite fail. If a second binary lands, move them to
their own package rather than leaving the surprise in place.

## No executable image is ever tracked, or ever reachable

`go build ./` inside `cmd/<name>/` writes an extensionless binary named after
that directory. `bin/` is the sanctioned output; a 9.6 MB `cmd/define/define`
reached a commit here when `git add -A` swept one in.

Two tests, because the mistake costs money in two different places:

| test | reads | catches |
|---|---|---|
| `TestNoCommittedBinaries` | the **index** (`git ls-files -s`) | an artifact staged or committed — the last moment it is free |
| `TestNoBinariesInHistory` | **history** (`git rev-list --objects HEAD`) | an artifact still reachable from `HEAD`, which every clone fetches |

The split is the whole point. Deleting a binary in a *follow-up commit* leaves
the index clean and the blob reachable: this branch once cloned to 5.9 MB against
`main`'s 604 KB while `git ls-files` reported the file zero times and an
index-only guard was green. The remedy is to rewrite the commit that **adds** it
(`git filter-branch --index-filter` over `main..HEAD`), while the branch is still
unpushed.

Both decide by **magic bytes** — Mach-O in either endianness, 32- or 64-bit, or
universal, and ELF — not by filename. An earlier version tested "extensionless
file in a source directory" and false-positived on a tracked symlink.

Three properties they hold deliberately, each learned from a version that lacked
it:

- **They fail rather than skip.** A guard that reports nothing when it cannot run
  certifies nothing. Every `git` invocation's exit status is checked.
- **They refuse to pass vacuously.** Each asserts it had something to examine.
- **They assert they consumed the whole work list.** `scanForExecutables` compares
  records read against records requested, because a partial scan reporting a
  clean result is indistinguishable from a clean repo. Feeding `cat-file` only
  the first five objects once left the history guard green with a planted binary
  present; it now fails with `scanned 5 of 761 objects`.

## No runtime state is ever tracked, or ever reachable

`define` writes its deck to the CURRENT directory, and a test's cwd is the
PACKAGE directory — so `go test` and the pty suite created a deck at
`cmd/define/words/` and `cmd/define/events/`, which the root-anchored ignore
patterns did not match, and `git add -A` committed somebody's vocabulary across
five commits.

| test | reads | catches |
|---|---|---|
| `TestNoTrackedRuntimeState` | the **index** | a deck file staged or committed |
| `TestNoRuntimeStateInHistory` | **history** | deck blobs still reachable from `HEAD` |

**Directories were only half of it (`#23`).** `store.RuntimeDirs` single-sources
`words/`, `events/` and `usage/` into all three places; it structurally could not
see a runtime *file*, so `user-model.md` — which `--reflect` writes into the
current directory, carrying inferred claims about the learner — reached none of
them. `git check-ignore -v user-model.md` matched nothing. Nothing leaked, but
`#23`'s `lang.txt` would have been the second instance, so `store.RuntimeFiles`
is the sibling list and `TestGitignoreCoversRuntimeFiles` plus an
`isRuntimeFile(basename)` arm on both guards close the same three places.

**A runtime basename is RESERVED**, and that falls out of the un-anchored
patterns rather than being an extra rule: a tracked file sharing one of these
names is silently un-addable after a `git rm`. The golden fixture at `cmd/define/testdata/golden/` was squatting on exactly
that and is now `user-model.golden.md`; the index guard is what says so out loud
if it happens again.

That also decided the setting's *name*. `lang.txt` rather than `lang`, because a
bare un-anchored `lang` would additionally hide any **directory** of that name
anywhere in the tree — and a basename guard cannot see that. The extension costs
nothing and closes the hole.

**The entries are gitignore-style PATTERNS**, because two of the four are
families rather than files: the learner model is per-language
(`user-model.es.md`), and `writeBytesAtomic` leaves a `.tmp-*` shadow beside
whatever it writes — in the working-directory ROOT for these two, where no
runtime directory covers it. `isRuntimeFile` uses `filepath.Match`, whose `?`
and `*` agree with gitignore's for a single path element.

`user-model.??.md` rather than `user-model*.md`, deliberately: the looser pattern
would shadow `testdata/golden/user-model.golden.md` and quietly re-break the
reserved-basename rule the rename above established. `??` is exactly a two-letter
`Lang`, which `ParseLang` guarantees.

Because a pattern cannot name a file, the writers no longer derive their names
from the list. `TestRuntimeFilePatternsCoverWhatWeWrite` keeps the two in step
instead, and it is the stronger check: it asserts the real output of the
functions that write, so a name drifting away from its pattern fails there rather
than silently escaping `.gitignore` and both guards.

## No document spells a name the code owns

The second rule this range added, and the one that took three review rounds to
state mechanically rather than in prose. A doc that spells a stale runtime
filename is not a style problem: the README tells the learner to hand-edit the
learner model's `## Corrections`, so a doc naming the wrong file sends their
corrections somewhere nothing reads.

| test | reads | catches |
|---|---|---|
| `TestRuntimeArtifactNamesAreSpelledOnceInSource` | non-test **Go** | a filename spelled outside the two consts that build every such name |
| `TestProseDoesNotSpellStaleRuntimeArtifactNames` | README, `atlas/`, `workshop/projects/` | a stale filename in a doc read as current truth |
| `TestPlanTablesNameEntitiesThatExist` | active plans' Core-concepts tables | a plan naming a SYMBOL the tree does not declare |
| `TestNoArtifactNamesARetiredSymbol` | non-test Go, README, `atlas/`, active plans | a RENAMED symbol surviving anywhere read as current |
| `TestTheCResolverAndTheGoSymbolListAgree` | the cgo preamble vs `dcsPrivateSymbols` | a `dlsym` the conformance check does not cover |
| `TestCaptureScriptUsesTheCuratedDictionaries` | `capture.sh` vs `curated` | fixtures captured from a dictionary production never asks |

**Records are exempt, and identifying them is the interesting part.** Docs that
describe the tool as it IS must be swept; docs that RECORD what was true when
written must not, or the "fix" is falsifying history. `currentTruthOnly` finds
records by SHAPE — a `## Revisions` or `## Log` heading, or a `###` block
carrying `**closed:**` — rather than by a list of filenames, so a new record
section is covered without anyone remembering it. Issues, lessons and `workshop/history/` are records
wholesale and are not swept at all. **Active plans are the exception**, and it is
worth stating: their Core-concepts tables and their live prose ARE swept — by
`TestPlanTablesNameEntitiesThatExist` and `TestNoArtifactNamesARetiredSymbol` —
because a plan's design sections describe the design as it IS, while everything
from its `## Revisions` heading on is a record and is skipped.

**Rows three to six are the SYMBOL half**, which recurred NINE times while the
first two ratchets counted filenames only. Table rows were the tractable part; plans named `deckDeps`, `MigrateFlatDeck`,
`dictChoice` and `dcsDictionaries`, none of which the tree had. A plan's
Core-concepts table already states "this identifier lives at this path" in
machine-readable form, so making the plan a CONSUMER of the tree is cheap. A row
whose file does not exist yet is skipped — a plan legitimately precedes its code;
a row pointing at a real file that does not declare the name is a lie.

**`retiredSymbolNames` is where the rule needs a human.** A rename cannot be
detected automatically — only the person doing it knows the old name — so
renaming adds one row there, and everything after that is mechanical. It is what
caught three `newDeck` comments left behind by the very commit that added the
table guard.

**The last two rows are the same move applied to boundaries Go cannot see
into.** A cgo preamble and a shell script cannot import a Go declaration, so the
two copies are COMPARED by reading the source rather than unified. Both close a
gap that had already bitten: a `dlsym` the conformance check did not cover, and
a corpus captured through a dictionary production does not consult.

**The history arm carries a ratchet, not an exemption.** Two 995-byte blobs of
the renamed fixture stay reachable from `HEAD`, and rewriting history for
synthetic sample output would be disproportionate. `legacyRuntimeFilePaths` pins
them as an exact set — per this repo's rule that a known limitation is a ratchet
rather than a comment — so a *second* such path fails immediately and the map can
only ever shrink.

Same two-place split as the binary guards, and for the same reason — the first
version of the runtime-state guard checked only the index, which is precisely the
half-fix that left a 9.6 MB binary reachable in `#4` after its file was removed.
The patterns in `.gitignore` are **un-anchored** for this class (`words/`,
`events/`), because anchoring only covers the root and tests do not run there.

The root cause is fixed too, not only guarded: `pty_conformance_test.go` gives
the child an explicit `cmd.Dir` of a `t.TempDir`, so the conformance flow no
longer writes into the source tree at all.

### Why `.gitignore` cannot carry this alone

`.gitignore` has no backreferences, so "a file named after its parent directory"
is inexpressible, and an un-anchored `define` would ignore the `cmd/define/`
**source** directory. The patterns there are therefore necessarily per-tool and
only keep `git status` quiet for the tools that exist. A new `cmd/foo`'s binary
shows up as untracked until someone adds a line — the tests are what make that
non-silent.

### Verifying a change to them

Plant what they hunt; a passing guard on a clean tree is evidence about nothing.
The cycle, in a throwaway clone: clean → **pass**; binary staged in a *new*
`cmd/newtool/` → index guard **fails**; `git rm --cached` plus a later commit →
index guard **passes** while the history guard **fails**. Run with `-count=1`, or
Go's test cache will answer a question about the previous source.
