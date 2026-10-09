package main

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"testing"
	"time"
)

// checkCut asserts the timing model's properties for one cut. Each one names
// the clause of the contract it defends.
func checkCut(t *testing.T, in Cast, caps []Caption, tm Timing, out Cast) {
	t.Helper()
	const eps = 2e-3 // caption bounds are rounded to 3 decimals

	// (a) Input events survive in order; every other event is a caption
	// marker, except at most one trailing empty hold.
	ot := outTimes(out)
	inOut := make([]float64, 0, len(in.Events)) // output time of each input event
	markers := 0
	for j, e := range out.Events {
		switch k := len(inOut); {
		case k < len(in.Events) && e.Kind == in.Events[k].Kind && bytes.Equal(e.Data, in.Events[k].Data):
			inOut = append(inOut, ot[j])
		case e.Kind == "m" && isCaptionMarker(e, caps):
			markers++
		case j == len(out.Events)-1 && e.Kind == "o" && string(e.Data) == `""`:
		default:
			t.Fatalf("(a) out event %d %+v is neither input event %d nor a marker", j, e, k)
		}
	}
	if len(inOut) != len(in.Events) || markers != len(caps) {
		t.Fatalf("(a) %d/%d input events, %d markers for %d captions", len(inOut), len(in.Events), markers, len(caps))
	}

	// (b) Output time never goes backwards.
	for i, e := range out.Events {
		if e.Gap < 0 {
			t.Fatalf("(b) event %d has negative interval %v", i, e.Gap)
		}
	}

	// (c) Each caption plays for exactly its reading time, and
	// (e) captions are sorted, disjoint and at least min-hold long.
	hc := headerCaptions(t, out)
	ws := planWindows(caps, tm)
	if len(hc) != len(ws) {
		t.Fatalf("(c) %d header captions, %d windows", len(hc), len(ws))
	}
	for i, c := range hc {
		if !near(c.End-c.Start, ws[i].End-ws[i].Start, eps) {
			t.Fatalf("(c) caption %d lasts %v, its hold is %v", i, c.End-c.Start, ws[i].End-ws[i].Start)
		}
		if c.End-c.Start < tm.MinHold-eps {
			t.Fatalf("(e) caption %d shorter than min-hold: %+v", i, c)
		}
		if i > 0 && c.Start < hc[i-1].End-eps {
			t.Fatalf("(e) caption %d overlaps the previous: %+v after %+v", i, c, hc[i-1])
		}
	}

	// (d) Away from captions, a gap costs at most idle/speed of output time.
	h := make(map[string]json.RawMessage, len(in.Header))
	for k, v := range in.Header {
		h[k] = v
	}
	limit, _ := (&Cast{Header: h}).idleLimit()
	view := 0.0
	for i, e := range in.Events {
		gap := e.Gap
		if limit > 0 {
			gap = math.Min(gap, limit)
		}
		a, b := view, view+gap
		view = b
		if i == 0 || touchesWindow(a, b, ws) {
			continue
		}
		if d := inOut[i] - inOut[i-1]; d > tm.Idle/tm.Speed+1e-5 {
			t.Fatalf("(d) gap %d (view %v-%v) costs %v, bound %v", i, a, b, d, tm.Idle/tm.Speed)
		}
	}

	// (f), markers before output at one instant, is an example row:
	// TestCutCaptionPlaysInRealTimeWithMarkerFirst.
}

func isCaptionMarker(e Event, caps []Caption) bool {
	for _, c := range caps {
		if bytes.Equal(e.Data, jsonString(c.Text)) {
			return true
		}
	}
	return false
}

func touchesWindow(a, b float64, ws []window) bool {
	for _, w := range ws {
		if a < w.End && w.Start < b {
			return true
		}
	}
	return false
}

// randomTake draws a take that exercises the model's edges: zero gaps, gaps
// beyond the idle cap, duplicate and overlapping stamps, stamps at the very end.
func randomTake(r *rand.Rand, n int) (Cast, []Caption, Timing) {
	limit := 0.0
	if r.Intn(2) == 0 {
		limit = 0.5 + r.Float64()*3
	}
	gaps := make([]float64, n)
	view := 0.0
	for i := range gaps {
		switch r.Intn(5) {
		case 0:
			gaps[i] = 0
		case 1:
			gaps[i] = r.Float64() * 20
		default:
			gaps[i] = r.Float64() * 0.5
		}
		g := gaps[i]
		if limit > 0 {
			g = math.Min(g, limit)
		}
		view += g
	}
	c := castOf(limit, gaps...)
	var caps []Caption
	words := []string{"a", "few", "words", "<b>", "&", "é→✓", `"q"`}
	for k := r.Intn(6); k > 0; k-- {
		at := r.Float64() * view
		if r.Intn(4) == 0 && len(caps) > 0 {
			at = caps[len(caps)-1].At // duplicate stamp
		}
		if r.Intn(6) == 0 {
			at = view
		}
		text := words[r.Intn(len(words))]
		for j := r.Intn(12); j > 0; j-- {
			text += " " + words[r.Intn(len(words))]
		}
		caps = append(caps, Caption{at, text})
	}
	caps, _ = parseCaptions("", []byte(captionsText(caps)))
	tm := Timing{
		Speed: 0.5 + r.Float64()*10, Idle: r.Float64() * 2, Lead: r.Float64() * 2,
		MinHold: r.Float64() * 5, WPS: 0.5 + r.Float64()*5, Beat: r.Float64() * 2,
	}
	return c, caps, tm
}

// captionsText renders captions the way annotate writes them, so random
// captions go through the real parser (and its sort).
func captionsText(caps []Caption) string {
	var b bytes.Buffer
	for _, c := range caps {
		m := int(c.At / 60)
		b.WriteString("~")
		b.WriteString(itoa(m))
		b.WriteString(":")
		b.WriteString(ftoa(c.At - float64(m*60)))
		b.WriteString("  ")
		b.WriteString(c.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func TestCutProperties(t *testing.T) {
	r := rand.New(rand.NewSource(83))
	for i := 0; i < 500; i++ {
		c, caps, tm := randomTake(r, 1+r.Intn(60))
		out, _, err := Cut(c, caps, tm)
		if err != nil {
			// Only a stamp rounded past the end may refuse.
			continue
		}
		t.Run("", func(t *testing.T) { checkCut(t, c, caps, tm, out) })
	}
}

// naiveWarp is the prototype's warp: every segment from the start, per call.
func naiveWarp(segs []segment, x float64) float64 {
	out := 0.0
	for _, s := range segs {
		if x <= s.P {
			break
		}
		out += (math.Min(x, s.Q) - s.P) * s.Rate
	}
	return out
}

func TestWarpMatchesTheNaiveReference(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		c, caps, tm := randomTake(r, 1+r.Intn(40))
		times := make([]float64, len(c.Events))
		v := 0.0
		for j, e := range c.Events {
			v += e.Gap
			times[j] = v
		}
		segs := buildSegments(times, planWindows(caps, tm), tm)
		w := newWarper(segs)
		for k := 0; k < 50; k++ {
			x := r.Float64() * (v + 1)
			if k%5 == 0 && len(segs) > 0 {
				x = segs[r.Intn(len(segs))].P
			}
			if a, b := w.warp(x), naiveWarp(segs, x); !near(a, b, 1e-9) {
				t.Fatalf("warp(%v) = %v, naive %v", x, a, b)
			}
		}
	}
}

func TestCutStaysInsideItsEnvelope(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	c, _, _ := randomTake(r, 200_000)
	delete(c.Header, "idle_time_limit") // caption stamps below are in uncapped view time
	var caps []Caption
	view := 0.0
	for _, e := range c.Events {
		view += e.Gap
	}
	for k := 0; k < 1000; k++ {
		caps = append(caps, Caption{view * float64(k) / 1000, "a caption of six short words"})
	}
	start := time.Now()
	out, _, err := Cut(c, caps, defaultTiming)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("200k events, 1000 captions took %v (budget 3s)", d)
	}
	checkCut(t, c, caps, defaultTiming, out)
}

func FuzzCut(f *testing.F) {
	f.Add([]byte(`{"version": 3, "idle_time_limit": 2}`+"\n[0.5, \"o\", \"a\"]\n[9, \"o\", \"b\"]\n[0, \"o\", \"c\"]\n"),
		[]byte("~0:01  one two three\n~0:01  dup\n~0:02.5  at the end\n"))
	f.Add([]byte(`{"version": 3}`+"\n[0, \"o\", \"a\"]\n"), []byte(""))
	f.Add([]byte(`{"version": 3}`+"\n[1e308, \"o\", \"a\"]\n[1e308, \"o\", \"b\"]\n"), []byte("~0:00  x\n"))
	f.Fuzz(func(t *testing.T, castData, capsData []byte) {
		c, err := parseCast("f.cast", castData)
		if err != nil {
			return
		}
		caps, err := parseCaptions("f.txt", capsData)
		if err != nil {
			return
		}
		out, _, err := Cut(c, caps, defaultTiming)
		if err != nil {
			return
		}
		checkCut(t, c, caps, defaultTiming, out)
	})
}
