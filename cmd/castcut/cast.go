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

// splitLines splits on \n, \r\n and a lone \r, the line endings an editor or
// another tool may leave behind.
func splitLines(data []byte) []string {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(s, "\r", "\n"), "\n")
}

func parseCast(path string, data []byte) (Cast, error) {
	var c Cast
	n := 0
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
		if json.Unmarshal(raw[0], &e.Gap) != nil || e.Gap < 0 {
			return Cast{}, fmt.Errorf("%s: event interval is not a non-negative number", at)
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
	if v == nil || *v <= 0 {
		return 0, nil
	}
	return *v, nil
}

// encodeCast writes the header on one line, then one `[interval, "kind", data]`
// line per event, in asciinema's own spacing.
func encodeCast(c Cast) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c.Header); err != nil {
		return nil, err
	}
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
