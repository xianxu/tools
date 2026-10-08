# Boundary Review — tools#83 (milestone M2)

| field | value |
|-------|-------|
| issue | 83 — castcut: record, annotate and cut terminal demos |
| repo | tools |
| issue file | workshop/issues/000083-castcut-record-annotate-and-cut-terminal-demos.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | d48faad3b1d5c37234c80956aaaaf92ecfcc83cf..9da583d290ffb2f3f7dfa9ce512731fb9a1bc06b |
| command | sdlc milestone-close --issue 83 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-08T14:11:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers what the plan asks for. `record` is built from a pure argv builder (`recordArgs`, `shellJoin`, `nextTake`) and a thin exec layer, with a stateful fake asciinema plus a live conformance test that share one `recordContract`. `annotate` runs the real handler on a real listener and tests the Host, Origin and size guards. `help_test.go` reads every flag from each command's own `-h`, so a new flag without a manual entry fails the test. `go test ./cmd/castcut` and `go vet` pass, including `GOOS=linux go vet`. The SRI hashes in `viewer.html` match the jsDelivr 3.17.0 assets, which I fetched and hashed myself. Two things stop a plain SHIP:
1. **`record` misreports asciinema's own failures.** When asciinema itself fails, `record` treats that as the recorded command's exit status and still prints "take saved". I reproduced this.
2. **The in-browser check was skipped without a plan revision.** Plan Task 6 calls for a manual check in the browser; the Log moves it to M3 but no `## Revisions` entry records the change.

**Strengths**
- `cmd/castcut/record.go:26` — `recordArgs` always adds `--return`, and both the fake and real asciinema are held to the same `recordContract` (`record_test.go:93`, `record_conformance_test.go:13`). Real fake-plus-conformance coverage for ARCH-MOCK.
- `cmd/castcut/annotate.go:28-90` — the handler is built after `Listen` from the bound port, checks Host exactly on every route and Origin exactly on PUT, caps notes with `MaxBytesReader` and writes atomically. Each guard has a test row (`annotate_test.go:96-127`).
- `cmd/castcut/help_test.go:12` — the manual's flag coverage comes from the commands themselves, not from a hand-kept list.
- `viewer.html` keeps at most one PUT in flight and the latest text wins (PQ-6), sets captions via `textContent`, and pins the player with SRI.
- `cmd/castcut/help.md` names the "files and their lifetimes", the `--capture-input` password hazard, and the last-writer-wins rule for two tabs (ARCH-FUNERAL, ARCH-SECURE).

**Critical**
- None.

**Important**
1. **`record.go:132-138` blames the command for asciinema's own failures, and says the take was saved.**
   - Any non-zero exit from asciinema becomes `exitCode(ee.ExitCode())`, and "take saved to <out>" is printed whenever the output file exists.
   - Reproduced: running the conformance test in this environment, asciinema failed with `Error: EPERM: Operation not permitted`. castcut still printed `castcut: take saved to …/t.cast` and exited 1 as though the command had exited 1.
   - A process killed by a signal gives `ExitCode() == -1`, which leaves castcut as status 255 with no message.
   - This is the 2nd finding in family `error-message-misstates-cause`. The rule that covers both: a status line names a cause only from evidence castcut has read.
   - Fix sketch: after asciinema exits, `parseCast` the output and take the exit status from its `x` event, which asciinema 3 writes. Only a parseable take with an `x` event earns "take saved" and the command's status. Otherwise report `asciinema failed (exit N); <out> is incomplete` and exit 1.
   - Add a fake mode that fails before running the command, so a test pins this.
2. **Plan Task 6's manual browser check is not done, and the plan was not revised.**
   - The steps were: play, Alt+T, reload so the notes come back from the sidecar, and see the overlay on a cut cast.
   - The Log says in-browser behaviour has "not yet been exercised — M3's couch take does that". But the M2 boundary claims annotate is delivered, and the viewer JS has only been through `node --check`.
   - Fix: either run the check now (cheap with the operator's browser) or add a `## Revisions` entry that moves it to M3 explicitly.

**Minor**
- `viewer.html` (load IIFE): the textarea is editable and autosaves before `GET /notes` succeeds. If that load fails (or `/session` fails), typing PUTs over the existing sidecar.
  - This is the 6th finding in family `untrusted-input-fabricated-output`. Rule: a failed read never becomes a default that is later written back. Fix: keep the textarea disabled until the notes have loaded.
- Plan Task 7 asked for a test that `castcut --version` prints `castcut <version>`. There is none, though the command does work. M3's Homebrew `test do` will depend on it.
- `writeAtomic` makes the temp file with mode 0600, so the first save resets the sidecar's permissions. A crash between create and rename leaves a `.take.captions.txt.*.tmp` file that nothing removes (ARCH-FUNERAL residue; small).
- A recorded command that exits 2 can't be told apart from castcut's usage exit 2. The manual documents this, so it's acceptable.
- `fmt` in the viewer can display `0:60.0` because of `toFixed` rounding. It parses back to the correct value, so this is cosmetic only.

**Test coverage notes**
- The fake asciinema doesn't model asciinema failing before it runs the command, which is how Important #1 went uncaught.
- `runAnnotate`'s serve and shutdown path is only covered by the manual smoke run; the handler itself is tested properly.
- In this review environment, `CONFORMANCE_STRICT=1 go test -tags conformance -run RealAsciinema` failed with asciinema EPERM. That is an environment restriction, not the code; the Log reports it passing unsandboxed. The run did, however, demonstrate the misreport in Important #1.

**Architecture notes**
| Principle | Result | Why |
|---|---|---|
| ARCH-DRY | pass | |
| ARCH-PURE | pass | argv, quoting and numbering are pure; exec is thin |
| ARCH-PURPOSE | pass | `--capture-input` is passthrough only, as decided in D2 |
| ARCH-MOCK | pass | the fake and the real binary share one contract; the failure-mode gap is noted above |
| ARCH-CONSTRAINTS | pass | |
| ARCH-SECURE | pass | Host and Origin checks; the load-failure overwrite is the Minor above |
| ARCH-ORDER | pass | the viewer's save state machine is small, with explicit `saving`/`dirty` flags |
| ARCH-FUNERAL | pass | lifetimes are documented; only the temp-file residue is noted |

For M3, the Homebrew `test do` should cover `--version` and an offline `cut`.

**Plan revision recommendations**
- `## Revisions`: "M2 Task 6 manual browser check moved to M3 (couch end-to-end take); M2 verified the viewer only by `node --check` plus the server smoke run."
- `## Revisions`: "record derives the exit status and 'take saved' from the take's `x` event; an asciinema failure is reported as such."

```findings
findings:
  - id: new
    severity: Important
    family: error-message-misstates-cause
    title: |
      record reports asciinema's own failure as the command's status and claims the take was saved
    detail: |
      record.go:132-138 maps any asciinema non-zero exit to exitCode and prints "take saved" if the file exists. Reproduced: asciinema EPERM printed "take saved to t.cast" and exited 1; a signal gives 255 silently. Rule: a status line names a cause only from evidence castcut read. Derive status and the saved claim from parseCast plus the take's x event, else report "asciinema failed; out is incomplete". Add a fake failure mode as the regression test.
  - id: new
    severity: Important
    family: verification-deferred-past-boundary
    title: |
      Plan Task 6 manual in-browser check of the viewer not done and not revised in the plan
    detail: |
      The Log defers play/Alt+T/reload-persists/overlay to M3 but the plan has no Revisions entry; the viewer JS was only node --check'ed. Run the check now or record the move to M3 in Revisions.
  - id: new
    severity: Minor
    family: untrusted-input-fabricated-output
    title: |
      viewer autosaves before GET /notes succeeds, so a failed load can overwrite the sidecar
    detail: |
      6th finding in this family. Rule: a failed read never becomes a default that is later written back. Keep the textarea disabled until the notes load succeeds.
  - id: new
    severity: Minor
    family: plan-test-row-missing
    title: |
      castcut --version has no test, which Plan Task 7 called for
  - id: new
    severity: Minor
    family: artifact-without-removal-path
    title: |
      writeAtomic resets sidecar mode to 0600 and leaves a .tmp file if the process dies mid-write
```
