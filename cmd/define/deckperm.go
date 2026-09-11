package main

import (
	"bufio"
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
	ask   func() bool
	state deckDecision
}

func newDeckPermission(ask func() bool) *deckPermission {
	return &deckPermission{ask: ask}
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
	switch p.state {
	case deckAllow:
		return true, true
	case deckDeny:
		return false, true
	default:
		return false, false
	}
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
func deckAsker(dir string, opt options, in io.Reader, out io.Writer, stdinIsTerminal func() bool) func() bool {
	return func() bool {
		if store.IsDeck(dir) {
			return true
		}
		if opt.here {
			return true
		}
		if stdinIsTerminal == nil || !stdinIsTerminal() {
			// SAID, not silent. Someone piping into define in a fresh directory
			// gets a working lookup and no deck; without this line they would
			// never learn why nothing was saved.
			fmt.Fprintf(out, "define: %s is not a deck and there is no terminal to ask; "+
				"nothing will be saved (use --here to create one)\n", dir)
			return false
		}
		fmt.Fprintf(out, "define: %s is not a deck yet. Create one here? [y/N] ", dir)
		answer, err := bufio.NewReader(in).ReadString('\n')
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
