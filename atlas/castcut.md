# castcut

Record, annotate and cut captioned terminal demos (`cmd/castcut/`). Grew out of
parley.nvim's `demo/cut.py` + `demo/viewer.html` prototype; the timing model and
flags carried over, the bytes did not. `castcut --help` is the operator/agent
manual; this is the map.

## Pipeline

```
record ──► take.cast ──► annotate ──► take.captions.txt ──► cut ──► cut.cast ──► embed
(asciinema)               (browser)    (~m:ss.s  text)       (pure)   (v3 + header captions)
```

## Files

| file | owns |
|---|---|
| `main.go` | subcommand dispatch, `run(args, stdout, stderr) int`, flags-after-positionals |
| `cast.go` | asciicast v3 parse/encode; header kept as raw fields so unknown keys pass through |
| `captions.go` | `~m:ss.s  text` parsing, the `<take>.captions.txt` sidecar name |
| `cut.go` | the timing model: `planWindows` → `buildSegments` → `warper` → `Cut` |
| `record.go` | asciinema argv (`recordArgs`, always `--return`), `take-NN` numbering, exec |
| `annotate.go` | 127.0.0.1 server: `/` viewer, `/session`, `/cast`, `GET`/`PUT /notes` → sidecar (atomic write) |
| `viewer.html` | embedded viewer (asciinema-player 3.17.0 from jsDelivr, SRI-pinned); Alt+T stamps; one PUT in flight |
| `help.md` | embedded manual (`castcut --help`); `help_test.go` derives every flag from each command's `-h` |

## The timing model (`cut.go`)

View time = the take's time with its `idle_time_limit` applied (what the
annotate clock shows). Each caption gets a real-time window
`[stamp − lead, + max(min-hold, words/wps + beat))`, pushed after the previous
window if they would overlap. Outside windows, each inter-event gap is squeezed
to at most `idle` seconds and sped up `speed`×. A window that outlasts the take
holds the final frame with one trailing empty `o` event.

Linear in events: segments are built with cursors over sorted times and window
bounds, and `warper` answers each lookup by binary search over cumulative
output time. Measured: 2×10⁵ events + 10³ captions cut in under 1 s (`TestCutStaysInsideItsEnvelope`, 3 s budget); a 30-minute nvim take is ~10⁵ events.

## Output contract

asciicast v3; header gains `captions: [{start, end, text}]` in output seconds
(3 decimals) and loses `idle_time_limit`; one `m` marker per caption at `start`,
sorted before output at the same instant. Consumer:
`xianxu.dev/src/components/blog/CastEmbed.astro`.

## Seams

- **asciinema** (`record`): `record_test.go`'s fake runs `--command` through `sh`
  and honours `--return`/`--window-size`; `record_conformance_test.go`
  (`-tags conformance`, unsandboxed) holds real asciinema to the same
  `recordContract`.
- **browser** (`annotate`): the handler answers only `Host: 127.0.0.1:<port>`
  (DNS rebinding), writes only from that origin or none (CORS preflight blocks
  cross-origin PUTs anyway), caps notes at 1 MiB. Tests run the real handler on a
  real listener; the viewer's JS is checked by hand (no browser in CI).

## Tests

`cut_prop_test.go` pins the model as properties (events preserved, monotone,
windows real-time, idle bound, captions disjoint ≥ min-hold) over random takes,
a 2×10⁵-event envelope run, a naive-`warp` reference, and `FuzzCut`;
`main_test.go` drives `run` on `testdata/take.cast`.
