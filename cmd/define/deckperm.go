package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// deckDecision is the three-valued answer to "may define write here".
//
// THREE VALUES, NOT A BOOL PLUS A BOOL. "Not yet asked" is a real state with its
// own behaviour — reads are served from disk, a write puts the question — and
// encoding it as `decided bool; allowed bool` makes {false, true} representable
// and meaningless. A tagged set of legal states is what ARCH-ORDER asks for in
// place of a boolean constellation.
type deckDecision int

const (
	deckUndecided deckDecision = iota
	deckAllow
	deckDeny
)

// deckPermission is THE decision for this process, resolved at most once.
//
// ONE INSTANCE, SHARED BY EVERYTHING THAT COULD WRITE, and the sharing is the
// design rather than an optimisation. A session builds TWO YAML stores — the
// flat one backing history and the news cache, and the per-language one — and
// `/lang` rebuilds more. Holding the decision inside a store wrapper would ask
// once PER WRAPPER: the learner answers, switches language, and is asked again
// about the same directory. Worse, a wrapper built after a switch would start
// undecided and throw away an answer already given.
//
// It is deliberately NOT persisted. The answer lives for the process; writing a
// "no" to disk would be new state to invalidate the moment the learner changes
// their mind, in a directory they have by definition not agreed to write to.
//
// NOT SAFE FOR CONCURRENT USE, and it does not need to be: every caller is on
// the command's own path, and the two loop shells resolve it before their reader
// goroutines start (that pre-resolution is why it is not a mutex — see repl.go).
type deckPermission struct {
	// ask puts the question. A nil ask allows, which is what makes a permission
	// optional at every seam that has not been told about the policy yet.
	ask func() bool
	// quiet settles the decision when it needs nobody — an existing deck, --here,
	// -raw, or no terminal at all. It returns deckUndecided when only a question
	// can settle it. Optional: a nil quiet simply never settles anything early.
	quiet func() (deckDecision, deckReason)
	// explain says WHY, once, when the state settles to a denial nobody was asked
	// about. It lives here rather than in deckAsker because either path can settle
	// first, and a learner should be told the same thing regardless of which did.
	explain   func(deckReason)
	explained bool
	state     deckDecision
}

func newDeckPermission(ask func() bool) *deckPermission {
	return &deckPermission{ask: ask}
}

// withQuiet attaches the no-question half of the policy and the explanation that
// belongs to it.
func (p *deckPermission) withQuiet(quiet func() (deckDecision, deckReason), explain func(deckReason)) *deckPermission {
	p.quiet, p.explain = quiet, explain
	return p
}

// settleQuietly resolves the decision ONLY if doing so asks nobody anything.
//
// It is what makes `--stats` honest without making it intrusive. Reading your
// figures in a piped, non-deck directory should say "nothing is being saved",
// because that is TRUE and knowable — no terminal means no deck, and no question
// was needed to learn it. What it must never do is settle the one case that would
// put a prompt on screen, which is why this is separate from allowed().
func (p *deckPermission) settleQuietly() {
	if p == nil || p.state != deckUndecided || p.quiet == nil {
		return
	}
	if d, why := p.quiet(); d != deckUndecided {
		p.state = d
		if d == deckDeny {
			p.sayWhy(why)
		}
	}
}

// sayWhy emits the explanation at most once per process, whichever path settled
// the state. Twice would mean a lookup that reads and then writes tells the
// learner the same thing two ways.
func (p *deckPermission) sayWhy(r deckReason) {
	if p.explained || p.explain == nil {
		return
	}
	p.explained = true
	p.explain(r)
}

// allowed resolves the decision, at most once, and answers it.
//
// A NIL RECEIVER ALLOWS. Every seam that predates this policy passes no
// permission, and must keep behaving exactly as it did — so "no policy" means
// "no restriction" rather than a panic or a refusal.
func (p *deckPermission) allowed() bool {
	if p == nil {
		return true
	}
	p.resolve()
	return p.state == deckAllow
}

// resolve forces the decision now, idempotently.
//
// It exists so the two REPL shells can settle the question before either starts
// reading stdin on its own goroutine. Without it the first write inside a
// session would put a question to a terminal whose keystrokes are already being
// consumed by the loop's reader, and the two would race for the answer.
func (p *deckPermission) resolve() {
	if p == nil || p.state != deckUndecided {
		return
	}
	// THE QUIET HALF FIRST, ALWAYS. Every path that settles the state runs the
	// same two steps in the same order, so a denial nobody was asked about is
	// explained identically whether a read, a write or the loop's pre-resolution
	// got there first. An earlier version had allowed() call settleQuietly and
	// resolve() not, which made the explanation depend on which path arrived —
	// the exact shape of the bug this was fixing (#50 BR-12).
	p.settleQuietly()
	if p.state != deckUndecided {
		return
	}
	if p.ask == nil || p.ask() {
		p.state = deckAllow
		return
	}
	p.state = deckDeny
}

// saving reports (allowed, decided) WITHOUT resolving.
//
// THE POINT IS THAT IT DOES NOT ASK. `--stats` renders an empty screen and has to
// say whether anything is being saved; learning that through allowed() would make
// putting the question a side effect of READING, so `define --stats` in the wrong
// directory would offer to create a deck it is never going to write to. That is
// the false alarm the whole lazy design exists to prevent, and false alarms train
// people to answer yes.
//
// An undecided permission reports (false, false): nothing is being saved YET and
// nothing has been settled. The caller renders the ordinary empty screen, because
// a directory nobody has asked about is exactly a directory where the usual "look
// a word up and it joins your deck" is still true.
func (p *deckPermission) saving() (allowed, decided bool) {
	if p == nil {
		return true, true
	}
	// Free answers only — never the one that would prompt.
	p.settleQuietly()
	switch p.state {
	case deckAllow:
		return true, true
	case deckDeny:
		return false, true
	default:
		return false, false
	}
}

// deckReason is WHY the decision came out the way it did.
//
// It exists because the decision alone is not enough to act on: a denial because
// there is no terminal deserves an explanation ("nothing will be saved, use
// --here"), and a denial because -raw writes nothing at all deserves silence. The
// same three-valued answer needs two different behaviours, so the reason travels
// with it (#50 BR-12, BR-14).
type deckReason int

const (
	reasonMustAsk deckReason = iota
	reasonAlreadyDeck
	reasonHere
	reasonRaw
	reasonNoTerminal
)

// deckPolicy answers what can be answered WITHOUT putting a question, and returns
// deckUndecided when only the learner can settle it.
//
// SPLIT OUT OF deckAsker because four of the five inputs need nobody: an existing
// deck, --here, -raw and the absence of a terminal are all facts about the world.
// Only the fifth is a question. Separating them is what lets --stats be HONEST
// without becoming intrusive: it settles the free cases and leaves the one that
// would prompt alone.
//
// -raw NEVER ASKS, because -raw RECORDS NOTHING — the usage text has said so
// since #2. Asking permission to create a deck that this invocation will not
// write to is the false alarm the lazy design exists to prevent, and it was
// reaching real users: `define -raw word` in a fresh directory put a question on
// screen, and piped it advised scripts to pass --here for a deck it would never
// have made (#50 BR-14).
func deckPolicy(dir string, opt options, stdinIsTerminal func() bool) (deckDecision, deckReason) {
	if store.IsDeck(dir) {
		return deckAllow, reasonAlreadyDeck
	}
	if opt.here {
		return deckAllow, reasonHere
	}
	if opt.raw {
		return deckDeny, reasonRaw
	}
	if stdinIsTerminal == nil || !stdinIsTerminal() {
		return deckDeny, reasonNoTerminal
	}
	return deckUndecided, reasonMustAsk
}

// deckAsker builds the question this directory needs, or the answer it already
// has.
//
// FOUR INPUTS, IN THIS ORDER, AND THE ORDER IS THE DESIGN:
//
//   - ALREADY A DECK -> yes, silently. There is nothing to ask about: define has
//     written here before, so the directory is already its own.
//   - --here -> yes, silently. This is the automation path, and it is
//     load-bearing rather than a nicety: since a non-terminal no longer gets a
//     deck, --here is the ONLY way a script can make one.
//   - NOT A TERMINAL -> no, with a warning. The accident this exists for is a
//     human in the wrong shell; a script names its directory deliberately. A
//     question nobody can answer must never become a hang.
//   - otherwise -> ask, and DEFAULT TO NO. A bare Enter declines, because the
//     cost of a wrong yes is a stray deck in someone's home directory and the
//     cost of a wrong no is re-running one command.
//
// The QUESTION goes to `out` (stderr) rather than stdout: a lookup's output is
// data someone may be redirecting, and a prompt in a redirected stream is a hang
// with no visible cause.
func deckAsker(ctx context.Context, dir string, opt options, in io.Reader, out io.Writer, stdinIsTerminal func() bool) func() bool {
	return func() bool {
		switch d, _ := deckPolicy(dir, opt, stdinIsTerminal); d {
		case deckAllow:
			return true
		case deckDeny:
			// The EXPLANATION is not printed here — see deckPermission.settle. A
			// read can settle the state before any write asks, in which case this
			// function is never called, and only one of the two paths said why.
			return false
		}

		fmt.Fprintf(out, deckPrompt, dir)
		answer, err := readLineCancellable(ctx, in)
		if errors.Is(err, context.Canceled) {
			// CTRL-C AT THE QUESTION DECLINES AND SAYS SO (#50 BR-15).
			//
			// Without an arm on ctx this read blocks forever: the question is put
			// before the loop's own reader exists, so there is nothing else
			// watching the terminal, and ^C left the program sitting there. An
			// interrupt is the clearest possible "no" a person can give, so it
			// means no — and it prints a newline first, because the cursor is
			// parked after the prompt.
			fmt.Fprintln(out)
			fmt.Fprintln(out, "define: not saving in this directory; the lookup still works.")
			return false
		}
		if err != nil && answer == "" {
			// EOF mid-question declines, for the same reason a bare Enter does:
			// the safe answer is the one that writes nothing.
			fmt.Fprintln(out)
			return false
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes":
			return true
		default:
			fmt.Fprintln(out, "define: not saving in this directory; the lookup still works.")
			return false
		}
	}
}

// explainDenial is what a learner is told when the answer was settled without
// them, and it is keyed on the REASON rather than on the decision.
//
// ONE PLACE, reached from wherever the state settles (#50 BR-12). The
// explanation used to live in deckAsker, so it was printed only when a WRITE
// resolved the question — and a READ settling it quietly first meant the same
// invocation said nothing at all. Measured on the real binary: `echo word |
// define` explained itself, `define word` with stdin redirected did not.
func explainDenial(r deckReason, dir string, out io.Writer) {
	switch r {
	case reasonNoTerminal:
		fmt.Fprintf(out, "define: %s is not a deck and there is no terminal to ask; "+
			"nothing will be saved (use --here to create one)\n", dir)
	case reasonRaw:
		// Silence is correct: -raw records nothing by design, so nothing is being
		// lost and there is nothing to act on.
	}
}

// deckPrompt is the question, in ONE place so the READMEs can be pinned against
// it rather than quoting a string nothing keeps in step (#50 BR-11).
const deckPrompt = "define: %s is not a deck yet. Create one here? [y/N] "

// readLineUnbuffered reads one line WITHOUT buffering ahead (#50 BR-6).
//
// bufio.Reader would be the obvious choice and is the wrong one here: `in` is the
// program's shared stdin, and a buffered reader pulls as much as it can get. The
// bytes it swallowed past the newline would be lost to whoever reads stdin next —
// which, since the loop shells resolve this question before they start, is the
// loop itself. The learner would answer "y" and watch the next line of input
// vanish.
//
// One byte at a time is cheap: this reads a single short answer, once per
// process, from a human.
func readLineUnbuffered(in io.Reader) (string, error) {
	var b []byte
	buf := make([]byte, 1)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return string(b), nil
			}
			b = append(b, buf[0])
		}
		if err != nil {
			return string(b), err
		}
	}
}

// readLineCancellable is readLineUnbuffered with an arm on the context.
//
// The read runs on its own goroutine so ctx.Done can win. The goroutine is
// ABANDONED rather than joined when the context fires — it is blocked in a read
// on the program's stdin, which cannot be interrupted, and the process is on its
// way out. What matters is that it writes only to its own channel, so nothing it
// does later can be observed by a caller that has already moved on.
func readLineCancellable(ctx context.Context, in io.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1) // buffered: the abandoned goroutine must not leak on send
	go func() {
		line, err := readLineUnbuffered(in)
		ch <- result{line, err}
	}()
	select {
	case r := <-ch:
		return r.line, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
