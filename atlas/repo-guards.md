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
