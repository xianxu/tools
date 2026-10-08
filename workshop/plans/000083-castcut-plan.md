# castcut Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

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

- [ ] Copy `../parley.nvim/demo/cut.py` → `cmd/castcut/testdata/cut_reference.py`; record parley.nvim `git rev-parse HEAD` in `testdata/README.md` with the regen command:
      `python3 cut_reference.py raw.cast captions.txt -o cut.golden.cast > cut.golden.stdout` (run in `testdata/`, so the printed path is the bare name; the golden test runs `run()` with the same relative `-o` from a temp dir holding copies).
- [ ] Hand-write `raw.cast` (≈30 events) exercising every row of the contract table: header with nested `term` object, int `timestamp`, float `idle_time_limit: 2.0`, a duplicate key, `env`; gaps above the cap; `o` data with ESC, `\r\n`, `\u0007`, `é→✓`, `"`, `\\`, U+2028; one `r` and one `i` event; an event whose output time collides with a caption start (marker-before-output ordering); a gap that yields a sub-1e-4 delta (exponent repr).
- [ ] `captions.txt`: unsorted lines, one without `~`, two overlapping, one long (> 4 s of words), one with `\x1f` padding, blank lines.
- [ ] Generate the goldens; eyeball that the golden contains `e-05`-style numbers and a `"captions"` header. Commit `#83 M1: castcut: cut.py oracle and fixture`.

#### Task 2: pyjson (TDD)

- [ ] Write `pyjson_test.go` table: decode→encode round trips (`{"b": 1, "a": [1.50, -0, 1e3, "x"]}` → `{"b": 1, "a": [1.5, 0, 1000.0, "x"]}`, duplicate keys, nested empty `{}`/`[]`, strings with every escape class); `pyFloat` rows (`0.0`, `-0.0`, `2.0`, `0.1`, `1e-05`, `0.0001`, `1e+16`, `1234567890123456.0`, `5e-324`); `pyRound` rows (`round(0.0078125,6)`→`0.007812`, `round(2.675,2)`→`2.67`, `round(-1e-9,6)`→`-0.0`). Expected strings come from running the noted Python one-liners.
- [ ] Run `go test ./cmd/castcut/ -run PyJSON` → FAIL (undefined).
- [ ] Implement:

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
- [ ] Tests green. Commit `#83 M1: castcut: Python-compatible JSON`.

#### Task 3: cast + captions parse (TDD)

- [ ] Tests: v2 header → `expected asciicast v3`; blank lines skipped; 2-element event → error; caption regex rows (`~1:02.5  hi` → 62.5, `0:05 x`, `1:2` no text → error with cut.py's message text), sort by (time, text), `pySpace` strip, `sidecarPath`.
- [ ] Implement `parseCast(path string, data []byte)` and `parseCaptions(path string, data []byte)` (regex `^~?(\d+):(\d+(?:\.\d+)?)\s+(.+)` on the stripped line; `At = m1*60 + m2` in that order).
- [ ] Green; commit `#83 M1: castcut: cast and captions parsing`.

#### Task 4: the cut (TDD) and CLI wiring

- [ ] `cut_test.go` pure rows: no captions → header gains `"captions": []`, all gaps squeezed by `min(1, idle/fast)/speed`; one caption → its window plays at rate 1 and the marker lands at `round(warp(start),3)`; overlapping captions start at the previous end; caption past the end → `caption at 9.9s is past the end of the recording (5.0s): <text>`.
- [ ] Port `plan_windows`, `build_segments` (cut set = sorted unique `{prev, t, bounds in (prev,t)}`), `warp`, and `main`'s assembly line by line; `idle_time_limit` popped before encoding.
- [ ] `main.go`: `run(args []string, stdout, stderr io.Writer) int`; `cut` flags via `flag.FlagSet` (`-o/--out` required, `--speed --idle --lead --min-hold --wps --beat`); flags may follow positionals (parse positionals out first, as argparse does). Errors → `castcut: <msg>` on stderr, exit 1. Summary lines exactly as cut.py prints.
- [ ] `cut_golden_test.go`: copy fixture into `t.TempDir()`, `chdir`, `run([]string{"cut","raw.cast","captions.txt","-o","cut.golden.cast"})`, compare file bytes and stdout to goldens; on mismatch print the first differing line. Mutation check: flip marker priority in the sort → test goes red; revert.
- [ ] `cut_diff_test.go` (`//go:build conformance`): 200 seeded random casts/captions (random gaps incl. 0 and > cap, random unicode/control data, random caption times incl. duplicates, random flag values), each written to temp, run via `python3 cut_reference.py` and via `run()`; byte-compare. `python3` missing → `conformance.SkipOrFail`.
- [ ] `go test ./cmd/castcut/` and `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/castcut/` (unsandboxed) green.
- [ ] Atlas: `atlas/castcut.md` (map: pipeline, byte-identity seam, file list) + link in `atlas/index.md`. Commit; `sdlc milestone-close --issue 83 --milestone M1`.

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
