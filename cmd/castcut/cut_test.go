package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

// castOf builds a v3 cast of single-character output events with these gaps.
func castOf(limit float64, gaps ...float64) Cast {
	h := map[string]json.RawMessage{"version": json.RawMessage("3")}
	if limit > 0 {
		h["idle_time_limit"] = json.RawMessage(fmt.Sprint(limit))
	}
	c := Cast{Header: h}
	for i, g := range gaps {
		c.Events = append(c.Events, Event{Gap: g, Kind: "o", Data: jsonString(fmt.Sprint(i))})
	}
	return c
}

func headerCaptions(t *testing.T, c Cast) []headerCaption {
	t.Helper()
	var hc []headerCaption
	if err := json.Unmarshal(c.Header["captions"], &hc); err != nil {
		t.Fatalf("header captions: %v (%s)", err, c.Header["captions"])
	}
	return hc
}

// outTimes is each output event's absolute output time.
func outTimes(c Cast) []float64 {
	ts := make([]float64, len(c.Events))
	at := 0.0
	for i, e := range c.Events {
		at += e.Gap
		ts[i] = at
	}
	return ts
}

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestCutWithoutCaptionsSqueezesEveryGap(t *testing.T) {
	out, sum, err := Cut(castOf(0, 0.5, 10, 0.2), nil, defaultTiming)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Header["captions"]) != "[]" {
		t.Errorf("captions = %s, want []", out.Header["captions"])
	}
	// Each gap is squeezed to at most idle (1s), then divided by speed (5).
	want := []float64{0.1, 0.2, 0.04}
	for i, e := range out.Events {
		if !near(e.Gap, want[i], 1e-9) {
			t.Errorf("gap %d = %v, want %v", i, e.Gap, want[i])
		}
	}
	if !near(sum.Total, 0.34, 1e-9) || sum.View != 10.7 {
		t.Errorf("summary = %+v", sum)
	}
}

func TestCutIdleLimitIsConsumed(t *testing.T) {
	out, sum, err := Cut(castOf(2, 1, 30), nil, defaultTiming)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Header["idle_time_limit"]; ok {
		t.Error("idle_time_limit survived the cut")
	}
	if sum.View != 3 {
		t.Errorf("view = %v, want 3 (30s gap capped at 2)", sum.View)
	}
}

func TestCutCaptionPlaysInRealTimeWithMarkerFirst(t *testing.T) {
	// Events at view 1..10. A caption at 4 with lead 1 starts its window at 3,
	// exactly on event "2": the marker must precede it. Speed 1 keeps every
	// rate exactly 1, so the two land on the same output instant with no
	// rounding between them.
	gaps := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	tm := defaultTiming
	tm.Speed = 1
	out, _, err := Cut(castOf(0, gaps...), []Caption{{4, "one two"}}, tm)
	if err != nil {
		t.Fatal(err)
	}
	hc := headerCaptions(t, out)
	if len(hc) != 1 || !near(hc[0].End-hc[0].Start, 4, 1e-3) {
		t.Fatalf("captions = %+v, want one 4s window (min-hold)", hc)
	}
	ts := outTimes(out)
	for i, e := range out.Events {
		if e.Kind == "m" {
			if !near(ts[i], hc[0].Start, 1e-6) || string(e.Data) != `"one two"` {
				t.Errorf("marker at %v (%s), want %v", ts[i], e.Data, hc[0].Start)
			}
			if string(out.Events[i+1].Data) != `"2"` || out.Events[i+1].Gap != 0 {
				t.Errorf("marker is not immediately before event 2 at the same instant: %+v", out.Events[i+1])
			}
			return
		}
		if string(e.Data) == `"2"` {
			t.Fatal("event 2 came before the marker")
		}
	}
	t.Fatal("no marker")
}

func TestCutOverlappingCaptionStartsWhenPreviousEnds(t *testing.T) {
	out, _, err := Cut(castOf(0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1), []Caption{{2, "a"}, {3, "b"}}, defaultTiming)
	if err != nil {
		t.Fatal(err)
	}
	hc := headerCaptions(t, out)
	if !near(hc[1].Start, hc[0].End, 1e-3) {
		t.Errorf("second caption starts at %v, want first's end %v", hc[1].Start, hc[0].End)
	}
}

func TestCutHoldsTheLastFrameForALateCaption(t *testing.T) {
	// Take ends at view 5; a caption at 4 needs real time until 3+4 = 7.
	out, sum, err := Cut(castOf(0, 1, 1, 1, 1, 1), []Caption{{4, "late"}}, defaultTiming)
	if err != nil {
		t.Fatal(err)
	}
	last := out.Events[len(out.Events)-1]
	if last.Kind != "o" || string(last.Data) != `""` {
		t.Fatalf("last event = %+v, want the empty hold", last)
	}
	hc := headerCaptions(t, out)
	if ts := outTimes(out); !near(ts[len(ts)-1], hc[0].End, 1e-3) || !near(sum.Total, ts[len(ts)-1], 1e-6) {
		t.Errorf("take ends at %v, caption at %v, summary %v", ts[len(ts)-1], hc[0].End, sum.Total)
	}
	if sum.View != 7 {
		t.Errorf("view = %v, want 7", sum.View)
	}
}

func TestCutRejects(t *testing.T) {
	_, _, err := Cut(castOf(0, 1, 1), []Caption{{2.5, "too late"}}, defaultTiming)
	if err == nil || !strings.Contains(err.Error(), "caption at 2.5s is past the end of the recording (2.0s): too late") {
		t.Errorf("past end: err = %v", err)
	}
	for _, gaps := range [][]float64{{math.NaN()}, {1, -1}, {math.Inf(1)}, {}} {
		if _, _, err := Cut(castOf(0, gaps...), nil, defaultTiming); err == nil {
			t.Errorf("gaps %v accepted", gaps)
		}
	}
	for _, at := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, _, err := Cut(castOf(0, 1, 1), []Caption{{at, "bad"}}, defaultTiming); err == nil || !strings.Contains(err.Error(), "not a time") {
			t.Errorf("caption at %v: err = %v", at, err)
		}
	}
	if _, _, err := Cut(castOf(0, 4e5, 4e5), nil, defaultTiming); err == nil || !strings.Contains(err.Error(), "past 168 hours") {
		t.Errorf("overflowing take: err = %v", err)
	}
	for _, tm := range []Timing{
		{Speed: 0, WPS: 1}, {Speed: 1, WPS: -1}, {Speed: 1, WPS: 1, Idle: -1},
		{Speed: math.NaN(), WPS: 1}, {Speed: 1, WPS: 1, MinHold: 1e308}, {Speed: 1, WPS: 1, Lead: math.Inf(1)},
	} {
		if _, _, err := Cut(castOf(0, 1), nil, tm); err == nil {
			t.Errorf("%+v accepted", tm)
		}
	}
}

func TestCutBoundsIdleLimit(t *testing.T) {
	// A negative limit was "no limit" in the prototype; castcut refuses it.
	for _, limit := range []string{"1e9", "-1"} {
		c := castOf(0, 1)
		c.Header["idle_time_limit"] = json.RawMessage(limit)
		if _, _, err := Cut(c, nil, defaultTiming); err == nil || !strings.Contains(err.Error(), "idle_time_limit "+limit+" is not in [0,") {
			t.Errorf("limit %s: err = %v", limit, err)
		}
	}
}

func TestCutDoesNotMutateItsInput(t *testing.T) {
	in := castOf(2, 1, 1)
	if _, _, err := Cut(in, []Caption{{1, "x"}}, defaultTiming); err != nil {
		t.Fatal(err)
	}
	if _, ok := in.Header["idle_time_limit"]; !ok || in.Header["captions"] != nil || in.Events[0].Gap != 1 {
		t.Errorf("input changed: %+v", in)
	}
}
