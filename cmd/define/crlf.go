package main

import "io"

// crlfWriter translates a bare "\n" into "\r\n" for output written while the
// terminal is in RAW mode.
//
// #14's rule was "render cooked, play raw", and that cannot extend to a stream:
// deltas arrive continuously and flapping the terminal per delta is not a thing.
// So the ask path stays raw and translates here — which is also what keeps the
// key reader seeing bytes, so Ctrl-C can mean something narrower than quitting.
//
// lastWasCR carries across Write calls because deltas arrive chunked: a reply
// split as "one\r" then "\ntwo" must not become "one\r\r\ntwo".
type crlfWriter struct {
	w         io.Writer
	lastWasCR bool
}

func (c *crlfWriter) Write(p []byte) (int, error) {
	out := make([]byte, 0, len(p)+8)
	for _, b := range p {
		if b == '\n' && !c.lastWasCR {
			out = append(out, '\r')
		}
		out = append(out, b)
		c.lastWasCR = b == '\r'
	}
	if _, err := c.w.Write(out); err != nil {
		return 0, err
	}
	// n is what the CALLER handed over, not what reached the terminal: an
	// io.Writer that reports more bytes than it was given breaks io.Copy and
	// every wrapper that checks n against len(p).
	return len(p), nil
}
