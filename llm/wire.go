package llm

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

// The one place the neutral Request meets the Anthropic shape (ARCH-DRY):
// params() calls these and nothing else does. PURE, so every translation rule is
// a table test on marshalled params with no server.

// toolChoiceKeywords share ToolChoice's string with tool names, which is why a
// tool may not be named after one.
var toolChoiceKeywords = map[string]bool{"auto": true, "any": true, "none": true}

// checkRequest refuses what cannot be sent, before anything is.
func checkRequest(r Request) error {
	if len(r.Messages) == 0 && r.Prompt == "" {
		return fmt.Errorf("%w: request has neither Messages nor Prompt", ErrRequest)
	}
	names := map[string]bool{}
	for _, t := range r.Tools {
		if t.Name == "" {
			return fmt.Errorf("%w: a tool has no name", ErrRequest)
		}
		if toolChoiceKeywords[t.Name] {
			return fmt.Errorf("%w: tool %q collides with the ToolChoice keyword of that name", ErrRequest, t.Name)
		}
		if names[t.Name] {
			return fmt.Errorf("%w: tool %q is declared twice", ErrRequest, t.Name)
		}
		names[t.Name] = true
	}
	if c := r.ToolChoice; c != "" && !toolChoiceKeywords[c] && !names[c] {
		return fmt.Errorf("%w: ToolChoice %q is neither auto/any/none nor a declared tool", ErrRequest, c)
	}
	for i, m := range r.Messages {
		if m.Role != RoleUser && m.Role != RoleAssistant {
			return fmt.Errorf("%w: message %d has role %q", ErrRequest, i, m.Role)
		}
		if len(m.Content) == 0 {
			return fmt.Errorf("%w: message %d has no content", ErrRequest, i)
		}
		for j, b := range m.Content {
			if _, err := blockJSON(b); err != nil {
				return fmt.Errorf("%w: message %d block %d: %v", ErrRequest, i, j, err)
			}
		}
	}
	return nil
}

// wireMessages is Messages then Prompt. Prompt keeps the SDK's own constructor,
// so a single-turn request goes out exactly as it did before Messages existed.
func wireMessages(r Request) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(r.Messages)+1)
	for _, m := range r.Messages {
		out = append(out, param.Override[anthropic.MessageParam](messageJSON(m)))
	}
	if r.Prompt != "" {
		out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(r.Prompt)))
	}
	return out
}

// messageJSON renders one turn. checkRequest has already proven every block
// renders, so an error here is unreachable and the block is dropped rather than
// panicking a send path.
func messageJSON(m Message) json.RawMessage {
	blocks := make([]json.RawMessage, 0, len(m.Content))
	for _, b := range m.Content {
		if raw, err := blockJSON(b); err == nil {
			blocks = append(blocks, raw)
		}
	}
	out, _ := json.Marshal(struct {
		Role    Role              `json:"role"`
		Content []json.RawMessage `json:"content"`
	}{m.Role, blocks})
	return out
}

// blockJSON is Raw when present (byte for byte: a thinking block's signature is
// checked upstream), else the typed fields.
func blockJSON(b Block) (json.RawMessage, error) {
	if len(b.Raw) > 0 {
		if !json.Valid(b.Raw) {
			return nil, fmt.Errorf("block Raw is not valid JSON")
		}
		return b.Raw, nil
	}
	var v any
	switch b.Type {
	case "text":
		v = map[string]any{"type": "text", "text": b.Text}
	case "tool_result":
		if b.CallID == "" {
			return nil, fmt.Errorf("tool_result has no CallID")
		}
		m := map[string]any{"type": "tool_result", "tool_use_id": b.CallID, "content": b.Text}
		if b.IsError {
			m["is_error"] = true
		}
		v = m
	case "tool_use":
		if b.ToolCall == nil {
			return nil, fmt.Errorf("tool_use block has no ToolCall")
		}
		in := b.ToolCall.Input
		if len(in) == 0 {
			in = json.RawMessage("{}")
		}
		v = map[string]any{"type": "tool_use", "id": b.ToolCall.ID, "name": b.ToolCall.Name, "input": in}
	case "thinking":
		v = map[string]any{"type": "thinking", "thinking": b.Text, "signature": b.Signature}
	default:
		return nil, fmt.Errorf("block type %q has no Raw and no typed form", b.Type)
	}
	return json.Marshal(v)
}

// wireTools renders tool definitions; nil when there are none, so a request
// without tools is unchanged on the wire.
func wireTools(ts []Tool) []anthropic.ToolUnionParam {
	if len(ts) == 0 {
		return nil
	}
	out := make([]anthropic.ToolUnionParam, 0, len(ts))
	for _, t := range ts {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		def := map[string]any{"name": t.Name, "input_schema": schema}
		if t.Description != "" {
			def["description"] = t.Description
		}
		tp := param.Override[anthropic.ToolParam](def)
		out = append(out, anthropic.ToolUnionParam{OfTool: &tp})
	}
	return out
}

// wireToolChoice is the zero union for "", which omits the field.
func wireToolChoice(c string) anthropic.ToolChoiceUnionParam {
	switch c {
	case "":
		return anthropic.ToolChoiceUnionParam{}
	case "auto":
		return anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}
	case "any":
		return anthropic.ToolChoiceUnionParam{OfAny: &anthropic.ToolChoiceAnyParam{}}
	case "none":
		return anthropic.ToolChoiceUnionParam{OfNone: &anthropic.ToolChoiceNoneParam{}}
	default:
		return anthropic.ToolChoiceParamOfTool(c)
	}
}
