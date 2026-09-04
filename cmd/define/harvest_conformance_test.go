//go:build conformance

package main

// Live conformance for banding (ARCH-MOCK, applied to judgment).
//
// THE FLOOR LIVES HERE, and that placement is the point. The unit test owns
// agreement's arithmetic on synthetic input; a fake seeded to vary would only
// report how the fake was seeded. The fraction says something about the MODEL
// only when the model is the real one, so the number Done-when 3 asks for is
// asserted against the live service and nowhere else.
//
// Cadence is on-demand with the rest of the conformance suite:
//
//	go test -tags conformance -run Band ./cmd/define/
//
// It SKIPS rather than fails when the seam is unreachable: "not running" is not
// "wrong", and a check that reddens on a flat network is one people learn to
// ignore.

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/conformance"
	"github.com/xianxu/tools/internal/llm"
)

// agreementFloor is the stability the cache's premise requires.
//
// 0.8 over five assignments means at most one dissenting answer per word. Below
// that, "assign once and reuse forever" is picking a band at random from a
// distribution, and the honest response is not to lower this number — it is to
// build the hand-labelled sample the issue defers.
const agreementFloor = 0.8

// A spread of registers and levels, because a floor measured only on words the
// model finds easy is a floor measured on nothing. `run` and `set` are here
// deliberately: common polysemous words are where banding should be least stable.
// Every one is in the committed corpus, so the gloss the production path sends
// is actually present — a word the fake dictionary lacks would silently fall
// back to the bare shape this row exists to stop measuring.
var bandingWords = []string{
	"sycophantic", "ephemeral", "defenestrate", "quokka",
	"run", "set", "record", "alewife",
}

func TestBandingIsStableAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}
	client := llm.New(cfg)
	// THE PROMPT PRODUCTION SENDS, not a bare word.
	//
	// bandTask exists so --harvest and the measurement mode cannot ask different
	// questions, and the first cut of this row broke that from the outside: it
	// passed an empty gloss and no known domain, so it floored a shape --harvest
	// essentially never sends — every English deck word in NOAD has a gloss. The
	// milestone's one MEASURED claim came from the glossless prompt.
	//
	// testDict is in-package and unconstrained by the build tag, so the row can
	// derive exactly what runHarvest derives.
	dict := testDict(t)

	const rounds = 5
	total := 0.0
	for _, w := range bandingWords {
		var gloss string
		var known store.Domain
		if raw, err := dict.Lookup(w); err == nil {
			gloss, known = senseFacts(w, ParseEntry(raw))
		}
		bands := make([]store.Band, 0, rounds)
		for range rounds {
			claim, err := llm.Run(t.Context(), client, bandTask(store.DefaultLang, w, gloss, known))
			if err != nil {
				conformance.SkipOrFail(t, "banding "+w, err)
			}
			// Off-scale answers go in AS THEMSELVES rather than being dropped, so
			// a model that refuses the scale scores as unstable — which is what it
			// is. Dropping them would let a model answering "B2+" four times out
			// of five score a perfect 1.00 on the one time it complied.
			bands = append(bands, store.Band(claim.Band))
		}
		a := agreement(bands)
		t.Logf("%-14s agreement %.2f  %v  (gloss %t, domain %q)", w, a, bands, gloss != "", known)
		total += a
	}

	mean := total / float64(len(bandingWords))
	t.Logf("mean agreement %.2f over %d words x %d assignments", mean, len(bandingWords), rounds)
	if mean < agreementFloor {
		t.Errorf("banding agreement %.2f is below the floor of %.2f.\n"+
			"A band is assigned ONCE and reused forever, so this is the property the whole "+
			"cache rests on. The fix is not to lower the floor: it is the hand-labelled "+
			"sample the issue defers, because unstable banding and confidently-wrong "+
			"banding look identical from here.", mean, agreementFloor)
	}
}

// The other half of a conformance row: the SHAPE the fake assumes is the shape
// the service actually returns. A unit suite driven entirely by a fake cannot
// see the two disagreeing.
func TestBandClaimShapeAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}

	dict := testDict(t)
	raw, err := dict.Lookup("record")
	if err != nil {
		t.Fatalf("record is not in the committed corpus: %v", err)
	}
	gloss, known := senseFacts("record", ParseEntry(raw))
	claim, err := llm.Run(t.Context(), llm.New(cfg), bandTask(store.DefaultLang, "record", gloss, known))
	if err != nil {
		conformance.SkipOrFail(t, "banding record", err)
	}
	if _, ok := store.ParseBand(claim.Band); !ok {
		t.Errorf("the live service answered band %q, which ParseBand refuses — "+
			"every unit test in this package assumes a value on the scale", claim.Band)
	}
	// The domain half must land in the CLOSED set when the prompt enumerated it.
	// A service answering free text here is how topicSpread would come to count
	// values the parse then flattens to `general`, reporting a spread of 1.00
	// forever while looking healthy.
	if _, ok := store.ParseDomain(claim.Domain); !ok {
		t.Errorf("the live service answered domain %q, which is outside the closed set "+
			"the prompt enumerated; it will degrade to general", claim.Domain)
	}
}
