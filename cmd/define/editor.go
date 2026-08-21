package main

import (
	"fmt"
	"strings"
)

// Editor is the whole editing state for one line. No IO, no history handle —
// candidates are passed in, so Apply is pure over plain data and no store query
// runs per keystroke once #3 fills the History seam.
type Editor struct {
	Line   []rune
	Cursor int

	// draft is what was typed before an Up-arrow walk began, restored when the
	// walk returns past the newest entry. Losing it is the classic history bug.
	draft    []rune
	walkIdx  int // -1 = not walking
	walkBase string
}

// NewEditor returns an editor with no walk in progress.
func NewEditor() Editor { return Editor{walkIdx: -1} }

// Action is what the LOOP must do next. The editor never acts itself.
type Action int

const (
	ActNone Action = iota
	ActSubmit
	ActInterrupt
	ActEOF
)

// Apply is the state machine: one key in, the next editor state out.
//
// matches is a newest-first, deduped snapshot of history entries that begin with
// whatever the walk is anchored on — resolved by the caller.
func Apply(e Editor, k Key, matches []string) (Editor, Action) {
	switch k.Kind {
	case KeyRune:
		e = e.stopWalk()
		e.Line = append(e.Line[:e.Cursor:e.Cursor], append([]rune{k.Rune}, e.Line[e.Cursor:]...)...)
		e.Cursor++
	case KeyBackspace:
		e = e.stopWalk()
		if e.Cursor > 0 {
			e.Line = append(e.Line[:e.Cursor-1:e.Cursor-1], e.Line[e.Cursor:]...)
			e.Cursor--
		}
	case KeyDelete:
		e = e.stopWalk()
		if e.Cursor < len(e.Line) {
			e.Line = append(e.Line[:e.Cursor:e.Cursor], e.Line[e.Cursor+1:]...)
		}
	case KeyLeft:
		if e.Cursor > 0 {
			e.Cursor--
		}
	case KeyRight:
		// At end of line, Right ACCEPTS the suggestion; elsewhere it moves.
		if e.Cursor == len(e.Line) {
			return acceptSuggestion(e, matches)
		}
		e.Cursor++
	case KeyHome:
		e.Cursor = 0
	case KeyEnd:
		if e.Cursor == len(e.Line) {
			return acceptSuggestion(e, matches)
		}
		e.Cursor = len(e.Line)
	case KeyTab:
		// Tab accepts the suggestion too. Right/End are the zsh-autosuggestions
		// bindings, but Tab is what a hand reaches for after typing a prefix —
		// and #15's command mode gives Tab a complementary job (completing a
		// /command), not a conflicting one.
		return acceptSuggestion(e, matches)
	case KeyUp:
		return walk(e, matches, +1), ActNone
	case KeyDown:
		return walk(e, matches, -1), ActNone
	case KeyEnter:
		return e, ActSubmit
	case KeyInterrupt:
		return e, ActInterrupt
	case KeyKillLine:
		// Clears the whole line, not just back to the cursor: the gesture people
		// reach for is "start over", and a partial kill from mid-word would leave
		// a tail they did not ask to keep. Also ends any history walk, so the
		// next Up starts from the full list rather than mid-traversal.
		e = e.stopWalk()
		e.Line, e.Cursor = nil, 0
	case KeyEOF:
		if len(e.Line) == 0 { // Ctrl-D ends the session only on an empty line
			return e, ActEOF
		}
	}
	return e, ActNone
}

// String returns the typed text — and ONLY the typed text. Submitting must never
// include the suggestion: looking up a word the user did not ask for is the most
// damaging thing this editor could do.
func (e Editor) String() string { return string(e.Line) }

// WalkBase is the prefix a history walk is anchored on: whatever was typed when
// the walk began, so Up narrows to matching entries rather than walking all.
func (e Editor) WalkBase() string {
	if e.walkIdx >= 0 {
		return e.walkBase
	}
	return string(e.Line)
}

func (e Editor) stopWalk() Editor { e.walkIdx = -1; return e }

func walk(e Editor, matches []string, dir int) Editor {
	if e.walkIdx < 0 { // starting: remember the draft and what we are anchored on
		e.draft = append([]rune(nil), e.Line...)
		e.walkBase = string(e.Line)
		e.walkIdx = -1
	}
	next := e.walkIdx + dir
	switch {
	case next < -1:
		return e // already at the draft; Down does nothing
	case next == -1: // returned past the newest: restore what was typed
		e.walkIdx = -1
		e.Line = append([]rune(nil), e.draft...)
		e.Cursor = len(e.Line)
		return e
	case next >= len(matches):
		return e // exhausted; no wrap
	}
	e.walkIdx = next
	e.Line = []rune(matches[next])
	e.Cursor = len(e.Line)
	return e
}

// Suggestion is the grey tail: the newest history entry starting with the typed
// text, minus that text. Empty when nothing matches, when the line is empty, or
// when the cursor is not at the end.
func Suggestion(e Editor, matches []string) string {
	if len(e.Line) == 0 || e.Cursor != len(e.Line) || e.walkIdx >= 0 {
		return ""
	}
	typed := string(e.Line)
	for _, m := range matches {
		if len(m) > len(typed) && m[:len(typed)] == typed {
			return m[len(typed):]
		}
	}
	return ""
}

func acceptSuggestion(e Editor, matches []string) (Editor, Action) {
	if s := Suggestion(e, matches); s != "" {
		e.Line = append(e.Line, []rune(s)...)
		e.Cursor = len(e.Line)
	}
	return e, ActNone
}

// RenderLine draws the whole input line as ONE frame: return to column 0, clear,
// then prompt + typed text + grey suggestion, then park the cursor after the
// typed text.
//
// A whole frame rather than a partial update is the point. #2 placed its
// indicator with cursor arithmetic against a terminal that had already echoed
// Enter, and documented that it breaks if the user types during playback. In raw
// mode nothing is echoed and the frame is simply redrawn, so that arithmetic —
// eraseLineAndStepBack, skipPrompt — is deleted rather than ported.
func RenderLine(e Editor, sug string, color bool) string {
	var b strings.Builder
	b.WriteString(eraseLine)
	if color {
		// The input line has to be findable in a screen full of definition text.
		// The prompt gets an accent colour and the typed word is bold, so the one
		// line you can act on reads differently from everything you cannot.
		b.WriteString(promptOn + prompt + sgrOff)
		b.WriteString(inputOn + string(e.Line) + sgrOff)
	} else {
		b.WriteString(prompt)
		b.WriteString(string(e.Line))
	}
	if sug != "" {
		if color {
			b.WriteString(greyOn + sug + greyOff)
		} else {
			b.WriteString(sug)
		}
	}
	// Park the cursor after the typed text: the suggestion sits ahead of it, and
	// typing must overwrite the suggestion rather than append to it.
	if back := len([]rune(sug)) + (len(e.Line) - e.Cursor); back > 0 {
		fmt.Fprintf(&b, "\x1b[%dD", back)
	}
	return b.String()
}

// greyOn/greyOff render the suggestion as dim text — the zsh-autosuggestions
// convention: visibly present, visibly not yet yours.
const (
	greyOn  = "\x1b[90m"
	greyOff = "\x1b[0m"

	promptOn = "\x1b[1;36m" // bold cyan — the marker for "this line is yours"
	inputOn  = "\x1b[1m"    // bold — what you have typed, against dim suggestion
	sgrOff   = "\x1b[0m"
)
