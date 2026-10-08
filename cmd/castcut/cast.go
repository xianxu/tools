package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Cast is an asciicast v3 recording: a header object and its event stream.
//
// The header is kept as raw fields so anything castcut does not interpret
// (term, env, timestamp, title…) passes through a cut untouched.
type Cast struct {
	Header map[string]json.RawMessage
	Events []Event
}

// Event is one `[interval, kind, data]` line. Gap is the interval since the
// previous event, as recorded; Data is passed through verbatim.
type Event struct {
	Gap  float64
	Kind string
	Data json.RawMessage
}

// maxSeconds bounds every duration castcut accepts — a take's length and each
// timing flag — so the timing arithmetic stays finite. A week is far beyond any
// demo and far inside float64.
const maxSeconds = 7 * 24 * 3600

// inRange is the one range check for every duration castcut reads. Written as
// a positive comparison so NaN fails it.
func inRange(x float64) bool { return x >= 0 && x <= maxSeconds }

// splitLines splits on \n, \r\n and a lone \r, the line endings an editor or
// another tool may leave behind.
func splitLines(data []byte) []string {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(s, "\r", "\n"), "\n")
}

func parseCast(path string, data []byte) (Cast, error) {
	var c Cast
	n, total := 0, 0.0
	for i, line := range splitLines(data) {
		if strings.TrimSpace(line) == "" {
			continue
		}
		at := fmt.Sprintf("%s:%d", path, i+1)
		if n++; n == 1 {
			if err := json.Unmarshal([]byte(line), &c.Header); err != nil || c.Header == nil {
				return Cast{}, fmt.Errorf("%s: header is not a JSON object", at)
			}
			var v float64
			if json.Unmarshal(c.Header["version"], &v) != nil || v != 3 {
				return Cast{}, fmt.Errorf("%s: expected asciicast v3", path)
			}
			continue
		}
		var raw []json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil || len(raw) != 3 {
			return Cast{}, fmt.Errorf("%s: expected an event [interval, kind, data]", at)
		}
		var e Event
		if json.Unmarshal(raw[0], &e.Gap) != nil || !inRange(e.Gap) {
			return Cast{}, fmt.Errorf("%s: event interval is not a number of seconds in [0, %d]", at, maxSeconds)
		}
		if total += e.Gap; !inRange(total) {
			return Cast{}, fmt.Errorf("%s: recording runs past %d hours; not a take", at, maxSeconds/3600)
		}
		if json.Unmarshal(raw[1], &e.Kind) != nil {
			return Cast{}, fmt.Errorf("%s: event kind is not a string", at)
		}
		e.Data = raw[2]
		c.Events = append(c.Events, e)
	}
	if n == 0 {
		return Cast{}, fmt.Errorf("%s: empty file", path)
	}
	if len(c.Events) == 0 {
		return Cast{}, fmt.Errorf("%s: no events", path)
	}
	return c, nil
}

// idleLimit pops the header's idle_time_limit. A missing, null or zero limit
// means none.
func (c *Cast) idleLimit() (float64, error) {
	raw, ok := c.Header["idle_time_limit"]
	if !ok {
		return 0, nil
	}
	delete(c.Header, "idle_time_limit")
	var v *float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, fmt.Errorf("idle_time_limit is not a number: %s", raw)
	}
	if v == nil || *v == 0 {
		return 0, nil
	}
	if !inRange(*v) {
		return 0, fmt.Errorf("idle_time_limit %v is past %d hours", *v, maxSeconds/3600)
	}
	return *v, nil
}

// encodeCast writes the header on one line, then one `[interval, "kind", data]`
// line per event, in asciinema's own spacing.
func encodeCast(c Cast) ([]byte, error) {
	var b bytes.Buffer
	header, err := marshalNoEscape(c.Header)
	if err != nil {
		return nil, err
	}
	b.Write(header)
	b.WriteByte('\n')
	for _, e := range c.Events {
		kind, _ := json.Marshal(e.Kind)
		b.WriteByte('[')
		b.WriteString(strconv.FormatFloat(e.Gap, 'f', -1, 64))
		b.WriteString(", ")
		b.Write(kind)
		b.WriteString(", ")
		b.Write(e.Data)
		b.WriteString("]\n")
	}
	return b.Bytes(), nil
}

// jsonString encodes s without HTML escaping, as asciinema does.
func jsonString(s string) json.RawMessage {
	b, _ := marshalNoEscape(s)
	return b
}
