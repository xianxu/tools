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
`lookupAndRender` the one capture site, an invariant
`TestCaptureArityIsOnePerLookup` (`capture_test.go:89`) pins.

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

**D4 — `readsAsQuestion` has three arms, and only ever sees NOAD misses.** That
containment is what makes it safe: `hot dog`, `use`, `a priori` never reach it.

| arm | example | why |
|---|---|---|
| trailing `?` | `is it pejorative?` | the explicit mark |
| first word is interrogative/auxiliary | `what's the difference to obsequious` | the wh/aux opener |
| first word is a request verb **and** ≥2 words | `use it in a sentence`, `give me three more examples` | a follow-up is imperative, not interrogative |

`sycophanti` fails all three → not-found, unchanged.

**Length is deliberately NOT an arm** (operator, 2026-08-23). A draft had "≥5
words → question" as a backstop for `difference between sycophantic and
obsequious`, which reads as neither interrogative nor imperative. That is a word
count wearing a different hat, and the spec rejects word count as the signal. The
cost is named rather than hidden: a long phrasal request with no wh-word, no `?`
and no request verb answers `not found`, and `?` is its recovery. If real use
shows that landing often, the arm is one line — but it is an operator decision,
not a drift.

**D5 — One answer to "what does an interrupt mean right now", and it is not the
transport's.** Ctrl-C reaches this program two ways, and the design must not bet
on either. In raw mode `term.MakeRaw` clears ISIG so `\x03` is a byte the key
reader decodes; but `main` also installs `signal.NotifyContext`, and the pty
suite's own header records the measurement that a `\x03` written to a master
**reached define as a SIGINT** and left through `NotifyContext`, not the key
reader (`pty_conformance_test.go:13-25`). A scoped interrupter that only the byte
path feeds would therefore be cancelled out from under by the signal path, ending
the session while a question streams.

So the sink is the single answer and **both** transports feed it:

| transport | reaches the sink via |
|---|---|
| byte `\x03` (raw mode, ISIG off) | `readKeys` decoding `KeyInterrupt` |
| SIGINT (startup race, a terminal that kept ISIG, `kill -INT`) | a `signal.Notify` channel the loop owns |

For that to hold, the interactive loop's context must not *also* be cancelled
behind the sink's back, so `run` detaches it — `context.WithoutCancel(ctx)` —
before deriving the loop's own cancellable context. `main` keeps
`signal.NotifyContext` unchanged for the one-shot and piped paths, whose contract
really is "SIGINT ends the program".

**Where this is installed is part of the rule, not an implementation detail.**
The detach and the sink go together, at the place the loop is actually chosen —
`repl` (`repl.go:77`, deciding `replRaw` at `:95` vs `replLines` at `:97`) — and
never in `run`'s `case 0:` (`main.go:314-321`), which serves **both** loops. Put
the detach there and give the watcher to `replRaw` only, and every non-terminal
zero-arg run — `echo word | define`, `define < words.txt`, a redirected stdout,
and `replRaw`'s own two fallbacks at `replraw.go:20,25` — reaches `replLines`,
whose only interrupt transport is `ctx.Done()` (`repl.go:132`). Detached from
`NotifyContext` and unwatched, that loop would be **uninterruptible**: SIGINT is
diverted from default termination by `main.go:154` and then delivered to nothing.

So the enumeration is the deliverable, and every cell is wired:

| loop entry | byte `\x03` | SIGINT | default sink does |
|---|---|---|---|
| `replRaw` (terminal, raw mode entered) | `readKeys` → `Fire()` | `repl`'s watcher → `Fire()` | cancels the loop ctx — today's behaviour |
| `replLines` (piped, redirected, or a raw-mode fallback) | n/a — no key reader | `repl`'s watcher → `Fire()` | cancels the loop ctx → `repl.go:132` returns 0 — today's behaviour |
| one-shot / `-forget` / `--llm-check` | n/a | `main`'s `NotifyContext`, unchanged | ends the program |

The default sink is the loop's own cancel, so quitting at the prompt, Ctrl-C
during playback, and Ctrl-C in a piped run behave exactly as they do today; the
ask path points it at the question's own `CancelFunc` for the duration of a
stream. The signal source is injected as a `<-chan os.Signal` so a test drives it
without raising a real signal in the test binary.


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
| `readsAsQuestion` / `openerStem` | `cmd/define/question.go` | new |
| `truncateQuestion` | `cmd/define/question.go` | new |
| `parseREPLLine` / `replCommand` | `cmd/define/repl.go` | modified |
| `recallLine` / `nothingSays` | `cmd/define/repl.go` | new |
| `lookupOutcome` | `cmd/define/main.go` | new |
| `question` / `mayAsk` | `cmd/define/ask.go` | new |
| `askContext` | `cmd/define/askctx.go` | new |
| `renderAskPrompt` | `cmd/define/askctx.go` | new |
| `recentTurns` | `cmd/define/askctx.go` | new |
| `recentDeck` | `cmd/define/askctx.go` | new |
| `consumed` / `carriedCR` | `cmd/define/crlf.go` | new |
| `session` | `cmd/define/session.go` | new |
| `exchange` | `cmd/define/askctx.go` | new |
| `crlfWriter` | `cmd/define/crlf.go` | new |
| `ReviewEvent` / `complete` | `cmd/define/store/event.go` | modified |

- **readsAsQuestion** — the semantic half of the decision table: does this line,
  which NOAD does not know, read as a question. Three arms (D4), no IO, no state.
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
  text of that word, `words` (this session's lookups — the signal an issue
  Done-when row quantifies over, distinct from the durable deck), and `turns`
  (the Q&A transcript). The last two are M2's.
  - **Relationships:** one per loop invocation. It **replaces** the bare
    `current string` declared in `replLines` (`repl.go:125`), in `runEditor`
    (`replraw.go`), and threaded through `submitLine` as `current *string`
    (`replraw.go:228-229`) — all three migrate in Task 4, not later.
  - **DRY rationale:** three declarations of "what is this session holding" is
    already one too many, and #16 adds two more facts to hold. It is also where
    the ask contract lives: a question must not become the current word, and one
    struct is what makes that a single assignment site to not make.
  - **Future extensions:** `#17`'s per-session signal and `#6`'s review state
    both want a home that is not a loop-local variable.

- **crlfWriter** — an `io.Writer` decorator mapping `\n` → `\r\n` for raw-mode
  output (D6).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `gatherAskContext` | `cmd/define/ask.go` | new | store reads |
| `runAsk` | `cmd/define/ask.go` | new | `llm.Client.Stream` |
| `deps.newLLM` / `deps.getenv` | `cmd/define/main.go` | new | `llm.New` + `llm.Resolve` |
| `Store.UserModel` / `SetUserModel` | `cmd/define/store/{store,yaml,mem}.go` | modified | `user-model.md` on disk |
| `writeBytesAtomic` | `cmd/define/store/yaml.go` | new | temp file + rename |
| `interrupter` | `cmd/define/interrupt.go` | new | what Ctrl-C means right now |
| `deps.notifySignals` | `cmd/define/main.go` | new | `signal.Notify` |
| `Capturer.CaptureAsk` | `cmd/define/capture.go` | modified | event append |
| `askScoped` | `cmd/define/ask.go` | new | the interrupt sink, for the duration of one answer |
| `unavailable` / `unavailableAfterSending` | `cmd/define/ask.go` | new | the two degradation messages |

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

- **interrupter** — a mutex-guarded holder of what Ctrl-C means right now (D5).
  `Set(fn)` returns a restore func; `Fire()` calls whatever is installed.
  - **Injected into:** `readKeys` (replacing its `context.CancelFunc` parameter)
    **and** the loop's signal watcher — the two transports, one sink. If only one
    of them fed it, the other would still end the session mid-stream.

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

- [x] **Step 1: Write the failing test**

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
		{"length alone is not a signal", "difference between sycophantic and obsequious please", false},
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

- [x] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run 'TestReadsAsQuestion|TestTruncateQuestion' -v`
Expected: FAIL — `undefined: readsAsQuestion`.

- [x] **Step 3: Implement**

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

// readsAsQuestion is the semantic half of the console's one decision table.
//
// It is only ever asked about a line NOAD has already missed. That containment
// is the safety argument: "hot dog", "a priori" and "use" are lookups because
// the dictionary said so, not because this function was careful.
//
// There is deliberately NO length arm. Word count is the signal the spec
// rejects, and a line being long is not the same fact as it reading as a
// question — see the note under D4.
func readsAsQuestion(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if strings.HasSuffix(line, "?") && strings.IndexFunc(line, unicode.IsLetter) >= 0 {
		return true
	}
	fields := strings.Fields(line)
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

- [x] **Step 4: Run the tests**

Run: `go test ./cmd/define/ -run 'TestReadsAsQuestion|TestTruncateQuestion' -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add cmd/define/question.go cmd/define/question_test.go
git commit -m "#16 M1: readsAsQuestion — the semantic half of the decision table"
```

### Task 2: the syntactic half — `cmdAsk`, `?` and `\`

**Files:**
- Modify: `cmd/define/repl.go:11-50`
- Test: `cmd/define/repl_test.go`

- [x] **Step 1: Write the failing test** (add rows to the existing table in `repl_test.go`)

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

- [x] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestParseREPLLine -v`
Expected: FAIL — `undefined: cmdAsk`.

- [x] **Step 3: Implement**

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

- [x] **Step 4: Run the tests**

Run: `go test ./cmd/define/ -run TestParseREPLLine -v`
Expected: PASS.

- [x] **Step 5: Commit**

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

- [x] **Step 1: Write the failing test** — the ONE decision table, driven
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
		{"a long phrase with no interrogative reading", "difference between sycophantic and obsequious", "not-found"},
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

- [x] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestConsoleDecisionTable -v`
Expected: FAIL — `lookupAndRender` returns `(int, bool)`.

- [x] **Step 3: Implement**

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
site invariant `TestCaptureArityIsOnePerLookup` (`capture_test.go:89`) pins.

- [x] **Step 4: Run the whole package**

Run: `go test ./cmd/define/`
Expected: PASS (expect to fix `capture_test.go` and `main_test.go` call sites).

- [x] **Step 5: Commit**

```bash
git add cmd/define/main.go cmd/define/route_test.go cmd/define/capture_test.go
git commit -m "#16 M1: a NOAD miss that reads as a question routes to the model"
```

### Task 4: one session, and an ask that touches none of the lookup state

**Files:**
- Create: `cmd/define/session.go`
- Modify: `cmd/define/repl.go:125,147-180` (line loop), `cmd/define/replraw.go:69,140-200,228-246`
  (raw loop + `submitLine`), `cmd/define/main.go:286-330` (one-shot)
- Test: `cmd/define/commandloop_test.go`, `cmd/define/editorloop_test.go`, `cmd/define/main_test.go`

Two things land together because they are one class: the loops have three
separate declarations of "what is this session holding" (`repl.go:125`,
`runEditor`'s `var current string`, `submitLine`'s `current *string`), and #16
adds an outcome none of them has a rule for. Wiring the outcome into three
places without unifying them is how the rule comes to be written twice and
believed once.

**The ask outcome's contract**, stated once here and asserted per entry mode:

| the ask path… | because |
|---|---|
| does **not** set `sess.current` | a bare Enter would then "replay" a question, and M2's `askContext.CurrentWord` would be the previous question rather than the word it was about |
| does **not** capture anything | D2 — the log is lookups |
| **does** add the line to editor recall (`hist.Add`) | Up-arrow must recall the question you just asked, exactly as it recalls a `/command` |
| returns the ask path's own exit code | not `defineOnce`'s; a question that could not be answered is not a failed lookup |

Until M2 the ask path is one function that prints the degradation message —
`askUnavailable(stderr, question)`. That is what makes M1 a shippable boundary.

- [x] **Step 1: Write the failing tests** — one per entry mode, because "every
      entry mode reaches it" is the invariant BR-13 cost us, plus one per
      contract row:

```go
// line loop
func TestLineLoopRoutesAQuestion(t *testing.T) { /* echo "what is the difference to obsequious?" */ }
// raw editor — a wiring only the loop shell supplies (lessons.md, define #15)
func TestEditorLoopRoutesAQuestion(t *testing.T) { /* scripted key channel through runEditor */ }
// one-shot
func TestOneShotRoutesAQuestion(t *testing.T) { /* run(ctx, []string{"?what is X"}, ...) */ }

// the contract rows
func TestAQuestionDoesNotBecomeTheCurrentWord(t *testing.T) {
	// look up "sycophantic", ask a question, press Enter: the REPLAY is
	// sycophantic. Deleting the guard in the ask branch reddens this.
}
func TestAQuestionIsNotCaptured(t *testing.T) {
	// a fake Capturer at the seam sees exactly one Capture for the lookup and
	// none for the question.
}
func TestAQuestionIsRecalledByUpArrow(t *testing.T) {
	// scripted keys: question, Up — the editor line is the question.
}
```

- [x] **Step 2: Run them and watch them fail**

Run: `go test ./cmd/define/ -run 'RoutesAQuestion|AQuestion' -v`
Expected: FAIL — the question is looked up, or the one-shot exits 2.

- [x] **Step 3: Implement**

1. `cmd/define/session.go`:

```go
// session is what one REPL session is holding. It replaced three separate
// declarations of the same idea — repl.go's `current`, runEditor's `current`,
// and submitLine's `current *string` — because #16 added a rule ("an ask
// touches none of this") that would otherwise have been written three times.
type session struct {
	// current is the word a bare Enter replays. Only a SUCCESSFUL lookup sets
	// it: not a typo, and not a question.
	current string
	// entry is the raw NOAD text of current, kept so a question that follows a
	// lookup can carry it as context without a second dictionary call (#16 D1).
	entry string
}
```

`exchange` and `turns` are M2's (Task 9); do not add them here.

2. `replLines`, `runEditor` and `submitLine` take `*session` in place of their
   `current` variables. `submitLine` sets `sess.current`/`sess.entry` from the
   `lookupOutcome` on a zero code, and returns without touching either when the
   outcome is an ask.
3. `repl.go`'s switch gains `case cmdAsk:`, and its `cmdDefine` case branches on
   `outcome.ask` — **both reaching the same** `askUnavailable` call, not two.
4. `replraw.go`'s `ActSubmit` gains the same two entries into **one** closure
   (`askInSession`), for the same reason: the forced (`?…`) and unforced
   (NOAD-miss) routes differ only in whether the dictionary was consulted, and
   Task 11 gives that closure the interrupter and `crlfWriter` wiring exactly
   once (ARCH-DRY).
5. `main.go`'s one-shot: `cmdAsk` dispatches like `cmdCommand` does, using the
   **joined** line and returning the ask path's code; and it is exempted from the
   `fs.NArg() > 1` usage guard for the same reason a command is — a question is
   multi-word by nature:

```go
	case !forgetting && oneShot.kind != cmdCommand && oneShot.kind != cmdAsk && fs.NArg() > 1:
```

The one-shot's unforced route (`define "what is X?"`, one quoted arg) reaches the
ask through `lookupOutcome.ask` in `defineOnce`, which must therefore return the
ask path's code too.

⚠️ An unquoted, *unforced* multi-word question (`define what is X`) stays a usage
error, exactly as `define hot dog` is today. Quoting works
(`define "what is X?"`), and so does the hatch (`define ?what is X`). Do not
"fix" the usage guard beyond the `cmdAsk` exemption.

- [x] **Step 4: Run the tests**

Run: `go test ./cmd/define/`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add cmd/define
git commit -m "#16 M1: one session, and an ask that touches none of the lookup state"
```

### Task 5: discoverability and the messages

**Files:**
- Modify: `cmd/define/command.go` (`runHelp`), `cmd/define/repl.go` (the `cmdNothing` message)
- Test: `cmd/define/command_test.go`

- [x] **Step 1: Write the failing test** — `/help` mentions both hatches; the
      bare-`?` message says what to do.
- [x] **Step 2: Run and watch it fail.**
- [x] **Step 3: Implement.** `/help` gains a trailing line:
      `ask anything — "?" forces a question, "\" forces a word`. The `cmdNothing`
      message stays as-is for a blank line; a bare `?` gets
      `define: type a question after "?"`. That needs `cmdNothing` to carry the
      reason — add a `note string` field rather than a new kind, and have the
      loops print it when set.
- [x] **Step 4: Run the tests.** `go test ./cmd/define/`
- [x] **Step 5: Commit.**

```bash
git commit -am "#16 M1: /help names both hatches"
```

### Task 6: close M1

- [x] Run the full suite: `go test ./... && go vet ./...`
- [x] Update `atlas/define.md` — a new "Free-form input" section under "Command
      mode" describing the decision table and both hatches (AGENTS.md §8:
      per-milestone, not an end-of-project sweep).
- [x] Tick the M1 rows in the issue's `## Plan` and log what was learned.
- [x] `sdlc milestone-close --issue 16 --milestone M1` — the binary dispatches
      the mandatory fresh-eyes review itself (AGENTS.md §3). Fix Critical and
      Important findings before crossing.

---

## Chunk 2 — M2: the answer

### Task 7: `crlfWriter`, and one sink for both interrupt transports

**Files:**
- Create: `cmd/define/crlf.go`, `cmd/define/crlf_test.go`
- Modify: `cmd/define/rawterm.go:37-75`, `cmd/define/repl.go:77-97` (the sink and
  the detach), `cmd/define/main.go:314-321` (hand the loop the signal ctx),
  `cmd/define/main.go` (`deps.notifySignals`)
- Test: `cmd/define/rawterm_test.go`, `cmd/define/main_test.go`

- [x] **Step 1: Write the failing tests**

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

// The findings this task exists for (PQ-1, PQ-6): every loop, every transport.
func TestBothInterruptTransportsReachTheSink(t *testing.T) {
	// (a) a KeyInterrupt decoded by readKeys fires the sink;
	// (b) a value on the injected signal channel fires the SAME sink;
	// (c) with the sink pointed at a question cancel, NEITHER cancels the
	//     loop's context — which is what makes "the session survives" true.
}

func TestThePipedLoopStillExitsOnASignal(t *testing.T) {
	// replLines, reached with a non-terminal stdin, must still return 0 when the
	// injected signal channel fires. Its only transport is the sink's default,
	// and PQ-6 is exactly the wiring where it has none.
}
```

- [x] **Step 2: Run and watch them fail.**

- [x] **Step 3: Implement.**

`crlfWriter` holds a `lastWasCR bool` so the split-across-writes case works.

```go
// interrupter is what Ctrl-C means RIGHT NOW.
//
// Two transports deliver it and neither may own the meaning: raw mode clears
// ISIG so \x03 is a byte the key reader decodes, but the pty suite measured a
// \x03 arriving as a SIGINT instead (pty_conformance_test.go:13-25), which
// would leave through NotifyContext and end a session mid-answer. Both feed
// this holder; whoever owns the foreground decides what it does.
type interrupter struct {
	mu sync.Mutex
	fn context.CancelFunc
}

func (i *interrupter) Set(fn context.CancelFunc) (restore func())
func (i *interrupter) Fire()
```

Three wirings, and the third is the one that makes the other two mean anything:

1. `readKeys` takes `*interrupter` instead of a `context.CancelFunc`, and calls
   `Fire()` where it called `cancel()`. Update its doc comment — it currently
   asserts Ctrl-C is a byte "not a signal", which the pty suite disproved.
2. **`repl` owns both** — the detach, the sink, and the watcher, installed
   before it chooses a loop (`repl.go:77-97`) so `replRaw` and `replLines` are
   served by the same three lines:

```go
	// The loop owns what an interrupt means (#16 D5), so it must not ALSO be
	// cancelled behind the sink's back by main's NotifyContext. This sits HERE,
	// above the replRaw/replLines choice, because BOTH loops need a transport:
	// detaching in run() and watching only in replRaw leaves the piped loop with
	// ctx.Done() that nothing can reach — SIGINT diverted by NotifyContext and
	// delivered to no one.
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	interrupts := &interrupter{fn: cancel}
	go func() {
		for range d.notifySignals(os.Interrupt) {
			interrupts.Fire()
		}
	}()
```

   `d.notifySignals` is a `func(...os.Signal) <-chan os.Signal` seam on `deps`,
   defaulting to `signal.Notify`'s channel. Injected rather than called directly
   so a test drives it without raising a real signal in the test binary.
3. `run`'s `case 0:` (`main.go:314-321`) hands `repl` the signal context
   unchanged and stops deriving a cancel of its own — that responsibility moved
   down to where the loop is chosen. The one-shot, `-forget` and `--llm-check`
   paths keep `NotifyContext` exactly as they have it, where "SIGINT ends the
   program" is the right contract.

⚠️ The default sink is the loop's own `cancel`, so quitting at the prompt, Ctrl-C
during playback, and Ctrl-C in a piped run are byte-for-byte what they are today.
Verify by running the existing `TestEditorLoopCtrlCExitsZero` and the pty suite
unchanged — if either needs editing, the default sink is wired wrong. Add one
test that a piped run still exits on a signal, because that is the path this
finding showed a plausible wiring silently strands.

- [x] **Step 4: Run the tests.** `go test ./cmd/define/` and
      `go test -tags conformance -run PTY ./cmd/define/`
- [x] **Step 5: Commit.**

```bash
git commit -am "#16 M2: one sink for both ways Ctrl-C arrives"
```

### Task 8: the store learns `user-model.md` and the `asked` event

**Files:**
- Modify: `cmd/define/store/store.go`, `store/event.go`, `store/yaml.go`, `store/mem.go`
- Test: `cmd/define/store/storetest/suite.go` (the conformance suite — both
  implementations), `cmd/define/store/yaml_test.go`

- [x] **Step 1: Write the failing suite rows**

```go
// in storetest.Suite — runs against Mem AND YAML, so "the fake behaves like the
// real thing" stays a test.
t.Run("UserModel is empty before anything writes one", ...)
t.Run("an asked event round-trips with its question", ...)
t.Run("a torn asked record is dropped", ...) // cut before `at:`
```

- [x] **Step 2: Run and watch them fail.** `go test ./cmd/define/store/...`
- [x] **Step 3: Implement.**

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

- [x] **Step 4: Run the tests.** `go test ./cmd/define/store/...`
- [x] **Step 5: Commit.**

```bash
git commit -am "#16 M2: the log records a question; the store reads user-model.md"
```

### Task 9: `askContext` and the pure prompt

**Files:**
- Create: `cmd/define/askctx.go`, `cmd/define/askctx_test.go`,
  `cmd/define/testdata/golden/ask-prompt.txt`
- Test: golden + table

- [x] **Step 1: Write the failing test** — build an `askContext` literal with a
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

- [x] **Step 2: Run and watch them fail.**
- [x] **Step 3: Implement.** `renderAskPrompt` is pure: an `askContext` in, two
      strings out. Sections are omitted entirely when their data is absent.
      `maxTurns = 6`. Keep the prompt itself short and specific — it is domain
      knowledge and it lives here, not in `internal/llm` (D9).
- [x] **Step 4: Run the tests.**
- [x] **Step 5: Commit.**

```bash
git commit -am "#16 M2: the directory, rendered as a prompt"
```

### Task 10: `runAsk` — stream the answer

**Files:**
- Create: `cmd/define/ask.go`, `cmd/define/ask_test.go`
- Modify: `cmd/define/main.go` (`deps.getenv`, `deps.newLLM`, `realDeps`)

- [x] **Step 1: Write the failing test** — against the **wire-level** fake, not a
      stubbed client. The API below is the real one
      (`internal/llm/llmtest/fake.go`): `Fake` embeds `*httptest.Server`, so
      `fake.URL` is a field; replies are queued with `Script(match, replies...)`;
      and what was received comes back as `Requests() []Recorded`, whose
      `Prompt()` and `System()` are what the assertions read.

```go
func TestAskStreamsAnAnswerWithTheDirectoryAsContext(t *testing.T) {
	fake := llmtest.NewFake(t)
	fake.Script("obsequious", llmtest.Reply{Text: "Obsequious is stronger…"})
	d := testDeps(t)
	d.getenv = envFor(fake.URL)      // DEFINE_LLM_BASE_URL + DEFINE_LLM_API_KEY
	d.newLLM = llm.New
	// a deck with two words and a user-model.md in the temp dir
	...
	code := runAsk(ctx, d, opt, sess, "what's the difference to obsequious?", &out, &errb)

	// The recorded REQUEST is the assertion: the context reached the model.
	got := fake.Requests()
	if len(got) != 1 { t.Fatalf("requests = %d, want 1", len(got)) }
	body := got[0].System() + "\n" + got[0].Prompt()
	for _, want := range []string{"sycophantic", "ephemeral", "reads judicial opinions"} {
		if !strings.Contains(body, want) { t.Errorf("prompt is missing %q", want) }
	}
	if !strings.Contains(out.String(), "Obsequious is stronger") { ... }
}

func TestAskDegradesWhenTheSeamIsUnavailable(t *testing.T) {
	// No key: the message names the question, elided, and a LOOKUP still works
	// afterwards in the same session.
}

// PQ-4: a cancelled stream is the user's own keypress, not a failure.
func TestAskSaysNothingWhenTheUserCancels(t *testing.T) {
	// ctx cancelled mid-stream: stderr is EMPTY and the code is 0. Without the
	// guard this prints "no model configured", because a cancelled request has
	// no HTTP status and mapError classifies statusless failures as
	// ErrUnavailable (anthropic.go:282-290) — the degradation message would be
	// the answer to pressing Ctrl-C.
}

func TestAskRecordsAnAskedEvent(t *testing.T) {
	// One event, kind "asked", carrying the question and the current word.
}

func TestAskFollowUpCarriesThePreviousExchange(t *testing.T) {
	// Two questions in one session; the second request's Prompt() contains the
	// first answer.
}
```

- [x] **Step 2: Run and watch them fail.**
- [x] **Step 3: Implement.**

```go
// runAsk is the one place a question reaches the network.
//
// It is thin on purpose: resolve, build, gather, render, stream, record. Every
// decision it looks like it makes — what context to include, how to phrase it,
// how many turns to keep — belongs to renderAskPrompt and is unit-tested without
// a socket (ARCH-PURE).
func runAsk(ctx context.Context, d deps, opt options, sess *session, question string, out, errOut io.Writer) int
```

Its taxonomy, in the order it must be asked:

| condition | behaviour |
|---|---|
| `ctx.Err() != nil` after the stream | **nothing printed, code 0** — this is the user's own Ctrl-C. Asked FIRST, before the error is classified at all. Precedent: `playAnnounced`'s cancelled-context guard, `llmcheck.go:62`'s `parent.Err()` check |
| `llm.Resolve` error, or `llm.ErrUnavailable` | ``define: no model configured; `what's the difference…` is not a word`` on stderr; code 1 |
| `llm.ErrTruncated` with partial text | keep what arrived, warn once that it was cut |
| `llm.ErrRequest` | **loud** — our bad prompt or model, per the taxonomy in `atlas/llm.md` |

⚠️ The cancel guard cannot be a `errors.Is(err, context.Canceled)` test:
`mapError` has already turned a statusless failure into `ErrUnavailable` by the
time `runAsk` sees it, so the original cause is gone. Ask the **context**, not
the error.

- [x] **Step 4: Run the tests.** `go test ./cmd/define/...`
- [x] **Step 5: Commit.**

```bash
git commit -am "#16 M2: runAsk — the question, the directory, and the stream"
```

### Task 11: Ctrl-C mid-stream, in the loop that actually owns the terminal

**Files:**
- Modify: `cmd/define/replraw.go` (the single `askInSession` closure from Task 4)
- Test: `cmd/define/editorloop_test.go`, `cmd/define/pty_conformance_test.go`

- [x] **Step 1: Write the failing tests** — drive `runEditor` with a scripted key
      channel: submit a question, let the fake stream two deltas, send
      `KeyInterrupt`, then submit a word and assert its definition renders. The
      session must survive; the process must not exit.

```go
func TestEditorCtrlCMidStreamReturnsToThePrompt(t *testing.T) { /* byte transport */ }
func TestEditorSignalMidStreamReturnsToThePrompt(t *testing.T) {
	// The SAME assertion driven through the injected signal channel. Two tests,
	// not one, because the two transports are exactly what PQ-1 showed a design
	// can silently serve only half of.
}
func TestForcedAndUnforcedAsksShareOneWiring(t *testing.T) {
	// "?why" and a NOAD-missing question both stream through crlfWriter and both
	// cancel on Ctrl-C. Deleting either call site's route into askInSession
	// reddens this — a second copy of the wiring would not (ARCH-DRY).
}
```

Add a **pty conformance** row too, and note what makes it different from the
three already there: those assert "exited, and the terminal is sane", an
observable the byte path, the signal path and a crash all produce — which is why
that suite's header says it cannot pin the byte path. "Ctrl-C mid-answer, then a
lookup renders" is an observable **only a surviving session** produces, so it
distinguishes what the existing rows cannot. That closes the gap that file
documents rather than restating a claim it disproved.

- [x] **Step 2: Run and watch them fail** (today the interrupt ends the loop).
- [x] **Step 3: Implement.** Inside `askInSession` — the ONE closure both the
      forced and unforced routes enter (Task 4):

```go
	qctx, qcancel := context.WithCancel(ctx)
	restore := interrupts.Set(qcancel)
	done := make(chan int, 1)
	go func() { done <- runAsk(qctx, d, opt, sess, question, &crlfWriter{w: stdout}, stderr) }()
	for streaming := true; streaming; {
		select {
		case <-done:
			streaming = false
		case k := <-keys:
			// Swallowed deliberately: the sink has already fired the scoped
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
between the stream ending and the prompt returning still means "quit". And
`qcancel()` after `restore()`, so the deferred cancel cannot fire a sink that is
no longer this question's.

- [x] **Step 4: Run the tests.** `go test ./cmd/define/` and
      `go test -tags conformance ./cmd/define/ -run PTY`
- [x] **Step 5: Commit.**

```bash
git commit -am "#16 M2: Ctrl-C stops the answer, not the session"
```

### Task 12: close the issue

- [x] Full suite: `go test ./... && go vet ./...`
- [x] Live check by hand, recorded in `## Log`:
      `define` → `sycophantic` → `what's the difference to obsequious?` →
      `give me three more examples` → Ctrl-C mid-answer → `hot dog`.
- [x] Confirm the event log: `cat events/*.yaml` shows `kind: asked` records
      carrying their questions.
- [x] Update `atlas/define.md` (the ask path, the context pack, the scoped
      interrupt) and `atlas/index.md` if a new file warrants a pointer.
- [x] Tick the project row for `tools#16` in `workshop/projects/define-learn.md`
      and add its `**actual:**` / `**closed:**` block.
- [x] `sdlc close --issue 16 --verified '<evidence>'` — omit `--actual`; the
      binary measures and adopts it (AGENTS.md §5).

---

## Done-when → task map

| Issue Done-when row | Task |
|---|---|
| classification is pure, every table row asserted | 1, 3 |
| both hatches work, each pinned by a test that fails without it | 2, 3 |
| full round trip against the fake, deck + user-model in the prompt | 10 |
| a follow-up resolves against the previous exchange | 9, 10 |
| Ctrl-C mid-stream returns to the prompt, session intact (both transports) | 7, 11 |
| seam unavailable → explanatory message, lookup still works | 4, 10 |
| driven through the raw TUI loop, not only the piped loop | 4, 11 |

## Operator decisions (2026-08-23)

All three open questions are answered; nothing below is still open.

1. **`\` is the force-literal hatch.** Accepted as proposed (D3).
2. **Follow the spec: NOAD-first, word count is not a signal.** Multi-word
   headwords keep working — `hot dog` and `a priori` are lookups because the
   dictionary has them. The `≥5 words` arm is dropped as a consequence; see the
   note under D4 for the cost that buys and the one line that would undo it.
3. **`Store.UserModel()` lands here** (D7). Operator: no preference, so the seam
   wins over an `os.ReadFile` at the call site — the store already owns which
   directory this session is, and #17 inherits the seam and its conformance rows
   rather than adding a second answer beside it (ARCH-DRY).

## Revisions

### 2026-08-23 — the classifier has three arms, not four

**Reason:** operator direction, "let's follow the spec", answering the open
question about the length backstop.

**Delta:** `readsAsQuestion` loses the `≥5 words` arm and the `longLineWords`
constant. D4's table drops the row and gains a note naming what that costs. The
fixture row `difference between sycophantic and obsequious please` flips from
`true` to `false`, and the end-to-end decision table in Task 3 gains a row
asserting the same line routes to `not-found` — so the limit is pinned by a test
rather than remembered.

### 2026-08-23 — plan-quality round 1 (PQ-1 … PQ-5)

**Reason:** `sdlc change-code` plan gate, four blocking findings and one minor.
Ledger: `workshop/plans/000016-console-qa-plan-gate.md`.

**Delta:**

- **PQ-1 (Critical) — addressed.** D5 rewritten. The draft assumed Ctrl-C is a
  byte in raw mode; the pty suite's own header records the measurement that a
  `\x03` reached define as a **SIGINT** and left through `NotifyContext`, which
  would have ended the session mid-answer whatever the scoped sink pointed at.
  The sink is now the single answer with **both** transports feeding it, and the
  interactive loop detaches from the signal context
  (`context.WithoutCancel`) so nothing cancels it behind the sink's back. Task 7
  wires all three pieces and tests both transports against one observable; Task
  11 adds the pty row that only a surviving session can produce.
- **PQ-2 (Important) — addressed.** `session` is created and wired in **Task 4**
  (M1), not assumed: it replaces `repl.go:125`'s `current`, `runEditor`'s
  `current`, and `submitLine`'s `current *string`. `exchange`/`turns` are
  explicitly M2's, in Task 9.
- **PQ-3 (Important) — addressed.** Task 4 states the ask outcome's contract as
  a table — no `current`, no capture, yes recall, its own exit code — with a
  test per row, and routes the forced and unforced asks into **one**
  `askInSession` closure so Task 11's interrupter and `crlfWriter` wiring exists
  once (the ARCH-DRY half of the finding).
- **PQ-4 (Important) — addressed.** `runAsk`'s taxonomy now leads with
  `ctx.Err() != nil` → print nothing, code 0. The finding's sharpest edge is
  recorded with it: `mapError` classifies a statusless failure as
  `ErrUnavailable`, so a cancelled stream would otherwise print "no model
  configured" as the answer to the user's own keypress. The guard therefore asks
  the context, not the error.
- **PQ-5 (Minor) — addressed.** Task 10 uses the real `llmtest` API (`fake.URL`
  as an embedded field, `Script`, `Requests() []Recorded` with
  `Prompt()`/`System()`), and D1 cites `TestCaptureArityIsOnePerLookup`
  (`capture_test.go:89`).

### 2026-08-23 — plan-quality round 2 (PQ-6, and PQ-5's class)

**Reason:** `sdlc change-code` round 2. PQ-1..PQ-4 accepted as addressed; PQ-5
came back **not-addressed** (instance fixed, class not swept) and PQ-6 opened as
the second finding in family `interrupt-delivery-path`.

**Delta:**

- **PQ-6 (Important) — addressed, as a rule rather than a patch.** Round 1 put
  the `WithoutCancel` detach in `run`'s `case 0:` and the signal watcher in
  `replRaw`. But `case 0:` calls `repl` (`main.go:314-321`), which chooses
  between `replRaw` and `replLines` (`repl.go:95,97`) — so every piped,
  redirected or raw-mode-fallback run would have been detached from
  `NotifyContext` and given no watcher, leaving `replLines`' `ctx.Done()`
  (`repl.go:132`) unreachable and the loop **uninterruptible**. The rule now
  stated in D5: *the detach and the sink are installed together, at the point
  the loop is chosen*, and D5 carries the full loop × transport enumeration
  rather than naming transports for one loop. Task 7 moves both into `repl`,
  and adds `TestThePipedLoopStillExitsOnASignal` — the path the wrong wiring
  strands silently.
- **PQ-5 (Minor) — addressed as a class.** Every `file.go:NNN` citation in the
  plan was extracted and resolved against the tree mechanically, not re-read by
  eye; `repl.go:145` → `:125`, `replraw.go:71` → `:69`, `main.go:318-324` →
  `:314-321`, and the second `TestCaptureHappensOncePerLookup` →
  `TestCaptureArityIsOnePerLookup`. Every named existing symbol
  (`TestEditorLoopCtrlCExitsZero`, `TestPTYCtrlCDuringPlaybackExitsPromptly`,
  `storetest.Suite`, `llmtest.NewFake`, `AssertGolden`, `decideCapture`,
  `completionsFor`, `menuLines`, `newCommandCtx`, `TestParseREPLLine`) was
  confirmed to exist. The enumeration is what disposes of the family — a line
  number fixed by eye is the instance again.

### 2026-08-23 — M2's design departures, and the M1 rounds

**Reason:** the plan gate's own rule, restated by the M2 boundary review (I9):
the plan artifact's state is part of the milestone deliverable, and a
`## Revisions` entry is owed for every fork taken differently from the plan —
in the milestone-close commit, not at issue close. M1's four boundary rounds
went unrecorded here, which is the same gap.

**M1 rounds (BR-1…BR-19), recorded late.** Four rounds; every finding fixed
rather than deferred. The through-line is one rule the plan did not state and
should have: *an assertion must be able to fail.* Four separate findings in
family `test-asserts-nothing` — a branch deletable with the suite green, a
recall test satisfied by the editor's own echo, an injected double `withStore`
discarded, and a message asserted against the production constant rather than
its bytes (which shipped a doubled backslash). The full round-by-round record is
in the issue's `## Log` and the gate ledger.

**M2 departures from the plan as written:**

1. **`renderAskPrompt` returns an `llm.Request`, not `(system, prompt string)`.**
   `llmtest.AssertGolden` snapshots a Request through the same renderer the
   transport hashes, so the golden is what gets SENT rather than a string the
   test concatenates. Strictly better, and it composes with the cassette key.
2. **The interrupt swallow lives in the READER, not the loop.** Task 11 sketched
   the loop reading keys during the stream and swallowing the interrupt there.
   That works and silently eats type-ahead: keys typed during a long answer
   vanish. `interrupter.Fire` now reports whether a scope consumed the
   interrupt and `readKeys` drops it, which made the key channel's buffering
   load-bearing — the loop stops reading during a stream, so on an unbuffered
   channel the reader blocks on the first key typed and never decodes the
   Ctrl-C behind it.
3. **The `asked` event goes through the `Capturer` seam, not `store.AppendEvent`.**
   The plan's Integration-points table named `askCapturer`; the first
   implementation appended directly through `deps.deck`, creating a second write
   path into the event log beside `capture` and falsifying `deps`' own comment.
   `Capturer` gained `CaptureAsk`, so the log keeps one writer (M2 boundary
   review, I3).

### 2026-08-24 — the entity tables, resolved mechanically

**Reason:** M2 boundary round 3 (BR-47), the 3rd finding in `plan-contract-drift`.
Round 2 fixed two table rows by eye and the next round found three more
discrepancies, which is the definition of fixing instances.

**Delta:** the tables above are now resolved against the enumeration the code
actually produces, rather than against the rows a finding named:

```
git diff <boundary>..HEAD -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) '
```

Run at this boundary it lists 29 additions; the three the tables were missing or
wrong about were `recentDeck` (a new pure entity, extracted in answer to a
Critical, absent entirely), `askCapturer` (the method is `CaptureAsk`), and
`session.words`. `askScoped` and `consumed`, created by rounds 2 and 3, are
added by the same pass rather than waiting to be named by a round 4.

**The rule this encodes:** at a boundary close the Core-concepts tables are a
CONSUMER of the diff, so they are resolved against it mechanically — the way
PQ-5 resolved every file:line citation. A row corrected because a review named
it is the instance again.

### 2026-08-24 — the enumeration, run against the tree being committed

**Reason:** BR-47, four rounds, `not-addressed` three times. Each attempt ran the
command and then kept editing, so the commit that ran it created entities it did
not list — and the third attempt's entry even said "runs LAST" while reporting a
count taken before the last two functions existed.

**What was actually wrong** was never the command. Two things:

1. **"Last" has to mean against the WORKING TREE**, not `base..HEAD`. Run against
   HEAD it lists what the previous commits added, which is a different set from
   what this commit ships — it showed a function this very change had renamed.
2. **The entry must not carry a count.** A count is a measured claim that drifts
   the moment anything is added, and it drifts silently. Same rule BR-56 states
   for taxonomy messages and BR-55 for mutation tables: name the check, not the
   number it produced once.

**The check, to be re-run at any future boundary:**

```
git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) '
```

No `..HEAD`: comparing the boundary to the working tree is what makes it a check
of the thing being shipped. Its output is reconciled against the two tables
above, and this pass added the rows for `openerStem`, `recallLine`,
`nothingSays`, `lookupOutcome`, `question`, `mayAsk`, `carriedCR`,
`writeBytesAtomic`, `SetUserModel` and the two degradation messages.
