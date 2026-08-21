# tools atlas

Current-state map of this repo. One entry per binary, plus the inherited
workflow layer. This is a map, not a changelog — history lives in
`workshop/plans/` and git.

## Binaries

- [define](define.md) — NOAD word lookup with Google-style IPA and spoken pronunciation.

## Repo-wide

- [repo-guards](repo-guards.md) — invariants owned by the repository, not by a
  binary: no executable image in the index or reachable from `HEAD`. They live in
  `cmd/define/repo_guard_test.go` for want of a package of their own.

## Inherited

- [workflow/](workflow/) — the ariadne SDLC layer, delivered by `weave`. Not
  authored here; see `../ariadne/atlas/`.
