# castcut Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Read the `## Revisions` at the end first:** byte-identity, `pyjson` and the Python oracle are superseded (2026-10-08); the M1 tasks there replace Tasks 1–4.

**Goal:** One `castcut` binary that records (via asciinema), annotates (embedded browser viewer) and cuts (Go port of parley.nvim's `demo/cut.py`, byte-identical) captioned terminal demos, with `castcut --help` written as agent instructions.

**Architecture:** `cmd/castcut/` only — no `internal/` package (first consumer; not an external-service transport). The cut is a pure pipeline: parse cast → plan caption windows → piecewise rate segments → warp times → encode, with a small Python-compatible JSON encoder as the byte-identity seam. `record` builds an asciinema argv (pure) and execs it; `annotate` is a localhost-only `net/http` server over an embedded `viewer.html` and a notes sidecar file.

**Tech Stack:** Go 1.26 stdlib only (`encoding/json` decoder tokens, `net/http`, `embed`, `os/exec`). External runtime dependency: `asciinema` ≥ 3 (record only). Reference oracle for tests: `python3` running the vendored `cut.py`.

---

## Decisions (operator: confirm or redirect)

- **D1 — record syntax.** `castcut record [-o FILE] [--cols 95] [--rows 36] [--capture-input] [--idle-time-limit S] -- CMD [ARGS…]`.
  The issue sketched `-- <command> <out.cast>`; a trailing positional after `--` is
  ambiguous with the command's own args, so the output is a flag. Default `-o` is
  `recordings/take-NN.cast` under the cwd (next free NN, created on demand) — the
  "easy to find later" place. Never overwrites an existing file. Always passes
  `--output-format asciicast-v3` (cut requires v3).
- **D2 — `--capture-input` is passthrough only.** asciinema records `i` events; cut
  passes them through like any event. *Showing* keystrokes needs a renderer in each
  embed destination — out of scope here; noted in `--help` as the future axis.
- **D3 — annotate notes live in a sidecar, not localStorage.** With a server we can
  write `<take>.captions.txt` beside the cast (autosaved, debounced PUT). This deletes
  the viewer's 20-draft localStorage machinery and its legacy-key migration, and makes
  the notes file the literal input to `cut`. "Download notes" stays. The "Open .cast"
  file picker goes: an annotate session is one take.
- **D4 — `cut`'s captions argument defaults to that sidecar.**
  `castcut cut <take.cast> [captions.txt] -o <cut.cast>`; omitted → `<take>.captions.txt`.
- **D5 — the player still loads from jsDelivr** (asciinema-player 3.17.0, pinned) as
  today; `annotate` states that the page needs network for the player. Vendoring it
  (~0.5 MB embedded) is a later choice, not this issue's.
- **D6 — release.** New formula `Formula/castcut.rb` in `../homebrew-tools`, same
  shape as `define.rb`, `depends_on "asciinema"`, version stamped by
  `-X main.version`. Tags are repo-wide; castcut ships at the next tag (`v0.1.8`)
  while `define.rb` keeps its own pin. Tagging and pushing the tap are outward-facing:
  **ask the operator before each**.
- **D7 — parley.nvim follow-up** is a parley.nvim issue (`sdlc issue new` there), filed
  at M3; its README edit and `cut.py`/`viewer.html`/`test_cast_viewer.js` removal
  land in that repo's own flow.

## Byte-identity contract (what "same as cut.py" means)

Same input bytes + same flags → same output file bytes **and** same stdout summary.
Python behaviors the port must reproduce, each with a fixture row:

| Python behavior | Go emulation |
|---|---|
| `json.loads` → `dict` keeps key order; duplicate key keeps first position, last value | ordered `object` with in-place set |
| int literal stays int (`95`, `-0`→`0`); float → `repr` (`2.0`, `1e-05`, `1.5e+16`) | number kept as literal, classified like Python |
| `json.dumps(ensure_ascii=False)` separators `", "` / `": "`; escapes only `"` `\` `\n\r\t\b\f` and other C0 as `\u00xx` | hand encoder (`encoding/json` escapes `<>&`, U+2028) |
| `round(x, n)` correctly rounded, ties-to-even on exact binary ties (`round(0.0078125, 6)` = `0.007812`) | `FormatFloat('f', n)` → `ParseFloat` |
| `header["captions"] = …` replaces in place if the key already exists | ordered set |
| `header.pop("idle_time_limit")`; falsy (null/0) cap means no cap | same |
| `sorted(caps)` sorts by (time, text) | stable sort, byte compare (== code-point order) |
| `str.strip()` / `str.split()` whitespace (adds `\x1c`–`\x1f` over Go's `unicode.IsSpace`) | `pySpace` predicate |
| stable sort of stream by (time, marker-before-output) | `sort.SliceStable` |
| `f"{x:.1f}"`, `f"{x:6.1f}"` | `%.1f`, `%6.1f` |

**Declared divergences** (Python crashes; Go is stricter or kinder — documented in
`--help`, not tested for equality): non-ASCII digits/whitespace in caption stamps
(Go's `\d`/`\s` are ASCII), lone surrogates and invalid UTF-8 (Python fails to write
or read), `splitlines` on U+2028/`\x85` inside a JSON line (Python fails to parse).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `pyValue` / `object` / `decodeValue` / `encodeValue` | `cmd/castcut/pyjson.go` | new |
| `pyFloat` / `pyRound` / `pySpace` | `cmd/castcut/pyjson.go` | new |
| `Cast` / `Event` / `parseCast` | `cmd/castcut/cast.go` | new |
| `Caption` / `parseCaptions` | `cmd/castcut/captions.go` | new |
| `Timing` / `window` / `segment` / `planWindows` / `buildSegments` / `warp` / `Cut` | `cmd/castcut/cut.go` | new |
| `recordArgs` / `nextTake` | `cmd/castcut/record.go` | new |
| `sidecarPath` | `cmd/castcut/captions.go` | new |

- **pyjson** — Python-`json`-compatible decode/encode of arbitrary JSON values. Tests
  in `pyjson_test.go`, table-driven against strings produced by Python (the expected
  literals are pasted in, with the Python one-liner that produced each row in a comment).
  - **DRY rationale:** the one place Python's number and string formatting is
    emulated; header, events and markers all encode through it.
  - **Future extensions:** none planned; if v2 casts are ever cut, same encoder.
- **Cast** — `{Header *object; Events []Event}`, `Event{Time float64; Kind string; Data pyValue}`.
  Rejects non-v3 (`<path>: expected asciicast v3`), empty event lists, events that are not 3-element arrays.
- **Caption** — `{At float64; Text string}`; `parseCaptions` ports `read_captions`
  including its error text `<path>:<n>: expected `~m:ss.s  text`, got '<line>'`
  (Python `repr` of the line — single-quoted; use `pyRepr` for the common case and
  document the edge).
  `sidecarPath("x/take.cast")` = `x/take.captions.txt`.
- **Cut** — `Cut(c Cast, caps []Caption, t Timing) (out []byte, summary Summary, err error)`.
  `Timing{Speed, Idle, Lead, MinHold, WPS, Beat}` with cut.py's defaults
  (5, 1, 1, 4, 3.5, 1). Float operations in the same order as cut.py (sum left to
  right; `max`/`min` argument order) so results are bit-identical, not merely close.
  `Summary` carries total, view length and windows so `main` prints cut.py's lines.
- **recordArgs** — `(out string, o recordOpts, cmd []string) []string`: the asciinema
  argv; the command is joined with POSIX single-quote quoting into one `--command`
  string (asciinema runs it through a shell). `nextTake(dir)` scans `take-NN.cast`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runAsciinema` | `cmd/castcut/record.go` | new | `os/exec` → `asciinema` |
| `annotateServer` | `cmd/castcut/annotate.go` | new | `net/http`, sidecar file |
| `viewer.html` | `cmd/castcut/viewer.html` | new (port) | browser + jsDelivr player |
| `main` / `run(args, stdout, stderr) int` | `cmd/castcut/main.go` | new | CLI |
| reference oracle | `cmd/castcut/testdata/cut_reference.py` | new (vendored cut.py) | `python3` |

- **runAsciinema** — `exec.LookPath("asciinema")` (miss → `castcut: asciinema not found; brew install asciinema`, exit 1), stdio inherited, exit code propagated. Prints `castcut: recording <cmd> to <out> with asciinema (95x36)` on stderr first.
  - **Test fake:** a stateful fake `asciinema` shell script written into a temp dir
    prepended to `PATH`; it appends its argv to a log file and writes a minimal v3
    cast to its last argument — so the test drives the real `run([]string{"record", …})`
    entry through a real `exec`, and asserts both the argv and the file it left.
- **annotateServer** — binds `127.0.0.1:<port>` (`--port`, default 0 = free port),
  prints the URL, opens it with `open` unless `--no-open` (or not darwin). Routes:
  `GET /` viewer, `GET /cast` the take bytes, `GET /notes` sidecar text (missing → empty 200),
  `PUT /notes` atomic write (temp + rename in the cast's dir, body ≤ 1 MiB).
  Every request must carry `Host: 127.0.0.1:<port>` (DNS-rebinding guard); `PUT` additionally requires a same-origin `Origin` when present.
  Tests use `httptest` against the real handler constructor.
  - **Future extensions:** vendored player route (D5).
- **reference oracle** — cut.py copied verbatim from parley.nvim (provenance commit in
  `testdata/README.md`); it outlives parley's removal and regenerates goldens.

### Lifecycles (ARCH-FUNERAL)

- `recordings/take-NN.cast` — created by `record`; last needed when the operator has cut and published; removed by the operator (castcut never deletes). Bound: one file per take, a few MB each.
- `<take>.captions.txt` — created/overwritten by `annotate`; last reader is `cut`; dies with its take. Bound: ≤ 1 MiB (PUT cap).
- Temp file from atomic write — renamed or removed in the same request.
- The annotate server process — dies on Ctrl-C; holds nothing durable.
- Cut output — the operator's published artifact; castcut overwrites only the `-o` path given.

## File structure

```
cmd/castcut/
  main.go            subcommand dispatch, flags, --help/--version
  help.md            go:embed — the agent-instruction help text
  pyjson.go(_test)   Python-compatible JSON
  cast.go(_test)     asciicast v3 parse
  captions.go(_test) captions.txt parse, sidecar path
  cut.go(_test)      the timing model + encode
  cut_golden_test.go fixture → golden bytes via run()
  cut_diff_test.go   //go:build conformance — random casts vs python3 oracle
  record.go(_test)   argv, take numbering, exec
  annotate.go(_test) server
  viewer.html        go:embed — ported viewer
  testdata/
    README.md          provenance + regen command
    cut_reference.py   vendored cut.py
    raw.cast           hand-built fixture
    captions.txt
    cut.golden.cast    produced by cut_reference.py
    cut.golden.stdout
```

---

## Plan (milestones = review boundaries)

### M1 — `castcut cut`, byte-identical to cut.py

#### Task 1: oracle and fixture

- [x] ~~superseded 2026-10-08 (see Revisions)~~ Copy `../parley.nvim/demo/cut.py` → `cmd/castcut/testdata/cut_reference.py`; record parley.nvim `git rev-parse HEAD` in `testdata/README.md` with the regen command:
      `python3 cut_reference.py raw.cast captions.txt -o cut.golden.cast > cut.golden.stdout` (run in `testdata/`, so the printed path is the bare name; the golden test runs `run()` with the same relative `-o` from a temp dir holding copies).
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Hand-write `raw.cast` (≈30 events) exercising every row of the contract table: header with nested `term` object, int `timestamp`, float `idle_time_limit: 2.0`, a duplicate key, `env`; gaps above the cap; `o` data with ESC, `\r\n`, `\u0007`, `é→✓`, `"`, `\\`, U+2028; one `r` and one `i` event; an event whose output time collides with a caption start (marker-before-output ordering); a gap that yields a sub-1e-4 delta (exponent repr).
- [x] ~~superseded 2026-10-08 (see Revisions)~~ `captions.txt`: unsorted lines, one without `~`, two overlapping, one long (> 4 s of words), one with `\x1f` padding, blank lines.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Generate the goldens; eyeball that the golden contains `e-05`-style numbers and a `"captions"` header. Commit `#83 M1: castcut: cut.py oracle and fixture`.

#### Task 2: pyjson (TDD)

- [x] ~~superseded 2026-10-08 (see Revisions)~~ Write `pyjson_test.go` table: decode→encode round trips (`{"b": 1, "a": [1.50, -0, 1e3, "x"]}` → `{"b": 1, "a": [1.5, 0, 1000.0, "x"]}`, duplicate keys, nested empty `{}`/`[]`, strings with every escape class); `pyFloat` rows (`0.0`, `-0.0`, `2.0`, `0.1`, `1e-05`, `0.0001`, `1e+16`, `1234567890123456.0`, `5e-324`); `pyRound` rows (`round(0.0078125,6)`→`0.007812`, `round(2.675,2)`→`2.67`, `round(-1e-9,6)`→`-0.0`). Expected strings come from running the noted Python one-liners.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Run `go test ./cmd/castcut/ -run PyJSON` → FAIL (undefined).
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Implement:

```go
// pyFloat is Python's repr(float): shortest round-trip digits, fixed notation
// for decimal exponents in [-4, 16), scientific otherwise.
func pyFloat(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case math.IsNaN(f):
		return "NaN"
	case f == 0:
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	s := strconv.FormatFloat(f, 'e', -1, 64) // -d.ddde±XX
	sign := ""
	if s[0] == '-' {
		sign, s = "-", s[1:]
	}
	mant, expPart, _ := strings.Cut(s, "e")
	exp, _ := strconv.Atoi(expPart)
	digits := strings.Replace(mant, ".", "", 1)
	if exp < -4 || exp >= 16 {
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		}
		es := "+"
		if exp < 0 {
			es, exp = "-", -exp
		}
		return fmt.Sprintf("%s%se%s%02d", sign, m, es, exp)
	}
	if exp < 0 {
		return sign + "0." + strings.Repeat("0", -exp-1) + digits
	}
	if len(digits) <= exp+1 {
		return sign + digits + strings.Repeat("0", exp+1-len(digits)) + ".0"
	}
	return sign + digits[:exp+1] + "." + digits[exp+1:]
}

// pyRound is Python's round(x, n): correctly rounded, ties to even.
func pyRound(x float64, n int) float64 {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return r
}
```

  plus `object{keys []string; vals map[string]pyValue}` with `set` (in place),
  `get`, `del`; `decodeValue(*json.Decoder)` over `Token()` with `UseNumber`;
  `encodeValue(*bytes.Buffer, pyValue)`: `json.Number` without `.eE` → int text
  (normalize `-0`), else `pyFloat(ParseFloat)`; `float64` → `pyFloat`; strings via the
  ensure_ascii=False escaper (`\u%04x` lowercase).
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Tests green. Commit `#83 M1: castcut: Python-compatible JSON`.

#### Task 3: cast + captions parse (TDD)

- [x] ~~superseded 2026-10-08 (see Revisions)~~ Tests: v2 header → `expected asciicast v3`; blank lines skipped; 2-element event → error; caption regex rows (`~1:02.5  hi` → 62.5, `0:05 x`, `1:2` no text → error with cut.py's message text), sort by (time, text), `pySpace` strip, `sidecarPath`.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Implement `parseCast(path string, data []byte)` and `parseCaptions(path string, data []byte)` (regex `^~?(\d+):(\d+(?:\.\d+)?)\s+(.+)` on the stripped line; `At = m1*60 + m2` in that order).
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Green; commit `#83 M1: castcut: cast and captions parsing`.

#### Task 4: the cut (TDD) and CLI wiring

- [x] ~~superseded 2026-10-08 (see Revisions)~~ `cut_test.go` pure rows: no captions → header gains `"captions": []`, all gaps squeezed by `min(1, idle/fast)/speed`; one caption → its window plays at rate 1 and the marker lands at `round(warp(start),3)`; overlapping captions start at the previous end; caption past the end → `caption at 9.9s is past the end of the recording (5.0s): <text>`.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Port `plan_windows`, `build_segments` (cut set = sorted unique `{prev, t, bounds in (prev,t)}`), `warp`, and `main`'s assembly line by line; `idle_time_limit` popped before encoding.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ `main.go`: `run(args []string, stdout, stderr io.Writer) int`; `cut` flags via `flag.FlagSet` (`-o/--out` required, `--speed --idle --lead --min-hold --wps --beat`); flags may follow positionals (parse positionals out first, as argparse does). Errors → `castcut: <msg>` on stderr, exit 1. Summary lines exactly as cut.py prints.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ `cut_golden_test.go`: copy fixture into `t.TempDir()`, `chdir`, `run([]string{"cut","raw.cast","captions.txt","-o","cut.golden.cast"})`, compare file bytes and stdout to goldens; on mismatch print the first differing line. Mutation check: flip marker priority in the sort → test goes red; revert.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ `cut_diff_test.go` (`//go:build conformance`): 200 seeded random casts/captions (random gaps incl. 0 and > cap, random unicode/control data, random caption times incl. duplicates, random flag values), each written to temp, run via `python3 cut_reference.py` and via `run()`; byte-compare. `python3` missing → `conformance.SkipOrFail`.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ `go test ./cmd/castcut/` and `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/castcut/` (unsandboxed) green.
- [x] ~~superseded 2026-10-08 (see Revisions)~~ Atlas: `atlas/castcut.md` (map: pipeline, byte-identity seam, file list) + link in `atlas/index.md`. Commit; `sdlc milestone-close --issue 83 --milestone M1`.

### M2 — `record`, `annotate`, `--help`

#### Task 5: record (TDD)

- [ ] `record_test.go`: `recordArgs` rows (defaults → `rec --output-format asciicast-v3 --window-size 95x36 --command 'cmd' 'a b' out`; `--capture-input`, `--idle-time-limit`); `nextTake` over a dir with `take-01.cast`, `take-03.cast`, `notes.txt` → `take-04.cast`; existing `-o` file → refuse before exec.
- [ ] Wiring test with the fake `asciinema` on `PATH` (see Integration points): asserts argv log, the created `recordings/` dir and that the file the fake wrote exists; second test with an empty `PATH` → the not-found message and exit 1.
- [ ] Implement; commit.

#### Task 6: annotate server + viewer (TDD for the server)

- [ ] `annotate_test.go` via `httptest.NewServer(newAnnotateHandler(castPath, host))`: `GET /cast` bytes equal file; `GET /notes` empty when missing; `PUT /notes` then file content equals body and no temp file remains in the dir; `PUT` > 1 MiB → 413 and file untouched; wrong `Host` → 403 on every route; `GET /` serves HTML containing `fetch('/cast')`.
- [ ] Port `viewer.html`: keep player (pinned CDN), time readout, caption overlay + `placeCaption`, pause-at-captions, Alt+T stamp, Download; replace the cast param/file-picker with `fetch('/cast')`; replace localStorage drafts with `GET /notes` on load and a 500 ms-debounced `PUT /notes` on input, status line "Saved to <name>" / "Unsaved: <error> — download notes".
- [ ] `annotate` CLI: `castcut annotate [--port N] [--no-open] <take.cast>`; stderr: `castcut: serving <take> at http://127.0.0.1:N/ (notes → <sidecar>; the page loads asciinema-player from cdn.jsdelivr.net)`.
- [ ] Manual check (operator or me in the built-in browser if available): open, play, Alt+T twice, edit, reload page → notes persist from the sidecar; play a cut cast → captions overlay shows. Commit.

#### Task 7: `--help` as agent instructions

- [ ] `help.md` (embedded, printed by `castcut`, `castcut help`, `castcut --help`): what castcut is; prerequisites (asciinema, network for the player); the four steps with exact commands (record → annotate → cut → embed) and what each leaves on disk; the timing model and every cut flag with defaults; the **captions header contract** (`captions: [{start, end, text}]` in output seconds, one `m` marker per caption at `start`, idle_time_limit removed, consumers show a caption while `start ≤ t < end`); embedding: xianxu.dev (`public/casts/<name>.cast` + `<div class="cast-embed" data-cast="/casts/<name>.cast"></div>`, optional `data-poster="npt:0:30"`, rendered by `src/components/blog/CastEmbed.astro`), plain asciinema-player (markers only, or poll `getCurrentTime()` against header captions — the 12-line pattern), another destination (overlay recipe: CAPTION_BOX fractions); declared divergences from cut.py; `--capture-input` future axis; what stays per-app (demo launcher, shot list).
- [ ] Test: `run([]string{"--help"})` exit 0 and stdout contains each subcommand name and `captions`; each subcommand's `-h` names its flags. `castcut --version` prints `castcut <version>` (`built from source` default).
- [ ] README tools table + `### castcut` section; atlas update. Commit; `sdlc milestone-close --issue 83 --milestone M2`.

### M3 — ship and prove end to end

- [ ] `../homebrew-tools/Formula/castcut.rb` modelled on `define.rb`: `depends_on "asciinema"`, `depends_on "go" => :build`, ldflags version; `test do` asserts `castcut --version` and runs `castcut cut` on a tiny inline cast (no network). `brew audit`-style sanity: `brew install --build-from-source ./Formula/castcut.rb` locally after the tag exists.
- [ ] **Ask operator**, then tag `v0.1.8` (after merge to main) and push; fill url/sha256; **ask**, then push the tap; verify `brew install xianxu/tools/castcut` + `castcut --version`.
- [ ] File parley.nvim follow-up issue (D7) with the README replacement text drafted in it.
- [ ] End to end on couch broadcast (operator drives the take and annotation): `castcut record -- couch …` → `castcut annotate recordings/take-01.cast` → `castcut cut recordings/take-01.cast -o couch-v1.cast` → `castcut annotate couch-v1.cast` shows the captions. Record commands + durations in `## Log`.
- [ ] `sdlc close --issue 83 --verified '…'`.

## Revisions

### 2026-10-08 — fresh-eyes plan review (byte identity, record exit code)

Reason: the reviewer tested the contract against Python 3.14.5 / asciinema 3.2.1. `pyFloat`/`pyRound`
sketches matched on 300k random floats; these deltas fix what did not.

- **`sum()` is compensated.** CPython ≥ 3.12 `sum()` over floats is Neumaier-compensated
  (`sum([0.1]*10) == 1.0`). New pure entity `pySum` in `pyjson.go` porting CPython's
  algorithm (running sum + compensation `c`, `c` added at the end when non-zero and finite); used for
  `fast` in `buildSegments`. Test row: `[0.1]*10 → 1.0`, plus 1k random lists vs pasted Python output.
  `testdata/README.md` records the oracle's Python version; the differential test requires
  `python3 ≥ 3.12` (older → `SkipOrFail` with that reason).
- **Whitespace is one set.** `pySpace` is the single source for strip, split, *and* the caption
  regex's `\s` — the regex is built from the same class (`[\t-\r\x1c-\x20\x85\xa0\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}]`),
  with a test asserting the class and the predicate agree on every rune ≤ U+3000.
- **Universal newlines** for captions.txt: split on `\r\n`, `\r`, `\n` (line numbers in errors follow).
  Cast files keep `\n` splitting (cut.py uses `splitlines`, which also splits `\r`; asciinema never
  writes raw `\r` outside JSON strings, so split casts on `\r\n|\r|\n` too — same helper).
- **Version check is numeric** (`3.0` accepted, as in cut.py).
- **Decoder strictness:** reject trailing data after a line's JSON value; accept `NaN`/`Infinity`/`-Infinity`
  number tokens by pre-scanning (Python accepts them) — if this costs more than a few lines, declare it instead.
- **`pyRepr`** joins the pure-entity table (Python `str.__repr__`: prefer `'`, switch to `"` when the text
  has `'` and no `"`; escape `\\`, the quote, `\n\r\t`, other non-printables as `\xNN`/`\uNNNN`/`\UNNNNNNNN`,
  printability via `unicode.IsPrint` with spaces other than U+0020 non-printable). Rows for each branch.
- **Declared divergences, corrected list:** argparse unique-prefix flags (`--sp 2`) and glued `-oFILE` are
  not accepted (exact long flags, `-o FILE`, `--out=FILE`); `--speed`/`--wps` ≤ 0 are rejected up front
  (Python raises ZeroDivisionError) and the random-flag generator stays positive; non-ASCII digits in stamps
  (Python *accepts* them via `int('١')`; Go rejects).
- **Assembly details made explicit:** markers sort on the *rounded* caption start, output events on the
  unrounded warp; caption time is `float64(m1*60 in int) + m2`; no output file is created on any error;
  `ParseFloat` ErrRange → ±Inf (encodes `Infinity`); fixture gains a JSON `true`.
- **record:** `recordArgs` always adds `--return` so the recorded command's exit status propagates
  (asciinema 3 exits 0 otherwise). `nextTake` uses `%02d` as a minimum width, so `take-100` follows `take-99`.
- **annotate:** the handler is built after `Listen` with the bound port; the Origin check is an exact match
  on `http://127.0.0.1:N`; printed/opened URL is always `127.0.0.1`; the jsDelivr tags gain SRI
  `integrity` hashes; captions render via `textContent`.
- **Close evidence:** the parley.nvim Done-when bullet is satisfied by *filing* the follow-up issue; `--verified` says so.

### 2026-10-08 — drop byte-identity: castcut is a new tool, not a port (supersedes)

Reason (operator): `cut.py` was a quick prototype; castcut is a brand-new feature built from
it. Matching its bytes buys nothing a player can see and costs ~100 lines of Python emulation
plus a Python test dependency.

**Superseded:** the "Byte-identity contract" section, the declared-divergences list, the
previous revision's `pySum`/`pyRepr`/whitespace-set/universal-newline/decoder-strictness items,
the `pyjson` entity, the `cut_reference.py` oracle, `cut.golden.*`, and `cut_diff_test.go`.
The previous revision's `record` (`--return`, `nextTake` width) and `annotate` items stand.

**What `cut` is now.** Same timing model and flags as the prototype (`--speed --idle --lead
--min-hold --wps --beat`, same defaults), same output *contract*: asciicast v3, header
`captions: [{start, end, text}]` in output seconds, one `m` marker per caption at `start`
sorting before output at the same instant, `idle_time_limit` removed from the header. Encoding
is stdlib: `encoding/json` with `SetEscapeHTML(false)`; header kept as
`map[string]json.RawMessage` so unknown fields pass through untouched (key order is not part
of the contract); event times rounded to 6 decimals, caption bounds to 3 (file size and
readability, not fidelity). `cut.py`'s odd edges are free to change: caption parse uses Go's
`strings.TrimSpace`/`strings.Fields`; `--speed`/`--wps` ≤ 0 are rejected; no output file on error.

**Pure entities (replaces the table's pyjson/Cast/Caption/Cut rows):**

| Name | Lives in | Status |
|------|----------|--------|
| `Cast` / `Event` / `parseCast` / `encodeCast` | `cmd/castcut/cast.go` | new |
| `Caption` / `parseCaptions` / `sidecarPath` | `cmd/castcut/captions.go` | new |
| `Timing` / `window` / `segment` / `planWindows` / `buildSegments` / `warp` / `Cut` | `cmd/castcut/cut.go` | new |

**M1 tasks (replace Tasks 1–4):**

- [ ] Task 1 — cast + captions parse (TDD): non-v3 rejected; blank lines skipped; event not a
  3-element `[number, string, value]` → error naming the line; header round-trips unknown fields
  (`env`, `term`, a `true`); caption rows (`~1:02.5  hi` → 62.5, missing `~` ok, no text → error
  with path:line, sort by (time, text)); `sidecarPath`.
- [ ] Task 2 — the timing model (TDD), example rows: no captions → `"captions": []` and every gap
  squeezed to `min(1, idle/fast)/speed`; one caption → window at rate 1, marker at the window start;
  overlapping captions start at the previous end; caption past the end → error.
- [ ] Task 3 — properties (`cut_prop_test.go`, `testing/quick` or seeded random over gaps incl. 0
  and > idle cap, caption sets incl. duplicates/overlaps, flag values > 0), each a separate assertion:
  (a) event count and kinds/data preserved in order, markers = captions; (b) output times
  non-decreasing; (c) each window's output length equals its hold within 1e-6 (real time); (d) outside
  windows, any input gap maps to ≤ `idle/speed` + rounding; (e) captions sorted, non-overlapping,
  each `end − start ≥ min-hold − 1e-3`; (f) a marker precedes any output event at the same instant.
  Plus `FuzzCut` over raw bytes → never panics, either error or a cast satisfying (a)–(f).
  Mutation checks: drop the `prev_end` clamp → (e) red; sort markers after output → (f) red;
  ignore `idle` → (d) red. Revert each.
- [ ] Task 4 — CLI wiring: `run(args, stdout, stderr) int`; `castcut cut <take.cast> [captions.txt]
  -o out.cast` (captions default to the sidecar, D4); flags may follow positionals; summary on
  stdout (`out: 42.0s (view 180.3s), 5 captions` + one line per caption); errors `castcut: …` on
  stderr, exit 1, no output file. Wiring test drives `run` on a small checked-in `testdata/take.cast`
  + `take.captions.txt` and asserts the produced file parses and satisfies the properties — and that
  `parseCast` of the output yields the header captions `CastEmbed.astro` reads (`header.captions`
  array of `{start,end,text}` numbers/strings).
- [ ] Atlas `atlas/castcut.md` + index link; `sdlc milestone-close --issue 83 --milestone M1`.

### 2026-10-08 — plan-quality round (PQ-1…PQ-6)

- **PQ-1 — a window past the end holds the last frame.** The prototype truncated a caption
  stamped near the end (segments stop at the last event). castcut extends the view timeline to
  `max(last event, last window end)` and, when a window ends after the last event, appends one
  `[gap, "o", ""]` event at that window's output end so the player keeps showing the final frame
  for the whole caption. Properties (c) and (e) then hold unconditionally; property (a) becomes
  "input events preserved in order, plus at most one trailing empty `o` hold event". Example row:
  caption 1 s before the end with a 4 s hold → output ends at the window end, last event is the hold.
- **PQ-2 — workload envelope and linear passes.** Envelope: a take is up to ~10⁶ events
  (~100 MB; a 30-minute nvim take is ~10⁵), captions ≤ 10³; `cut` targets < 1 s for 10⁵ events
  and stays O(N + W log W) in memory and time. `warp` is applied as one monotone pass with a
  segment cursor over the already-sorted view times (and the sorted window bounds), never a scan
  from 0 per event; the "is this piece inside a window" check uses a window cursor, since windows
  are sorted and non-overlapping. Property tests include a seeded 2×10⁵-event cast asserting the
  properties plus a generous wall-clock bound (< 3 s under `-race`-free `go test`), and a unit row
  pinning the cursor `warp` against a naive reference `warp` on random inputs (the naive one lives
  only in the test).
- **PQ-3 — live conformance for the asciinema fake.** `record_conformance_test.go`
  (`//go:build conformance`) runs the real binary through `run([]string{"record", "--no-tty"…})`
  — concretely `asciinema rec --headless --return --output-format asciicast-v3 --window-size 95x36
  --command 'printf hi; exit 3' <tmp>/t.cast` as built by `recordArgs` (plus `--headless`, which
  `record` exposes as a hidden `--headless` flag for this and for CI) — and asserts exit code 3 and
  that `parseCast` accepts the output with `term.cols == 95`. asciinema absent →
  `conformance.SkipOrFail`. The fake's state model (argv log + minimal v3 file + exit code from an
  env var) is checked against the same assertions, so fake and real satisfy one table.
- **PQ-4 —** pointer added under the Goal.
- **PQ-5 — test strategy per risky function, not case lists.** `parseCast` and `parseCaptions`:
  `FuzzParseCast` / `FuzzParseCaptions` seeded with the example rows and malformed lines (truncated
  JSON, 2-element events, non-numeric times, stamps without text); invariant: error or a value whose
  re-encode re-parses equal. `Cut`: Task 3's properties + `FuzzCut`. `recordArgs`: one table (it is
  argv construction; enumeration is the spec). `annotateHandler`: route table + the Host/Origin/size
  guards, each mutation-checked (drop the Host check → its row red).
- **PQ-6 — notes writes are serialized.** The viewer keeps at most one `PUT /notes` in flight;
  edits during a request set a dirty flag and the latest text is sent when it returns (latest body
  wins, no reordering). The server writes atomically, so a second tab on the same take is
  last-writer-wins at whole-file granularity — stated in `--help`.

### 2026-10-08 — M1 boundary review (BR-1 and minors)

- **Durations are bounded at the door.** `maxSeconds` (one week) caps a take's cumulative
  length in `parseCast` (line-naming error) and in `Cut` (for hand-built casts), and every timing
  flag in `Timing.validate` (finite, within range). Two 1e308 gaps used to produce `[NaN, …]` with
  exit 0; `FuzzCut` no longer skips huge gaps — it carries that input as a seed and asserts the
  properties for anything accepted.
- **Property (f) is an example row, deliberately.** Marker times are rounded to 3 decimals, so a
  generic "marker precedes output at the same instant" property is either vacuous or flaky; the
  example row uses speed 1 to make the tie exact and is mutation-checked.
- **No rounding drift:** event intervals are differences of rounded absolute times.
- `encodeCast` uses `marshalNoEscape`; a missing default sidecar names `castcut annotate`.

### 2026-10-08 — M1 boundary review rounds 2–3 (BR-6, BR-8: one rule, not three fixes)

The same family surfaced three times (NaN output, int-overflowing stamp minutes, NaN-blind
re-check). The rule now lives in one predicate: `inRange(x) = x >= 0 && x <= maxSeconds`
(positive comparison, so NaN fails), used by every boundary — event interval and cumulative
length in `parseCast`, `idle_time_limit`, caption stamps (minutes bounded before multiplying),
every timing flag, and `Cut`'s re-check of each gap, the take length and each caption. Superseded
M1 Task 1–4 rows are struck. The atlas states the measured envelope (2×10⁵ events), not 10⁶.
- BR-12 (demoted past the round cap, fixed anyway): derived durations obey the same bound — `Cut`
  refuses when caption holds push the view past `maxSeconds` (tiny `--wps`), so castcut never
  writes a cast it would itself refuse to read.

### 2026-10-08 — M2 boundary review (BR-13, BR-14)

- **BR-13 — record's status comes from the take, not from asciinema's exit code.** The
  recorded command's status is the take's final `x` event (asciinema writes it, `"0"` on
  success). No parseable `x` → `asciinema <exit> without finishing the take; <out> is
  incomplete`, exit 1, no "saved" claim. Rule: a status line names a cause only from evidence
  castcut read. The fake gains a failure mode (header only, EPERM, exit 1); verified live too
  (sandboxed asciinema EPERM → the incomplete message).
- **BR-14 — the in-browser viewer check moves to M3**, folded into the couch end-to-end take
  (play, Alt+T twice, edit, reload → notes persist from the sidecar, then `annotate` on the cut
  → header captions overlaid). Reason: this session has no browser to drive; the server side is
  covered by handler tests and the real-binary smoke run, the JS by `node --check`. M3's close
  evidence must name this check explicitly.
- Minors folded in: the notes textarea stays disabled until `GET /notes` succeeds (a failed
  read never becomes an empty draft that autosave writes back); the sidecar keeps its mode
  (0644 for a new one) through the atomic write; annotate sweeps its own stale `.tmp` leftovers
  at start.
