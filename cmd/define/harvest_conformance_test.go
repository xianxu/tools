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
	"strings"
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

// M2's three tasks against the live service (ARCH-MOCK).
//
// The plan committed to "a live conformance row each" and M2 shipped with none —
// on the milestone whose JUDGES are the whole product. A unit suite driven
// entirely by llmtest.Fake cannot see the fake and the service disagreeing about
// a schema, and a vetoVerdict whose `fits` field the service stops emitting is a
// missing-required error only if something reads a real body.

func TestAuthoredStemShapeAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}
	client := llm.New(cfg)

	// A RATE, not a never. The live model DOES sometimes return a stem with the
	// word already blanked, against an explicit instruction — measured here the
	// first time this row ran: "dismissed the biography as a ___ portrait of
	// Rupert Murdoch". That is why stemUsesTheWord exists and runs before either
	// judge, so production already handles it.
	//
	// What this row is for is the RATE: if the prompt ever regresses so far that
	// most stems come back unusable, authoring silently stops producing material
	// while every unit test stays green. Asserting "never" would instead redden
	// on a behaviour we have already defended against.
	words := []string{"sycophantic", "ephemeral", "keel", "bailiwick", "quokka", "potassium"}
	usable := 0
	for _, w := range words {
		got, err := llm.Run(t.Context(), client, authorTask(store.DefaultLang, w, "",
			store.WordFacts{Band: store.C1, Domain: store.DomainGeneral}, learnerFacts{}))
		if err != nil {
			conformance.SkipOrFail(t, "authoring "+w, err)
		}
		ok := stemUsesTheWord(got.Stem, w)
		if ok {
			usable++
		}
		t.Logf("%-12s usable=%-5v %q", w, ok, got.Stem)
	}

	// Two thirds. Below that the guard is throwing away more material than it is
	// protecting, and the prompt is the thing to fix rather than the threshold.
	if usable*3 < len(words)*2 {
		t.Errorf("only %d of %d live stems were usable; the author prompt is producing material "+
			"the deterministic guard rejects, so authoring would silently stall", usable, len(words))
	}
}

// THE COMMITTED KNOWN-BAD PAIR, live. The checkpoint showed the real model gets
// this right in both directions across three batches, so it is cheap — and it is
// the row that proves the veto is a real check rather than a fixture.
func TestTheVetoRejectsANearSynonymAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}
	client := llm.New(cfg)
	const stem = "The Times of London dismissed Oliver Stone's Putin interviews as sycophantic, " +
		"and the Kremlin reprinted long extracts within the week."

	near, err := llm.Run(t.Context(), client, vetoTask(store.DefaultLang, "sycophantic", stem, "obsequious"))
	if err != nil {
		conformance.SkipOrFail(t, "vetoing obsequious", err)
	}
	if !near.Fits {
		t.Errorf("the live veto passed `obsequious` beside `sycophantic` (%q) — a question with "+
			"two correct options teaches nothing, and this pair is the case the veto exists for",
			near.Reason)
	}

	// And it must not reject EVERYTHING: a veto that always fires would leave
	// every word unauthored, and the unit suite cannot tell the two apart.
	far, err := llm.Run(t.Context(), client, vetoTask(store.DefaultLang, "sycophantic", stem, "quokka"))
	if err != nil {
		conformance.SkipOrFail(t, "vetoing quokka", err)
	}
	if far.Fits {
		t.Errorf("the live veto rejected `quokka` as also fitting (%q); it is rejecting everything",
			far.Reason)
	}
}

// The entailment judge, live, on the stem the Spec opens with and on one the
// checkpoint produced. Both directions, for the same reason as the veto's row.
func TestTheEntailJudgeAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}
	client := llm.New(cfg)

	for _, tc := range []struct {
		name, stem  string
		wantEntails bool
		wantGloss   bool
	}{
		{"the Spec's own bad stem", "His sycophantic behaviour was noted by all.", false, false},
		{
			"a glossed stem, which ENTAILS and must still be caught",
			"Each spring biologists count the alewife, the small silver herring that leaves the " +
				"Atlantic to spawn upstream in fresh water.",
			true, true,
		},
		{
			"a stem the checkpoint produced",
			"The Times of London dismissed Oliver Stone's Putin interviews as sycophantic, and the " +
				"Kremlin reprinted long extracts within the week.",
			true, false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			word := "sycophantic"
			if strings.Contains(tc.stem, "alewife") {
				word = "alewife"
			}
			got, err := llm.Run(t.Context(), client, entailTask(store.DefaultLang, word, tc.stem))
			if err != nil {
				conformance.SkipOrFail(t, "judging "+word, err)
			}
			t.Logf("entails=%v glosses=%v named=%v — %s", got.Entails, got.Glosses, got.Named, got.Reason)
			if got.Glosses != tc.wantGloss {
				t.Errorf("glosses = %v, want %v (%s)", got.Glosses, tc.wantGloss, got.Reason)
			}
			if !tc.wantGloss && got.Entails != tc.wantEntails {
				t.Errorf("entails = %v, want %v (%s)", got.Entails, tc.wantEntails, got.Reason)
			}
		})
	}
}
