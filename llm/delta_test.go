package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
)

func event(t testing.TB, js string) anthropic.MessageStreamEventUnion {
	t.Helper()
	var e anthropic.MessageStreamEventUnion
	if err := json.Unmarshal([]byte(js), &e); err != nil {
		t.Fatalf("event %s: %v", js, err)
	}
	return e
}

const messageStart = `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5","content":[],"stop_reason":null,"usage":{"input_tokens":1,"output_tokens":0}}}`

func blockStart(i int, block string) string {
	return fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":%s}`, i, block)
}
func textDelta(i int, s string) string {
	b, _ := json.Marshal(s)
	return fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, i, b)
}
func inputDelta(i int, s string) string {
	b, _ := json.Marshal(s)
	return fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%s}}`, i, b)
}
func blockStop(i int) string { return fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, i) }

const messageEnd = `{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":9}}`
const messageStop = `{"type":"message_stop"}`

// FuzzDeltaAccumulator splits two tool calls' input at arbitrary points and
// interleaves their fragments with a text block's, then holds the stream
// guarantees: per Index, the Input fragments concatenate to End.Input; each Start
// has exactly one End; text concatenates to the answer; End equals what the
// SDK's own accumulator puts in Response.ToolCalls; nothing panics.
func FuzzDeltaAccumulator(f *testing.F) {
	f.Add(`{"city":"Oslo"}`, `{"q":"résumé — ok","n":[1,2,3]}`, "Let me check.", []byte{3, 1, 4, 1, 5, 9, 2, 6})
	f.Add(`{}`, `{"a":{"b":{"c":"d"}}}`, "", []byte{0})
	f.Add(`{"x":""}`, `{"emoji":"🌧"}`, "two calls", []byte{255, 254, 7})
	f.Fuzz(func(t *testing.T, in1, in2, text string, plan []byte) {
		if !utf8.ValidString(text) || !utf8.ValidString(in1) || !utf8.ValidString(in2) {
			t.Skip() // conformance:inapplicable — wire text is a JSON string, always valid UTF-8
		}
		for _, in := range []string{in1, in2} {
			var obj map[string]any
			if json.Unmarshal([]byte(in), &obj) != nil {
				t.Skip() // conformance:inapplicable — only valid inputs are a well-formed stream
			}
		}
		pick := func(i int) int {
			if len(plan) == 0 {
				return 1
			}
			return int(plan[i%len(plan)])
		}
		// Fragments of each input, cut at plan-chosen RUNE offsets: each
		// partial_json is itself a JSON string on the wire, so a fragment is always
		// valid UTF-8 even when it splits a JSON token.
		split := func(s string, seed int) []string {
			var out []string
			r := []rune(s)
			for i := 0; len(r) > 0; i++ {
				n := min(1+pick(seed+i)%7, len(r))
				out = append(out, string(r[:n]))
				r = r[n:]
			}
			return out
		}
		frames := [][]string{nil, nil, nil}
		for _, r := range split(text, 0) {
			frames[0] = append(frames[0], textDelta(0, r))
		}
		for _, p := range split(in1, 11) {
			frames[1] = append(frames[1], inputDelta(1, p))
		}
		for _, p := range split(in2, 23) {
			frames[2] = append(frames[2], inputDelta(2, p))
		}
		for i := range frames {
			frames[i] = append(frames[i], blockStop(i))
		}
		// Starts in index order (the protocol's rule); deltas and stops of open
		// blocks interleave in a plan-chosen order.
		seq := []string{messageStart,
			blockStart(0, `{"type":"text","text":""}`),
			blockStart(1, `{"type":"tool_use","id":"tu_1","name":"weather","input":{}}`),
			blockStart(2, `{"type":"tool_use","id":"tu_2","name":"search","input":{}}`)}
		for step := 0; len(frames[0])+len(frames[1])+len(frames[2]) > 0; step++ {
			i := pick(step+37) % 3
			for len(frames[i]) == 0 {
				i = (i + 1) % 3
			}
			seq = append(seq, frames[i][0])
			frames[i] = frames[i][1:]
		}
		seq = append(seq, messageEnd, messageStop)

		var st deltaState
		var msg anthropic.Message
		var got []Delta
		for _, js := range seq {
			e := event(t, js)
			if err := msg.Accumulate(e); err != nil {
				t.Fatalf("SDK accumulate %s: %v", js, err)
			}
			out, err := st.next(e)
			if err != nil {
				t.Fatalf("next: %v", err)
			}
			got = append(got, out...)
		}

		var answer strings.Builder
		inputs := map[int]*strings.Builder{}
		starts, ends := map[int]int{}, map[int]*ToolCall{}
		for _, d := range got {
			switch d.Kind {
			case DeltaText:
				answer.WriteString(d.Text)
			case DeltaToolCallStart:
				starts[d.Index]++
				inputs[d.Index] = &strings.Builder{}
			case DeltaToolCallInput:
				if inputs[d.Index] == nil {
					t.Fatalf("input for Index %d before its Start", d.Index)
				}
				inputs[d.Index].WriteString(d.Text)
			case DeltaToolCallEnd:
				if ends[d.Index] != nil {
					t.Fatalf("second End for Index %d", d.Index)
				}
				ends[d.Index] = d.Call
			}
		}
		if answer.String() != text {
			t.Errorf("text = %q, want %q", answer.String(), text)
		}
		for i, want := range map[int]string{1: in1, 2: in2} {
			if starts[i] != 1 || ends[i] == nil {
				t.Fatalf("Index %d: %d starts, end %v", i, starts[i], ends[i])
			}
			if inputs[i].String() != want || string(ends[i].Input) != want {
				t.Errorf("Index %d: fragments %q, End %q, want %q", i, inputs[i].String(), ends[i].Input, want)
			}
		}
		resp, err := buildResponse(&msg, time.Second)
		if err != nil {
			t.Fatalf("buildResponse: %v", err)
		}
		want := []ToolCall{*ends[1], *ends[2]}
		if !reflect.DeepEqual(resp.ToolCalls, want) {
			t.Errorf("Response.ToolCalls = %+v, End calls = %+v", resp.ToolCalls, want)
		}
	})
}

// A call's input is appended once per fragment and parsed once at its End, so a
// call split twice as finely costs twice the fragments and still ONE parse — the
// bound a rescan-per-fragment accumulator would break.
func TestDeltaForLinear(t *testing.T) {
	for _, n := range []int{8, 16} {
		input := `{"k":"` + strings.Repeat("v", n*4) + `"}`
		var st deltaState
		st.next(event(t, blockStart(0, `{"type":"tool_use","id":"a","name":"x","input":{}}`)))
		size := (len(input) + n - 1) / n
		fragments := 0
		for s := input; len(s) > 0; fragments++ {
			k := min(size, len(s))
			if _, err := st.next(event(t, inputDelta(0, s[:k]))); err != nil {
				t.Fatal(err)
			}
			s = s[k:]
		}
		out, err := st.next(event(t, blockStop(0)))
		if err != nil || len(out) != 1 || string(out[0].Call.Input) != input {
			t.Fatalf("n=%d: End %+v err %v", n, out, err)
		}
		if st.validations != 1 {
			t.Errorf("n=%d: %d JSON parses for one call of %d fragments, want 1", n, st.validations, fragments)
		}
	}
}

func TestDeltaEndRefusesInvalidInput(t *testing.T) {
	var st deltaState
	st.next(event(t, blockStart(0, `{"type":"tool_use","id":"a","name":"x","input":{}}`)))
	st.next(event(t, inputDelta(0, `{"cut":`)))
	if _, err := st.next(event(t, blockStop(0))); !errors.Is(err, ErrMalformed) {
		t.Errorf("End of truncated input = %v, want ErrMalformed", err)
	}
	if len(st.done) != 0 {
		t.Errorf("a malformed call was recorded as done: %+v", st.done)
	}
}

func TestDeltaThinkingIsNeverForwarded(t *testing.T) {
	var st deltaState
	for _, js := range []string{
		blockStart(0, `{"type":"thinking","thinking":"","signature":""}`),
		`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"secret"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig"}}`,
		blockStop(0),
	} {
		if out, _ := st.next(event(t, js)); len(out) != 0 {
			t.Errorf("%s forwarded %+v", js, out)
		}
	}
}

// Every delivered value is the receiver's: mutating one after delivery must not
// reach the transport's own record, and later events must not rewrite it.
func TestDeltasOwnedByReceiver(t *testing.T) {
	var st deltaState
	var kept []Delta
	var copies []string
	for _, js := range []string{
		blockStart(0, `{"type":"tool_use","id":"a","name":"x","input":{}}`),
		inputDelta(0, `{"k":`), inputDelta(0, `"v"}`), blockStop(0),
		blockStart(1, `{"type":"tool_use","id":"b","name":"y","input":{}}`),
		inputDelta(1, `{"z":1}`), blockStop(1),
	} {
		out, err := st.next(event(t, js))
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range out {
			kept = append(kept, d)
			b, _ := json.Marshal(d)
			copies = append(copies, string(b))
		}
	}
	for i, d := range kept {
		if b, _ := json.Marshal(d); string(b) != copies[i] {
			t.Errorf("delta %d changed after delivery: %s, was %s", i, b, copies[i])
		}
	}
	end := kept[3]
	end.Call.Input[2] = 'X'
	if string(st.done[0].Input) != `{"k":"v"}` {
		t.Errorf("receiver's mutation reached the transport's record: %s", st.done[0].Input)
	}
}

func TestTextOnly(t *testing.T) {
	var got []string
	fn := TextOnly(func(s string) { got = append(got, s) })
	for _, d := range []Delta{
		{Kind: DeltaText, Text: "a"}, {Kind: DeltaToolCallStart, Call: &ToolCall{}},
		{Kind: DeltaToolCallInput, Text: "{"}, {Kind: DeltaToolCallEnd, Call: &ToolCall{}},
		{Kind: DeltaText, Text: "b"},
	} {
		fn(d)
	}
	if strings.Join(got, "") != "ab" {
		t.Errorf("TextOnly forwarded %q, want only the text deltas", got)
	}
	if TextOnly(nil) != nil {
		t.Error("TextOnly(nil) must stay nil so Stream skips the callback")
	}
}

// The partial-response rule for EVERY block type: only blocks that reached their
// stop survive, plus answer text. A thinking block cut before its signature is
// dropped like a half-streamed tool_use.
func TestPartialKeepsOnlyCompleteBlocks(t *testing.T) {
	var st deltaState
	for _, js := range []string{
		blockStart(0, `{"type":"thinking","thinking":"","signature":""}`), blockStop(0),
		blockStart(1, `{"type":"thinking","thinking":"","signature":""}`),   // cut: no stop
		blockStart(2, `{"type":"text","text":""}`),                          // open text
		blockStart(3, `{"type":"tool_use","id":"a","name":"x","input":{}}`), // cut: no stop
	} {
		if _, err := st.next(event(t, js)); err != nil {
			t.Fatal(err)
		}
	}
	for idx, want := range map[int]bool{0: true, 1: false, 2: true, 3: false} {
		typ := map[int]string{0: "thinking", 1: "thinking", 2: "text", 3: "tool_use"}[idx]
		if got := st.complete(idx, typ); got != want {
			t.Errorf("block %d (%s): complete = %v, want %v", idx, typ, got, want)
		}
	}
}
