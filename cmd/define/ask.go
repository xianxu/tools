package main

import (
	"fmt"
	"io"
)

// askUnavailable is what a question gets when there is no model to answer it.
//
// The message says what happened rather than what failed: the line was
// understood as a question, and there is nowhere to send it. Attempting a
// dictionary lookup of a sentence instead — the pre-#16 behaviour — would report
// "not found" for something that was never a word.
//
// M1 routes every question here. M2 gives it a model, and this stays the
// degradation path (#16, "Seam unavailable").
func askUnavailable(w io.Writer, question string) int {
	fmt.Fprintf(w, "define: no model configured; `%s` is not a word\n", truncateQuestion(question))
	return 1
}
