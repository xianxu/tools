package main

import (
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
	// or no terminal at all. It returns deckUndecided when only a question can
	// settle it. Optional: a nil quiet simply never settles anything early.
	quiet func() deckDecision
	state deckDecision
}

func newDeckPermission(ask func() bool) *deckPermission {
	return &deckPermission{ask: ask}
}

// withQuiet attaches the no-question half of the policy.
func (p *deckPermission) withQuiet(quiet func() deckDecision) *deckPermission {
	p.quiet = quiet
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
	if d := p.quiet(); d != deckUndecided {
		p.state = d
	}
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

// deckPolicy answers what can be answered WITHOUT putting a question, and
// returns deckUndecided when only the learner can settle it.
//
// SPLIT OUT OF deckAsker because three of the four inputs need nobody: an
// existing deck, --here, and the absence of a terminal are all facts about the
// world. Only the fourth is a question. Separating them is what lets --stats be
// HONEST without becoming intrusive: it settles the free cases and leaves the
// one that would prompt alone (#50, found in smoke testing — `--stats` in a
// piped non-deck directory was printing "look a word up and it joins your deck",
// which is exactly the promise this feature exists to stop making).
func deckPolicy(dir string, opt options, stdinIsTerminal func() bool) deckDecision {
	if store.IsDeck(dir) || opt.here {
		return deckAllow
	}
	if stdinIsTerminal == nil || !stdinIsTerminal() {
		return deckDeny
	}
	return deckUndecided
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
		// ONE ENCODING OF THE PRECEDENCE (#50 BR-12). This used to re-implement
		// already-a-deck / --here / no-terminal alongside deckPolicy's copy. They
		// agreed, and nothing made them: the observable consequence was that only
		// this copy printed the "nothing will be saved" explanation, so whether a
		// piped learner was told depended on which encoding settled the state
		// first — `echo word | define` printed it and `define --forget cat` piped
		// did not.
		switch deckPolicy(dir, opt, stdinIsTerminal) {
		case deckAllow:
			return true
		case deckDeny:
			// SAID, not silent. Someone piping into define in a fresh directory
			// gets a working lookup and no deck; without this they never learn why
			// nothing was saved.
			fmt.Fprintf(out, "define: %s is not a deck and there is no terminal to ask; "+
				"nothing will be saved (use --here to create one)\n", dir)
			return false
		}

		fmt.Fprintf(out, deckPrompt, dir)
		answer, err := readLineUnbuffered(in)
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
