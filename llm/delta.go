package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// DeltaKind says what a stream event carries.
type DeltaKind int

const (
	// DeltaText carries answer text in Text.
	DeltaText DeltaKind = iota + 1
	// DeltaToolCallStart opens a call: Index, Call.ID and Call.Name.
	DeltaToolCallStart
	// DeltaToolCallInput carries a fragment of the call's input JSON in Text.
	DeltaToolCallInput
	// DeltaToolCallEnd closes a call: Call is complete and its Input is a valid
	// JSON object, byte-equal to the matching Response.ToolCalls entry.
	DeltaToolCallEnd
)

// Delta is one stream event, in wire order.
//
// Text and tool-call events interleave as the provider sends them; Index (the
// content-block index) ties a call's Start, Input and End together. Every Delta,
// and everything it points to, is freshly allocated for its event and never
// touched again: the receiver owns it and may keep it or hand it to another
// goroutine without copying.
type Delta struct {
	Kind  DeltaKind
	Index int
	Text  string
	Call  *ToolCall // Start and End only
}

// TextOnly adapts a text callback to Stream: answer text is forwarded, every
// other event dropped. A nil fn yields a nil callback.
func TextOnly(fn func(string)) func(Delta) {
	if fn == nil {
		return nil
	}
	return func(d Delta) {
		if d.Kind == DeltaText {
			fn(d.Text)
		}
	}
}

// deltaState turns SDK stream events into Deltas. It is PURE (no IO) and
// per-call, which is what keeps concurrent streams from crossing.
//
// Each open tool call's input is appended to one builder and parsed once, at its
// content_block_stop, so a call costs O(fragments + bytes), never a rescan per
// fragment.
type deltaState struct {
	open map[int]*openCall
	// stopped marks every content block whose content_block_stop arrived, so a
	// partial Response can keep exactly the complete blocks.
	stopped map[int]bool
	// done holds the calls that reached End, by Index, so a cancelled or failed
	// stream reports exactly those and drops a half-streamed one.
	done []ToolCall
	// validations counts JSON parses, so a test can hold the parse-once bound.
	validations int
}

type openCall struct {
	id, name string
	input    strings.Builder
}

// next maps one event to zero or more Deltas. An error means a call ended with
// input that is not a JSON object; the stream is then malformed.
func (s *deltaState) next(event anthropic.MessageStreamEventUnion) ([]Delta, error) {
	idx := int(event.Index)
	switch event.Type {
	case "content_block_start":
		if event.ContentBlock.Type != "tool_use" {
			return nil, nil
		}
		if s.open == nil {
			s.open = map[int]*openCall{}
		}
		s.open[idx] = &openCall{id: event.ContentBlock.ID, name: event.ContentBlock.Name}
		return []Delta{{Kind: DeltaToolCallStart, Index: idx,
			Call: &ToolCall{ID: event.ContentBlock.ID, Name: event.ContentBlock.Name}}}, nil
	case "content_block_delta":
		switch event.Delta.Type {
		case "text_delta":
			// ANSWER text only. Thinking deltas are accumulated into Blocks but
			// never forwarded: a learner watching a question appear must not see
			// the model's reasoning scroll past.
			return []Delta{{Kind: DeltaText, Index: idx, Text: event.Delta.Text}}, nil
		case "input_json_delta":
			c := s.open[idx]
			if c == nil || event.Delta.PartialJSON == "" {
				return nil, nil
			}
			c.input.WriteString(event.Delta.PartialJSON)
			return []Delta{{Kind: DeltaToolCallInput, Index: idx, Text: event.Delta.PartialJSON}}, nil
		}
	case "content_block_stop":
		c := s.open[idx]
		if c == nil {
			s.markStopped(idx)
			return nil, nil
		}
		delete(s.open, idx)
		// The SDK's accumulator uses the same rule: the start block's "{}" stands
		// until a fragment replaces it.
		in := c.input.String()
		if in == "" {
			in = "{}"
		}
		s.validations++
		if err := toolInputObject(c.name, []byte(in)); err != nil {
			return nil, err
		}
		s.markStopped(idx)
		call := ToolCall{ID: c.id, Name: c.name, Input: json.RawMessage(in)}
		s.done = append(s.done, call)
		end := call // a separate copy for the receiver; done stays ours
		end.Input = append(json.RawMessage(nil), call.Input...)
		return []Delta{{Kind: DeltaToolCallEnd, Index: idx, Call: &end}}, nil
	}
	return nil, nil
}

func (s *deltaState) markStopped(idx int) {
	if s.stopped == nil {
		s.stopped = map[int]bool{}
	}
	s.stopped[idx] = true
}

// complete reports whether block idx may stay in a partial Response: it reached
// its stop, or it is answer text (a text prefix is the documented salvage). A
// half-streamed tool_use, or a thinking block cut off before its signature,
// would be half-reported, and Message() would echo it back.
func (s *deltaState) complete(idx int, typ string) bool {
	return s.stopped[idx] || typ == "text"
}

// toolInputObject is THE check that a tool call's input is a JSON object, shared
// by the stream's End and Complete's response so the two paths cannot disagree.
func toolInputObject(name string, in []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(in, &obj); err != nil || obj == nil {
		return fmt.Errorf("%w: tool call %q input is not a JSON object: %s", ErrMalformed, name, in)
	}
	return nil
}
