# Free-form Q&A in the console — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Issue:** [tools#16](../issues/000016-console-qa.md) · **Project:** [define-learn](../projects/define-learn.md)

**Goal:** Typing a question at the `define` prompt answers it — no mode, no prefix
to remember — with the working directory (deck, recent lookups, `user-model.md`)
as the context.

**Architecture:** One decision table in two pure halves around the *free* signal
the tool already pays for. `parseREPLLine` stays the syntactic half (command /
blank / forced-question / forced-literal / word). The semantic half is
`readsAsQuestion`, a pure predicate consulted at exactly one place —
`lookupAndRender`'s miss branch, where NOAD has already answered "I have no entry
for this" at no extra cost. Neither loop gains a parser (ARCH-DRY); both learn one
new outcome. The answer itself streams through `internal/llm`'s existing `Stream`
seam, with the prompt assembled by a pure renderer from a pure `askContext`
struct, gathered by a thin IO step (ARCH-PURE).

**Tech Stack:** Go, `internal/llm` (transport + wire-level `llmtest` fake),
`cmd/define/store` (YAML + `Mem`, one conformance suite), the existing raw-mode
editor.

---

## Decisions

Each of these is a fork the implementation should not silently re-take.

**D1 — The NOAD lookup is not repeated.** The classifier's signal is "does NOAD
have an entry", and for an ordinary word that lookup *is* the feature. So routing
happens inside `lookupAndRender`, after its single `d.dict.Lookup` call, rather
than in a pre-pass that would look every line up twice (ARCH-DRY). This also keeps
`lookupAndRender` the one capture site, an invariant `capture_test.go` pins.

**D2 — A question is never captured as a lookup.** The route decision sits
*before* `d.capture.Capture(word, false, opt)` in the miss branch. A question
recorded as a not-found lookup would poison `#8`'s statistics and `#17`'s model
with events that are not lookups at all.

**D3 — Two escape hatches, one keystroke each, neither exclusive.**

| hatch | means | why this character |
|---|---|---|
| `?` in column 1 | force a question, skip NOAD entirely | mirrors `/`; no English headword starts with `?` |
| `\` in column 1 | force a literal lookup; a miss stays a miss | the shell's own "take this literally"; no headword starts with `\` |

Neither is the only way to reach its outcome: a bare question still asks, a bare
word still looks up. `\` matters for a phrase you believe is a headword and want
the dictionary's verdict on — `\how so` answers "not found" instead of chatting.

**D4 — `readsAsQuestion` has four arms, and only ever sees NOAD misses.** That
containment is what makes it safe: `hot dog`, `use`, `a priori` never reach it.

| arm | example | why |
|---|---|---|
| trailing `?` | `is it pejorative?` | the explicit mark |
| first word is interrogative/auxiliary | `what's the difference to obsequious` | the wh/aux opener |
| first word is a request verb **and** ≥2 words | `use it in a sentence`, `give me three more examples` | a follow-up is imperative, not interrogative |
| ≥5 words | `difference between sycophantic and obsequious please` | a five-word line NOAD does not know is not a headword typo |

`sycophanti` fails all four → not-found, unchanged.

**D5 — Ctrl-C mid-stream cancels the question, not the session.** Today
`readKeys` calls the session `cancel()` the moment it decodes an interrupt, which
is load-bearing during playback (the loop is blocked inside `speak` and cannot
read the key itself). During a stream the loop is *not* blocked — it selects on
keys — so the same byte must mean something narrower. One mechanism covers both:
`readKeys` fires a swappable sink whose default is the session cancel and which
the ask path temporarily points at the question's own `context.CancelFunc`.

**D6 — Stream raw, translate newlines.** "Render cooked, play raw" (#14) cannot
extend to a stream: deltas arrive continuously and flapping the terminal per
delta is not a thing. So the ask path stays in raw mode and writes through a
`crlfWriter` that maps `\n` → `\r\n`. This is also what keeps the key reader
seeing bytes, which D5 needs.

**D7 — The store owns `user-model.md`.** It already owns the directory
(`words/`, `events/`); the learner model is the third artifact in it, and `#17`
will write it through the same seam. `Store` gains `UserModel() (string, error)`
returning `""` when absent — one method, both implementations, rows in
`storetest.Suite` (ARCH-MOCK). Reading it through `os.ReadFile` at the call site
would put a second answer to "which directory is this session's" beside
`openStore`.

**D8 — An `asked` event records the question, not the answer.** `#17` needs to
know what the learner asked about; answers would bloat an append-only log whose
every consumer is a fold. `ReviewEvent` gains `Question string`, and `complete()`
generalises from "has a word" to "has a subject" — a word or a question.
`At` stays the **last** field in the struct, because the torn-record rule leans on
a cut record losing its timestamp (see `atlas/define.md`, "The store").

**D9 — Prompts live here, not in `internal/llm`.** `AGENTS.local.md`: the
transport owns the wire, the consumer owns the domain. `askctx.go` holds the
prompt.

**D10 — No typed task.** An answer is prose, so this uses `Client.Stream`
directly rather than `llm.Run[T]`. Nothing here has a schema.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `readsAsQuestion` | `cmd/define/question.go` | new |
| `truncateQuestion` | `cmd/define/question.go` | new |
| `parseREPLLine` / `replCommand` | `cmd/define/repl.go` | modified |
| `askContext` | `cmd/define/askctx.go` | new |
| `renderAskPrompt` | `cmd/define/askctx.go` | new |
| `recentTurns` | `cmd/define/askctx.go` | new |
| `session` | `cmd/define/session.go` | new |
| `crlfWriter` | `cmd/define/crlf.go` | new |
| `ReviewEvent` / `complete` | `cmd/define/store/event.go` | modified |

- **readsAsQuestion** — the semantic half of the decision table: does this line,
  which NOAD does not know, read as a question. Four arms (D4), no IO, no state.
  - **Relationships:** consulted from exactly one call site
    (`lookupAndRender`'s miss branch) plus the one-shot ask route.
  - **DRY rationale:** the alternative is a `strings.HasSuffix(line, "?")` at each
    loop, which is the two-parsers mistake this repo has paid for twice
    (`lessons.md`, define #14; `atlas/define.md`, "Command mode").
  - **Future extensions:** the wh/aux and request-verb sets widen for `#18`
    (Spanish `qué`, `cómo`, `por qué`) — they are package-level slices for exactly
    that reason.

- **truncateQuestion** — elides a question for a one-line diagnostic
  (`` `what's the difference…` ``), rune-safe at 40 runes.
  - **DRY rationale:** first occurrence; the degradation message and the event
    warning both use it.

- **replCommand** — gains `cmdAsk` and the `literal` flag (D3). `question` holds
  the text with the `?` stripped.
  - **Relationships:** produced by `parseREPLLine`, consumed by all three entry
    modes (one-shot, line loop, raw editor).

- **askContext** — everything the model is told, as data: `CurrentWord`,
  `CurrentEntry`, `SessionWords`, `DeckWords`, `UserModel`, `Turns`.
  - **Relationships:** 1:1 with a question; built by `gatherAskContext` (IO),
    rendered by `renderAskPrompt` (pure).
  - **DRY rationale:** one struct is what makes "what did we send" assertable
    without a network — the golden test renders it directly.
  - **Future extensions:** `#10`'s authoring prompt wants the same deck-and-model
    context; when it lands, this struct is what it reuses.

- **session** — the per-session state both loops carry: `current` word, `entry`
  text of that word, and `turns` (the Q&A transcript).
  - **Relationships:** one per loop invocation; replaces the bare `current string`
    both loops declare today.
  - **DRY rationale:** without it, "the last few words + the last exchange" gets
    declared twice, once per loop, and the two drift.

- **crlfWriter** — an `io.Writer` decorator mapping `\n` → `\r\n` for raw-mode
  output (D6).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `gatherAskContext` | `cmd/define/ask.go` | new | store reads |
| `runAsk` | `cmd/define/ask.go` | new | `llm.Client.Stream` |
| `deps.newLLM` / `deps.getenv` | `cmd/define/main.go` | new | `llm.New` + `llm.Resolve` |
| `Store.UserModel` | `cmd/define/store/{store,yaml,mem}.go` | modified | `user-model.md` on disk |
| `interrupter` | `cmd/define/rawterm.go` | new | the raw key reader's cancel |
| `askCapturer` (on `Capturer`) | `cmd/define/capture.go` | modified | event append |

- **gatherAskContext** — reads the deck, the user model and the session's own
  state into an `askContext`. Thin: no formatting, no truncation decisions beyond
  the counts, all of which are the pure renderer's.
  - **Injected into:** nothing — it *produces* the pure entity. Tests build an
    `askContext` literal and never call it.

- **runAsk** — resolves the config, builds the client, streams the answer to a
  writer, records the event. The only place a network call is made.
  - **Injected into:** both loops and the one-shot path, through `deps`.
  - **Future extensions:** `#10`'s harvest wants the same resolve-and-build; when
    it does, `deps.newLLM` is already the seam.

- **Store.UserModel** — reads `user-model.md` from the store's directory (D7).
  Absent file → `("", nil)`, because "no model yet" is the normal first-run state
  and not an error.
  - **Injected into:** `gatherAskContext`. `Mem` holds it in a field, so the
    conformance suite runs both.

- **interrupter** — a mutex-guarded holder of the current cancel func (D5).
  `Set(fn)` returns a restore func; `Fire()` calls whatever is installed.
  - **Injected into:** `readKeys`, replacing its `context.CancelFunc` parameter.

**Test surface.** Every pure entity above gets a colocated table test that runs
with no fake at all. The integration points are exercised against the existing
fakes: `fakeDictionary` (committed NOAD corpus), `llmtest`'s **wire-level**
httptest server (never a stubbed `Client` — placement is the whole point, see
`atlas/llm.md`), and `store.Mem` + `YAML` through `storetest.Suite`. One new
end-to-end test drives the **raw editor loop** (`runEditor` with a scripted key
channel), because a wiring only a loop shell supplies must be pinned by a test
that drives that loop shell (`lessons.md`, define #15).

---

## Milestones

Two review boundaries, genuinely separate (AGENTS.md §3):

- **M1 — the console knows a question from a word.** Classification, both
  hatches, routing through all three entry modes, and the honest degradation
  message when no model is configured. Ships standalone: no network, no store
  change, and every question gets the "no model configured" answer.
- **M2 — the answer.** Context pack, streaming, raw-mode Ctrl-C, the `asked`
  event, multi-turn.

---

## Chunk 1 — M1: classification and routing

### Task 1: `readsAsQuestion` and `truncateQuestion`

**Files:**
- Create: `cmd/define/question.go`
- Test: `cmd/define/question_test.go`

- [ ] **Step 1: Write the failing test**

```go
package main

import "testing"

func TestReadsAsQuestion(t *testing.T) {
	// Every row here is a line NOAD MISSED — that containment is the whole
	// safety argument, so a row for a real headword would be a lie about what
	// this function is asked.
	for _, tc := range []struct {
		name string
		line string
		want bool
	}{
		{"trailing question mark", "is it pejorative?", true},
		{"wh opener", "what's the difference to obsequious", true},
		{"contracted aux opener", "isn't that the same as fawning", true},
		{"a wh word ending in n is not a negation", "when is it used", true},
		{"request verb with an object", "use it in a sentence", true},
		{"follow-up request", "give me three more examples", true},
		{"five words is not a headword", "difference between sycophantic and obsequious please", true},
		{"a typo is not a question", "sycophanti", false},
		{"a bare request verb is a word", "give", false},
		{"a wh word alone is a word", "what", false},
		{"a two-word phrase is a headword shape", "amuse bouche", false},
		{"punctuation only", "?", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readsAsQuestion(tc.line); got != tc.want {
				t.Errorf("readsAsQuestion(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestTruncateQuestion(t *testing.T) {
	long := "what is the difference between sycophantic and obsequious"
	got := truncateQuestion(long)
	if []rune(got)[len([]rune(got))-1] != '…' {
		t.Errorf("truncateQuestion(%q) = %q, want an ellipsis", long, got)
	}
	if n := len([]rune(got)); n > 41 {
		t.Errorf("truncateQuestion returned %d runes, want <= 41", n)
	}
	if got := truncateQuestion("why"); got != "why" {
		t.Errorf("truncateQuestion(%q) = %q, want it unchanged", "why", got)
	}
	// Rune-safe: cutting mid-rune would print a replacement character.
	if got := truncateQuestion(strings.Repeat("é", 60)); !utf8.ValidString(got) {
		t.Errorf("truncateQuestion produced invalid UTF-8: %q", got)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run 'TestReadsAsQuestion|TestTruncateQuestion' -v`
Expected: FAIL — `undefined: readsAsQuestion`.

- [ ] **Step 3: Implement**

```go
package main

import (
	"strings"
	"unicode"
)

// questionOpeners are the words that make a line read as a question when it
// leads. Contractions are matched on the stem before the apostrophe ("what's"),
// and negated auxiliaries on the stem before "n't" ("isn't"), so the set stays
// the vocabulary rather than its inflections.
//
// A package-level slice rather than an inline switch because #18 widens it with
// Spanish (qué, cómo, por qué) and a switch would have to be found first.
var questionOpeners = []string{
	"what", "why", "how", "when", "where", "which", "who", "whom", "whose",
	"is", "are", "was", "were", "am", "do", "does", "did", "have", "has", "had",
	"can", "could", "should", "would", "will", "shall", "may", "might", "must",
}

// requestVerbs open an imperative that is still a question to this tool: a
// follow-up ("give me three more examples") has no wh-word and no question mark.
// They need a second word — bare "use" and "give" are headwords, and NOAD
// answers them before this function is ever consulted.
var requestVerbs = []string{
	"give", "show", "tell", "explain", "compare", "contrast", "use", "make",
	"write", "list", "translate", "rewrite", "define",
}

// longLineWords is the backstop arm: a line this long that NOAD does not know is
// not a headword the user expects to exist. NOAD's own long entries — proverbs,
// "the proof of the pudding is in the eating" — HAVE entries, so they never
// reach here.
const longLineWords = 5

// readsAsQuestion is the semantic half of the console's one decision table.
//
// It is only ever asked about a line NOAD has already missed. That containment
// is the safety argument: "hot dog", "a priori" and "use" are lookups because
// the dictionary said so, not because this function was careful.
func readsAsQuestion(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if strings.HasSuffix(line, "?") && strings.IndexFunc(line, unicode.IsLetter) >= 0 {
		return true
	}
	fields := strings.Fields(line)
	if len(fields) >= longLineWords {
		return true
	}
	first := openerStem(fields[0])
	if contains(questionOpeners, first) && len(fields) > 1 {
		return true
	}
	if contains(requestVerbs, first) && len(fields) > 1 {
		return true
	}
	return false
}

// openerStem lowercases and strips the inflection a question opener carries:
// "What's" → "what", "isn't" → "is", "how," → "how".
//
// The negation is cut as the whole unit "n't", BEFORE the apostrophe split and
// never as a bare trailing "n" — "when" is an opener whose last letter is the
// one a lazier rule would eat.
func openerStem(w string) string {
	w = strings.ToLower(strings.TrimFunc(w, func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	}))
	if s, ok := strings.CutSuffix(w, "n't"); ok {
		return s
	}
	if i := strings.Index(w, "'"); i > 0 {
		return w[:i]
	}
	return w
}

func contains(set []string, s string) bool {
	for _, v := range set {
		if v == s {
			return true
		}
	}
	return false
}

// maxQuestionRunes bounds a question quoted back in a diagnostic.
const maxQuestionRunes = 40

// truncateQuestion elides a question for a one-line message. Rune-based, not
// byte-based: cutting mid-rune prints a replacement character.
func truncateQuestion(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= maxQuestionRunes {
		return string(r)
	}
	return strings.TrimRight(string(r[:maxQuestionRunes]), " ") + "…"
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/define/ -run 'TestReadsAsQuestion|TestTruncateQuestion' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/define/question.go cmd/define/question_test.go
git commit -m "#16 M1: readsAsQuestion — the semantic half of the decision table"
```

### Task 2: the syntactic half — `cmdAsk`, `?` and `\`

**Files:**
- Modify: `cmd/define/repl.go:11-50`
- Test: `cmd/define/repl_test.go`

- [ ] **Step 1: Write the failing test** (add rows to the existing table in `repl_test.go`)

```go
{"a question mark in column 1 forces a question", "?what is X", false,
	replCommand{kind: cmdAsk, question: "what is X"}},
{"the forced question keeps its own punctuation", "?is it pejorative?", false,
	replCommand{kind: cmdAsk, question: "is it pejorative?"}},
{"a bare question mark asks nothing", "?", false, replCommand{kind: cmdNothing}},
{"a backslash forces a literal lookup", `\how so`, false,
	replCommand{kind: cmdDefine, word: "how so", literal: true}},
{"a backslash collapses whitespace like any word", `\hot   dog`, false,
	replCommand{kind: cmdDefine, word: "hot dog", literal: true}},
{"a question mark inside a word is part of it", "what?", false,
	replCommand{kind: cmdDefine, word: "what?"}},
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestParseREPLLine -v`
Expected: FAIL — `undefined: cmdAsk`.

- [ ] **Step 3: Implement**

In `cmd/define/repl.go`, extend the kind set and the struct:

```go
const (
	cmdNothing replKind = iota // blank line, nothing to replay
	cmdDefine                  // define this word
	cmdReplay                  // replay the current word
	cmdCommand                 // a /-prefixed command
	cmdAsk                     // a question for the model
)

type replCommand struct {
	kind     replKind
	word     string   // cmdDefine: the headword
	question string   // cmdAsk: the question, with any forcing prefix stripped
	name     string   // cmdCommand: the command name, "" for a bare "/"
	args     []string // cmdCommand: everything after the name
	// literal suppresses the question fallback: a `\`-prefixed line that NOAD
	// misses stays a miss. One of #16's two escape hatches — the other is the
	// "?" prefix, which reaches cmdAsk without a dictionary call at all.
	literal bool
}
```

and, in `parseREPLLine`, immediately after the command test (order matters: `/`
still wins, so `/help` is never a question):

```go
	// The two escape hatches, decided HERE for the same reason "/" is: this
	// function is the one place every entry mode routes a line through, so a
	// prefix test in a loop would make the loops disagree (BR-13, PQ-2).
	if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "?"); ok {
		if q := strings.TrimSpace(rest); q != "" {
			return replCommand{kind: cmdAsk, question: q}
		}
		return replCommand{kind: cmdNothing}
	}
	literal := false
	if rest, ok := strings.CutPrefix(strings.TrimSpace(line), `\`); ok {
		literal, line = true, rest
	}
```

and carry `literal` into the returned `cmdDefine`.

⚠️ A bare `?` returns `cmdNothing`, which the line loop reports as "type a word,
or press return to replay the last one" — wrong words for this case. Task 5
adjusts that message; do not special-case it here.

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/define/ -run TestParseREPLLine -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/define/repl.go cmd/define/repl_test.go
git commit -m "#16 M1: the two escape hatches live on parseREPLLine"
```

### Task 3: route a NOAD miss to the question path

**Files:**
- Modify: `cmd/define/main.go:355-373` (`lookupAndRender`), `:334-343` (`defineOnce`)
- Test: `cmd/define/route_test.go` (create)

The outcome type is what carries the third answer out of the define path:

```go
// lookupOutcome is what one line turned out to be, once NOAD has answered. It
// exists because the raw loop must know the difference: a definition is rendered
// COOKED, while an answer streams in RAW mode (#16 D6).
type lookupOutcome struct {
	code  int    // process/exit semantics, unchanged
	play  bool   // audio should follow
	ask   string // non-empty: this line is a question for the model
	entry string // the raw NOAD text, kept for the ask context
}
```

- [ ] **Step 1: Write the failing test** — the ONE decision table, driven
      end-to-end through the real route with the committed NOAD corpus.

```go
package main

import "testing"

// TestConsoleDecisionTable is the issue's decision table, asserted through the
// REAL route: parseREPLLine, then the dictionary, then readsAsQuestion. Driving
// the halves separately would prove each is correct and leave the table — which
// is the thing the spec promises — unasserted.
func TestConsoleDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want string // "command" | "lookup" | "question" | "not-found"
	}{
		{"a slash command is untouched", "/history 7", "command"},
		{"a one-word headword", "sycophantic", "lookup"},
		{"a two-word headword", "hot dog", "lookup"},
		{"a question NOAD cannot answer", "what's the difference to obsequious?", "question"},
		{"a typo stays a miss", "sycophanti", "not-found"},
		{"the ? hatch skips the dictionary", "?hot dog", "question"},
		{`the \ hatch suppresses the fallback`, `\how so`, "not-found"},
		{"a follow-up request", "use it in a sentence", "question"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := routeFor(t, tc.line); got != tc.want {
				t.Errorf("%q routed to %s, want %s", tc.line, got, tc.want)
			}
		})
	}
}
```

`routeFor` is a helper in the same file that builds `deps` with
`loadFakeDictionary("testdata/entries")`, runs `parseREPLLine`, and for
`cmdDefine` calls `lookupAndRender`, classifying the result from the returned
`lookupOutcome`. Keep it under 25 lines; it is a harness, not a second router.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestConsoleDecisionTable -v`
Expected: FAIL — `lookupAndRender` returns `(int, bool)`.

- [ ] **Step 3: Implement**

`lookupAndRender` returns `lookupOutcome`. Its miss branch becomes:

```go
	text, err := d.dict.Lookup(word)
	if err != nil {
		// The route decision comes BEFORE capture. A question recorded as a
		// not-found lookup would land in the event log that #8's statistics and
		// #17's learner model both fold over — data that is not a lookup at all.
		if !literal && readsAsQuestion(word) {
			return lookupOutcome{ask: word}
		}
		fmt.Fprintf(stderr, "define: %s: %v\n", word, err)
		d.capture.Capture(word, false, opt)
		return lookupOutcome{code: 1}
	}
```

`lookupAndRender` therefore takes the `literal` flag; pass `cmd.literal` from
each caller. Update the three call sites (`defineOnce`, `submitLine`,
`route_test.go`'s harness) and the existing assertions in `capture_test.go`.

⚠️ **Do not move the `d.dict.Lookup` call up into the loops.** That is the
double-lookup this design exists to avoid (D1) and it would break the one-capture-
site invariant `TestCaptureHappensOncePerLookup` pins.

- [ ] **Step 4: Run the whole package**

Run: `go test ./cmd/define/`
Expected: PASS (expect to fix `capture_test.go` and `main_test.go` call sites).

- [ ] **Step 5: Commit**

```bash
git add cmd/define/main.go cmd/define/route_test.go cmd/define/capture_test.go
git commit -m "#16 M1: a NOAD miss that reads as a question routes to the model"
```

### Task 4: both loops and the one-shot path learn `cmdAsk`

**Files:**
- Modify: `cmd/define/repl.go:147-180` (line loop), `cmd/define/replraw.go:140-200`
  (raw loop), `cmd/define/main.go:286-330` (one-shot)
- Test: `cmd/define/commandloop_test.go`, `cmd/define/editorloop_test.go`, `cmd/define/main_test.go`

Until M2 the ask path is one function that prints the degradation message —
`askUnavailable(stderr, question)`. That is what makes M1 a shippable boundary.

- [ ] **Step 1: Write the failing tests** — one per entry mode, all asserting the
      same message, because "every entry mode reaches it" is the invariant BR-13
      cost us:

```go
// line loop
func TestLineLoopRoutesAQuestion(t *testing.T) { /* echo "what is the difference to obsequious?" */ }
// raw editor — a wiring only the loop shell supplies (lessons.md, define #15)
func TestEditorLoopRoutesAQuestion(t *testing.T) { /* scripted key channel through runEditor */ }
// one-shot
func TestOneShotRoutesAQuestion(t *testing.T) { /* run(ctx, []string{"?what is X"}, ...) */ }
```

Each asserts stderr contains `no model configured` and the question, elided.

- [ ] **Step 2: Run them and watch them fail**

Run: `go test ./cmd/define/ -run 'RoutesAQuestion' -v`
Expected: FAIL — the question is looked up, or the one-shot exits 2.

- [ ] **Step 3: Implement**

1. `repl.go`'s switch gains `case cmdAsk:` calling the ask path, and its
   `cmdDefine` case branches on `outcome.ask`.
2. `replraw.go`'s `ActSubmit` gains the same two branches. Ask output goes
   through `crlfWriter` (Task 7 wires the real thing; M1 writes the one-line
   message with an explicit `"\r\n"`).
3. `main.go`'s one-shot: `cmdAsk` dispatches like `cmdCommand` does, using the
   **joined** line, and is exempted from the `fs.NArg() > 1` usage guard for the
   same reason a command is — a question is multi-word by nature:

```go
	case !forgetting && oneShot.kind != cmdCommand && oneShot.kind != cmdAsk && fs.NArg() > 1:
```

⚠️ An unquoted, *unforced* multi-word question (`define what is X`) stays a usage
error, exactly as `define hot dog` is today. Quoting works
(`define "what is X?"`), and so does the hatch (`define ?what is X`). Do not
"fix" the usage guard beyond the `cmdAsk` exemption.

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/define/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/define
git commit -m "#16 M1: every entry mode routes a question, none looks it up"
```

### Task 5: discoverability and the messages

**Files:**
- Modify: `cmd/define/command.go` (`runHelp`), `cmd/define/repl.go` (the `cmdNothing` message)
- Test: `cmd/define/command_test.go`

- [ ] **Step 1: Write the failing test** — `/help` mentions both hatches; the
      bare-`?` message says what to do.
- [ ] **Step 2: Run and watch it fail.**
- [ ] **Step 3: Implement.** `/help` gains a trailing line:
      `ask anything — "?" forces a question, "\" forces a word`. The `cmdNothing`
      message stays as-is for a blank line; a bare `?` gets
      `define: type a question after "?"`. That needs `cmdNothing` to carry the
      reason — add a `note string` field rather than a new kind, and have the
      loops print it when set.
- [ ] **Step 4: Run the tests.** `go test ./cmd/define/`
- [ ] **Step 5: Commit.**

```bash
git commit -am "#16 M1: /help names both hatches"
```

### Task 6: close M1

- [ ] Run the full suite: `go test ./... && go vet ./...`
- [ ] Update `atlas/define.md` — a new "Free-form input" section under "Command
      mode" describing the decision table and both hatches (AGENTS.md §8:
      per-milestone, not an end-of-project sweep).
- [ ] Tick the M1 rows in the issue's `## Plan` and log what was learned.
- [ ] `sdlc milestone-close --issue 16 --milestone M1` — the binary dispatches
      the mandatory fresh-eyes review itself (AGENTS.md §3). Fix Critical and
      Important findings before crossing.

---

## Chunk 2 — M2: the answer

### Task 7: `crlfWriter` and the scoped interrupt

**Files:**
- Create: `cmd/define/crlf.go`, `cmd/define/crlf_test.go`
- Modify: `cmd/define/rawterm.go:43-75`, `cmd/define/replraw.go:15-45`
- Test: `cmd/define/rawterm_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestCRLFWriterTranslatesNewlines(t *testing.T) {
	var buf bytes.Buffer
	w := &crlfWriter{w: &buf}
	io.WriteString(w, "one\ntwo\n")
	if got := buf.String(); got != "one\r\ntwo\r\n" { t.Errorf(...) }
}

func TestCRLFWriterDoesNotDoubleCarriage(t *testing.T) {
	// A delta that already ends "\r\n" must not become "\r\r\n".
}

func TestCRLFWriterSplitAcrossWrites(t *testing.T) {
	// Deltas arrive chunked: "one\r" then "\ntwo" must still yield "one\r\ntwo".
}

func TestInterrupterFiresWhatIsInstalled(t *testing.T) {
	// default fires the session cancel; Set swaps it; the restore func puts it back.
}
```

- [ ] **Step 2: Run and watch them fail.**
- [ ] **Step 3: Implement.** `crlfWriter` holds a `lastWasCR bool` so the
      split-across-writes case works. `interrupter`:

```go
// interrupter is what Ctrl-C means RIGHT NOW.
//
// The reader must cancel immediately — during playback the loop is blocked
// inside speak and cannot read the key itself — but "immediately cancel the
// session" is wrong while a question is streaming, where the answer should stop
// and the prompt should come back. One holder, swapped by whoever owns the
// foreground.
type interrupter struct {
	mu sync.Mutex
	fn context.CancelFunc
}

func (i *interrupter) Set(fn context.CancelFunc) (restore func())
func (i *interrupter) Fire()
```

`readKeys` takes `*interrupter` instead of a `context.CancelFunc` and calls
`Fire()`.

- [ ] **Step 4: Run the tests.** `go test ./cmd/define/`
- [ ] **Step 5: Commit.**

```bash
git commit -am "#16 M2: raw-mode newlines and a Ctrl-C that can mean something narrower"
```

### Task 8: the store learns `user-model.md` and the `asked` event

**Files:**
- Modify: `cmd/define/store/store.go`, `store/event.go`, `store/yaml.go`, `store/mem.go`
- Test: `cmd/define/store/storetest/suite.go` (the conformance suite — both
  implementations), `cmd/define/store/yaml_test.go`

- [ ] **Step 1: Write the failing suite rows**

```go
// in storetest.Suite — runs against Mem AND YAML, so "the fake behaves like the
// real thing" stays a test.
t.Run("UserModel is empty before anything writes one", ...)
t.Run("an asked event round-trips with its question", ...)
t.Run("a torn asked record is dropped", ...) // cut before `at:`
```

- [ ] **Step 2: Run and watch them fail.** `go test ./cmd/define/store/...`
- [ ] **Step 3: Implement.**

```go
// event.go
const EventAsked EventKind = "asked"

type ReviewEvent struct {
	Word     string    `yaml:"word,omitempty"`
	Kind     EventKind `yaml:"kind"`
	Found    bool      `yaml:"found"`
	Correct  bool      `yaml:"correct,omitempty"`
	Question string    `yaml:"question,omitempty"`
	// At stays LAST. The torn-record rule is termination plus completeness, and
	// completeness leans on a cut record losing its timestamp — a field added
	// after this one would be silently droppable instead (#16 D8).
	At time.Time `yaml:"at"`
}

// complete reports whether an event carries every field a real one does. A
// record identifies its SUBJECT — a word for a lookup, a question for an
// asked event — which is the generalisation an `asked` event forced.
func (e ReviewEvent) complete() bool {
	return (e.Word != "" || e.Question != "") && e.Kind != "" && !e.At.IsZero()
}
```

`Store` gains `UserModel() (string, error)`; `YAML` reads `user-model.md` from
its directory (absent → `"", nil`), `Mem` returns a field.

⚠️ Changing `Word`'s YAML tag to `omitempty` changes what a lookup event looks
like on disk. It does not — a lookup always has a word — but check
`yaml_test.go`'s golden day-file assertions and update them deliberately if they
render the field.

- [ ] **Step 4: Run the tests.** `go test ./cmd/define/store/...`
- [ ] **Step 5: Commit.**

```bash
git commit -am "#16 M2: the log records a question; the store reads user-model.md"
```

### Task 9: `askContext` and the pure prompt

**Files:**
- Create: `cmd/define/askctx.go`, `cmd/define/askctx_test.go`,
  `cmd/define/testdata/golden/ask-prompt.txt`
- Test: golden + table

- [ ] **Step 1: Write the failing test** — build an `askContext` literal with a
      current word and entry, three session words, a deck, a user model and one
      prior turn; assert the rendered prompt against a committed golden.

```go
func TestRenderAskPrompt(t *testing.T) {
	ctx := askContext{
		Question:     "what's the difference to obsequious?",
		CurrentWord:  "sycophantic",
		CurrentEntry: "sycophantic | adjective | behaving or done in an obsequious way…",
		SessionWords: []string{"sycophantic", "gaslighting"},
		DeckWords:    []string{"ephemeral", "defenestrate"},
		UserModel:    "## Level\nC1, reads judicial opinions.\n",
		Turns:        []exchange{{Question: "is it pejorative?", Answer: "Yes — strongly."}},
	}
	system, prompt := renderAskPrompt(ctx)
	assertGolden(t, "testdata/golden/ask-prompt.txt", system+"\n---\n"+prompt)
}

func TestRenderAskPromptOmitsAbsentContext(t *testing.T) {
	// No current word, no user model, no turns: the prompt must not contain the
	// section headers for them. An empty "## The learner" header tells the model
	// there is a learner model and it is blank, which is a different claim.
}

func TestRecentTurnsBoundsTheTranscript(t *testing.T) {
	// More than maxTurns exchanges: the OLDEST are dropped, order preserved.
}
```

- [ ] **Step 2: Run and watch them fail.**
- [ ] **Step 3: Implement.** `renderAskPrompt` is pure: an `askContext` in, two
      strings out. Sections are omitted entirely when their data is absent.
      `maxTurns = 6`. Keep the prompt itself short and specific — it is domain
      knowledge and it lives here, not in `internal/llm` (D9).
- [ ] **Step 4: Run the tests.**
- [ ] **Step 5: Commit.**

```bash
git commit -am "#16 M2: the directory, rendered as a prompt"
```

### Task 10: `runAsk` — stream the answer

**Files:**
- Create: `cmd/define/ask.go`, `cmd/define/ask_test.go`
- Modify: `cmd/define/main.go` (`deps.getenv`, `deps.newLLM`, `realDeps`)

- [ ] **Step 1: Write the failing test** — against the **wire-level** fake, not a
      stubbed client:

```go
func TestAskStreamsAnAnswerWithTheDirectoryAsContext(t *testing.T) {
	fake := llmtest.NewFake(t)          // httptest server speaking the wire protocol
	d := testDeps(t)
	d.getenv = envFor(fake.URL())        // DEFINE_LLM_BASE_URL + a key
	d.newLLM = llm.New
	// a deck with two words and a user-model.md in the temp dir
	...
	code := runAsk(ctx, d, opt, sess, "what's the difference to obsequious?", &out, &errb)
	// The recorded REQUEST is the assertion: the context reached the model.
	body := fake.LastRequestBody()
	for _, want := range []string{"sycophantic", "ephemeral", "reads judicial opinions"} {
		if !strings.Contains(body, want) { t.Errorf("prompt is missing %q", want) }
	}
	if !strings.Contains(out.String(), fake.LastAnswer()) { ... }
}

func TestAskDegradesWhenTheSeamIsUnavailable(t *testing.T) {
	// No key: the message names the question, elided, and a LOOKUP still works
	// afterwards in the same session.
}

func TestAskRecordsAnAskedEvent(t *testing.T) {
	// One event, kind "asked", carrying the question and the current word.
}

func TestAskFollowUpCarriesThePreviousExchange(t *testing.T) {
	// Two questions in one session; the second request body contains the first
	// answer.
}
```

- [ ] **Step 2: Run and watch them fail.**
- [ ] **Step 3: Implement.**

```go
// runAsk is the one place a question reaches the network.
//
// It is thin on purpose: resolve, build, gather, render, stream, record. Every
// decision it looks like it makes — what context to include, how to phrase it,
// how many turns to keep — belongs to renderAskPrompt and is unit-tested without
// a socket (ARCH-PURE).
func runAsk(ctx context.Context, d deps, opt options, sess *session, question string, out, errOut io.Writer) int
```

Degradation: `llm.Resolve` error or `errors.Is(err, llm.ErrUnavailable)` →
```
define: no model configured; `what's the difference…` is not a word
```
on stderr, exit 1 for the one-shot path, loop continues otherwise.
`llm.ErrRequest` stays **loud** (our bug), per the taxonomy in `atlas/llm.md`.

- [ ] **Step 4: Run the tests.** `go test ./cmd/define/...`
- [ ] **Step 5: Commit.**

```bash
git commit -am "#16 M2: runAsk — the question, the directory, and the stream"
```

### Task 11: Ctrl-C mid-stream, in the loop that actually owns the terminal

**Files:**
- Modify: `cmd/define/replraw.go` (the `cmdAsk` branch)
- Test: `cmd/define/editorloop_test.go`, `cmd/define/pty_conformance_test.go`

- [ ] **Step 1: Write the failing test** — drive `runEditor` with a scripted key
      channel: submit a question, let the fake stream two deltas, send
      `KeyInterrupt`, then submit a word and assert its definition renders. The
      session must survive; the process must not exit.

```go
func TestEditorCtrlCMidStreamReturnsToThePrompt(t *testing.T) { ... }
```

Add a pty conformance row alongside `TestPTYCtrlCDuringPlaybackExitsPromptly`,
because an in-process test proves the bytes were emitted, not that the terminal
came back.

- [ ] **Step 2: Run and watch it fail** (today the interrupt ends the loop).
- [ ] **Step 3: Implement.** The `cmdAsk` branch:

```go
	qctx, qcancel := context.WithCancel(ctx)
	restore := interrupts.Set(qcancel)
	done := make(chan int, 1)
	go func() { done <- runAsk(qctx, d, opt, &sess, cmd.question, &crlfWriter{w: stdout}, stderr) }()
	for streaming := true; streaming; {
		select {
		case <-done:
			streaming = false
		case k := <-keys:
			// Swallowed deliberately: the reader has already fired the scoped
			// cancel, and letting this key reach Apply would exit the session —
			// which is what Ctrl-C means everywhere EXCEPT here.
			if k.Kind != KeyInterrupt {
				continue
			}
		}
	}
	restore()
	qcancel()
```

⚠️ Order matters: `restore()` before the next `draw()`, so a Ctrl-C landing
between the stream ending and the prompt returning still means "quit".

- [ ] **Step 4: Run the tests.** `go test ./cmd/define/` and
      `go test -tags conformance ./cmd/define/ -run PTY`
- [ ] **Step 5: Commit.**

```bash
git commit -am "#16 M2: Ctrl-C stops the answer, not the session"
```

### Task 12: close the issue

- [ ] Full suite: `go test ./... && go vet ./...`
- [ ] Live check by hand, recorded in `## Log`:
      `define` → `sycophantic` → `what's the difference to obsequious?` →
      `give me three more examples` → Ctrl-C mid-answer → `hot dog`.
- [ ] Confirm the event log: `cat events/*.yaml` shows `kind: asked` records
      carrying their questions.
- [ ] Update `atlas/define.md` (the ask path, the context pack, the scoped
      interrupt) and `atlas/index.md` if a new file warrants a pointer.
- [ ] Tick the project row for `tools#16` in `workshop/projects/define-learn.md`
      and add its `**actual:**` / `**closed:**` block.
- [ ] `sdlc close --issue 16 --verified '<evidence>'` — omit `--actual`; the
      binary measures and adopts it (AGENTS.md §5).

---

## Done-when → task map

| Issue Done-when row | Task |
|---|---|
| classification is pure, every table row asserted | 1, 3 |
| both hatches work, each pinned by a test that fails without it | 2, 3 |
| full round trip against the fake, deck + user-model in the prompt | 10 |
| a follow-up resolves against the previous exchange | 9, 10 |
| Ctrl-C mid-stream returns to the prompt, session intact | 7, 11 |
| seam unavailable → explanatory message, lookup still works | 4, 10 |
| driven through the raw TUI loop, not only the piped loop | 4, 11 |

## Open questions for the operator

1. **`\` as the force-literal hatch** (D3). It is one keystroke and shell-native,
   but it is also the one character a user may have to escape depending on their
   terminal's paste handling. `=` or `.` are alternatives. Happy to switch.
2. **The ≥5-word arm** (D4). It makes the classifier generous — a long line NOAD
   misses becomes a question rather than "not found". That is the right default
   for a learning tool, but it means a badly mistyped long phrase gets an answer
   instead of a correction. Keep?
3. **`Store.UserModel` now** (D7) versus a plain file read deferred to `#17`.
   Adding the method here means `#17` inherits the seam and the conformance rows;
   the cost is one method on `Store` that only one caller uses until `#17` lands.
