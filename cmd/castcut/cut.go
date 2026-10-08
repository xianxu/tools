package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Timing is the cut's pacing model. Between captions the take is squeezed to
// at most Idle seconds per gap and then played Speed times faster; around each
// caption it drops to real time for long enough to read it.
type Timing struct {
	Speed   float64 // playback speed between captions
	Idle    float64 // max idle seconds per gap before speeding up
	Lead    float64 // start real time this long before a stamp
	MinHold float64 // minimum real-time seconds per caption
	WPS     float64 // caption reading speed, words per second
	Beat    float64 // extra seconds added to reading time
}

var defaultTiming = Timing{Speed: 5, Idle: 1, Lead: 1, MinHold: 4, WPS: 3.5, Beat: 1}

// validate bounds every flag, so window ends and rates stay finite.
func (t Timing) validate() error {
	if !(inRange(t.Speed) && t.Speed > 0) || !(inRange(t.WPS) && t.WPS > 0) {
		return fmt.Errorf("--speed and --wps must be positive and at most %d", maxSeconds)
	}
	for _, v := range []float64{t.Idle, t.Lead, t.MinHold, t.Beat} {
		if !inRange(v) {
			return fmt.Errorf("--idle, --lead, --min-hold and --beat must be between 0 and %d seconds", maxSeconds)
		}
	}
	return nil
}

// window is a stretch of view time played in real time while Text shows.
type window struct {
	Start, End float64
	Text       string
}

// segment maps view time [P, Q) to output time at Rate output seconds per view
// second. Segments are contiguous from 0.
type segment struct{ P, Q, Rate float64 }

// planWindows gives each caption a real-time window. A caption that would
// overlap the previous one starts when that one ends.
func planWindows(caps []Caption, t Timing) []window {
	var ws []window
	prevEnd := 0.0
	for _, c := range caps {
		hold := math.Max(t.MinHold, float64(len(strings.Fields(c.Text)))/t.WPS+t.Beat)
		start := math.Max(math.Max(c.At-t.Lead, prevEnd), 0)
		ws = append(ws, window{start, start + hold, c.Text})
		prevEnd = start + hold
	}
	return ws
}

// buildSegments covers the view timeline up to the last of times with rates:
// 1 inside a window; outside, each gap between consecutive times has its idle
// squeezed to Idle seconds and is then sped up by Speed.
//
// times must be non-decreasing and windows sorted and non-overlapping (as
// planWindows makes them), so both are walked once with a cursor.
func buildSegments(times []float64, ws []window, t Timing) []segment {
	var segs []segment
	bounds := make([]float64, 0, 2*len(ws))
	for _, w := range ws {
		bounds = append(bounds, w.Start, w.End)
	}
	type piece struct {
		p, q float64
		slow bool
	}
	bi, wi := 0, 0
	inside := func(x float64) bool {
		for wi < len(ws) && ws[wi].End <= x {
			wi++
		}
		return wi < len(ws) && ws[wi].Start <= x
	}
	prev := 0.0
	var pieces []piece
	for _, tm := range times {
		if tm <= prev {
			continue
		}
		cuts := []float64{prev}
		for bi < len(bounds) && bounds[bi] <= prev {
			bi++
		}
		for ; bi < len(bounds) && bounds[bi] < tm; bi++ {
			if bounds[bi] > cuts[len(cuts)-1] {
				cuts = append(cuts, bounds[bi])
			}
		}
		cuts = append(cuts, tm)
		pieces = pieces[:0]
		fast := 0.0
		for k := 1; k < len(cuts); k++ {
			p, q := cuts[k-1], cuts[k]
			slow := inside((p + q) / 2)
			pieces = append(pieces, piece{p, q, slow})
			if !slow {
				fast += q - p
			}
		}
		squeeze := 1.0
		if fast > 0 {
			squeeze = math.Min(1, t.Idle/fast)
		}
		for _, pc := range pieces {
			rate := squeeze / t.Speed
			if pc.slow {
				rate = 1
			}
			segs = append(segs, segment{pc.p, pc.q, rate})
		}
		prev = tm
	}
	return segs
}

// warper maps view time to output time over contiguous segments.
type warper struct {
	segs []segment
	cum  []float64 // output time at segs[k].P
}

func newWarper(segs []segment) warper {
	cum := make([]float64, len(segs))
	out := 0.0
	for k, s := range segs {
		cum[k] = out
		out += (s.Q - s.P) * s.Rate
	}
	return warper{segs, cum}
}

func (w warper) warp(x float64) float64 {
	// The last segment starting before x; earlier ones count in full.
	k := sort.Search(len(w.segs), func(i int) bool { return w.segs[i].P >= x }) - 1
	if k < 0 {
		return 0
	}
	s := w.segs[k]
	return w.cum[k] + (math.Min(x, s.Q)-s.P)*s.Rate
}

// Summary describes a cut for the operator.
type Summary struct {
	Total, View float64
	Captions    []window // in output time
}

type headerCaption struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func round(x float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(x*p) / p
}

// Cut applies the timing model to a take. The result carries
// `captions: [{start, end, text}]` in output seconds in its header and one `m`
// marker per caption at its start; idle_time_limit is consumed. When the last
// caption's window outlasts the take, an empty output event at the window's
// end holds the final frame on screen until the caption is read.
func Cut(c Cast, caps []Caption, t Timing) (Cast, Summary, error) {
	if err := t.validate(); err != nil {
		return Cast{}, Summary{}, err
	}
	header := make(map[string]json.RawMessage, len(c.Header)+1)
	for k, v := range c.Header {
		header[k] = v
	}
	in := Cast{Header: header, Events: c.Events}
	limit, err := in.idleLimit()
	if err != nil {
		return Cast{}, Summary{}, err
	}

	// parseCast and parseCaptions already enforce these ranges; a Cast or
	// caption built in code goes through the same check here.
	if len(c.Events) == 0 {
		return Cast{}, Summary{}, fmt.Errorf("no events")
	}
	times := make([]float64, len(c.Events))
	tm := 0.0
	for i, e := range c.Events {
		if !inRange(e.Gap) {
			return Cast{}, Summary{}, fmt.Errorf("event %d: interval %v is not in [0, %d]", i, e.Gap, maxSeconds)
		}
		gap := e.Gap
		if limit > 0 {
			gap = math.Min(gap, limit)
		}
		tm += gap
		times[i] = tm
	}
	last := times[len(times)-1]
	if !inRange(last) {
		return Cast{}, Summary{}, fmt.Errorf("recording runs past %d hours; not a take", maxSeconds/3600)
	}
	for _, cp := range caps {
		if !inRange(cp.At) {
			return Cast{}, Summary{}, fmt.Errorf("caption at %v is not a time in the recording: %s", cp.At, cp.Text)
		}
		if cp.At > last {
			return Cast{}, Summary{}, fmt.Errorf("caption at %.1fs is past the end of the recording (%.1fs): %s", cp.At, last, cp.Text)
		}
	}
	ws := planWindows(caps, t)
	view := last
	if len(ws) > 0 && ws[len(ws)-1].End > view {
		view = ws[len(ws)-1].End
	}
	w := newWarper(buildSegments(append(times, view), ws, t))

	type item struct {
		at   float64
		prio int // markers sort before output at the same instant
		e    Event
	}
	stream := make([]item, 0, len(times)+len(ws)+1)
	for i, e := range c.Events {
		stream = append(stream, item{w.warp(times[i]), 0, e})
	}
	if view > last {
		stream = append(stream, item{w.warp(view), 0, Event{Kind: "o", Data: jsonString("")}})
	}
	sum := Summary{View: view, Total: w.warp(view)}
	hc := make([]headerCaption, 0, len(ws))
	for _, win := range ws {
		oc := window{round(w.warp(win.Start), 3), round(w.warp(win.End), 3), win.Text}
		sum.Captions = append(sum.Captions, oc)
		hc = append(hc, headerCaption{oc.Start, oc.End, oc.Text})
		stream = append(stream, item{oc.Start, -1, Event{Kind: "m", Data: jsonString(win.Text)}})
	}
	sort.SliceStable(stream, func(i, j int) bool {
		if stream[i].at != stream[j].at {
			return stream[i].at < stream[j].at
		}
		return stream[i].prio < stream[j].prio
	})
	if header["captions"], err = marshalNoEscape(hc); err != nil {
		return Cast{}, Summary{}, err
	}

	// Intervals are differences of rounded absolute times, so rounding never
	// accumulates along the stream.
	out := Cast{Header: header, Events: make([]Event, len(stream))}
	prev := 0.0
	for i, it := range stream {
		e := it.e
		at := round(it.at, 6)
		e.Gap = round(at-prev, 6)
		out.Events[i] = e
		prev = at
	}
	return out, sum, nil
}

// marshalNoEscape is json.Marshal without HTML escaping: captions are not HTML.
func marshalNoEscape(v any) (json.RawMessage, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
