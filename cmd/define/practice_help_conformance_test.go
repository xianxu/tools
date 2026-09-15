//go:build conformance

package main

// Live conformance for English practice help (#61, ARCH-MOCK).
//
// The fake answers whatever a test scripts, so it cannot say whether a REAL
// translation keeps a cloze's blank, leaves the hidden word hidden, or picks
// the Spanish sense of a word that is also English. Only the live model can,
// and the checks here are the production ones: helpTask builds the request and
// acceptHelp decides what a learner would see.
//
// Cadence is the plan's PQ-2 policy: before a release that changes the help
// prompt, and after the proxy's model changes.
//
//	go test -tags conformance -run PracticeHelp ./cmd/define/
//
// It SKIPS when the seam is unreachable, unless CONFORMANCE_STRICT is set; a
// reply that arrives and fails the checks is a failure, not a skip.

import (
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
	"github.com/xianxu/tools/internal/llm"
)

func TestPracticeHelpAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}
	client := llm.New(cfg)
	batch := []helpKey{
		{"es", helpCloze, "Para llegar a la estación, Elena tiene que ___ cada lunes."},
		// `red` is Spanish for net as well as an English word: the Spanish sense
		// must survive translation.
		{"es", helpGloss, "red: conjunto de hilos o cuerdas entrelazados que forman una malla"},
		{"es", helpGloss, "Mueble compuesto de un tablero horizontal sostenido por una o varias patas."},
	}
	reply, err := llm.Run(t.Context(), client, helpTask("es", batch))
	if errors.Is(err, llm.ErrUnavailable) {
		conformance.SkipOrFail(t, "the model is unreachable", err)
	}
	if err != nil {
		t.Fatalf("the help task failed against the live service: %v", err)
	}
	needs := [][]helpNeed{{{key: batch[0], answer: "madrugar"}}, {{key: batch[1], slot: 0}}, {{key: batch[2], slot: 0}}}
	accepted := acceptHelp(needs, readHelpReply(batch, reply))
	if len(accepted) != len(batch) {
		t.Fatalf("accepted %d of %d translations: %+v", len(accepted), len(batch), reply)
	}
	if got := strings.ToLower(accepted[batch[1]]); !strings.Contains(got, "net") && !strings.Contains(got, "mesh") {
		t.Errorf("red (net) was translated as %q", accepted[batch[1]])
	}
	if got := strings.ToLower(accepted[batch[2]]); !strings.Contains(got, "furniture") && !strings.Contains(got, "table") {
		t.Errorf("the furniture gloss was translated as %q", accepted[batch[2]])
	}
	t.Logf("cloze: %s", accepted[batch[0]])
}
