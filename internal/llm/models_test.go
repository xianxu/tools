package llm

import (
	"errors"
	"reflect"
	"testing"
)

func TestSelectModel(t *testing.T) {
	for _, tc := range []struct {
		name   string
		models []ModelInfo
		want   ModelSelection
	}{
		{"provider first", []ModelInfo{{"gemini-3-flash", "antigravity"}, {"gpt-5.6", "openai"}, {"claude-opus-4-6", "anthropic"}}, ModelSelection{"anthropic", "claude-opus-4-6"}},
		{"owner is authoritative", []ModelInfo{{"claude-opus-5", "antigravity"}, {"gpt-6", "openai"}}, ModelSelection{"openai", "gpt-6"}},
		{"opus numeric", []ModelInfo{{"claude-opus-4-9", "anthropic"}, {"claude-opus-4-10", "anthropic"}}, ModelSelection{"anthropic", "claude-opus-4-10"}},
		{"opus dated", []ModelInfo{{"claude-opus-4-1-20250805", "anthropic"}, {"claude-opus-5", "anthropic"}}, ModelSelection{"anthropic", "claude-opus-5"}},
		{"gpt preference", []ModelInfo{{"gpt-7", "openai"}, {"gpt-6", "openai"}, {"gpt-5.6-codex", "openai"}}, ModelSelection{"openai", "gpt-5.6-codex"}},
		{"gpt six second", []ModelInfo{{"gpt-7", "openai"}, {"gpt-6", "openai"}}, ModelSelection{"openai", "gpt-6"}},
		{"gpt canonical", []ModelInfo{{"gpt-5.6-codex", "openai"}, {"gpt-5.6", "openai"}}, ModelSelection{"openai", "gpt-5.6"}},
		{"gpt tuple", []ModelInfo{{"gpt-5.9.9", "openai"}, {"gpt-5.10.1", "openai"}}, ModelSelection{"openai", "gpt-5.10.1"}},
		{"flash before pro", []ModelInfo{{"gemini-4-pro-high", "antigravity"}, {"gemini-2.5-flash", "antigravity"}}, ModelSelection{"antigravity", "gemini-2.5-flash"}},
		{"gemini newest", []ModelInfo{{"gemini-2.5-flash-high", "antigravity"}, {"gemini-3-flash-low", "antigravity"}}, ModelSelection{"antigravity", "gemini-3-flash-low"}},
		{"gemini high", []ModelInfo{{"gemini-3-flash", "antigravity"}, {"gemini-3-flash-high", "antigravity"}}, ModelSelection{"antigravity", "gemini-3-flash-high"}},
		{"gemini unsuffixed", []ModelInfo{{"gemini-3-pro-low", "antigravity"}, {"gemini-3-pro", "antigravity"}}, ModelSelection{"antigravity", "gemini-3-pro"}},
		{"gemini low", []ModelInfo{{"gemini-3-pro-lite", "antigravity"}, {"gemini-3-pro-low", "antigravity"}}, ModelSelection{"antigravity", "gemini-3-pro-low"}},
		{"skip unsuitable provider", []ModelInfo{{"claude-sonnet-5", "anthropic"}, {"gpt-5.6", "unknown"}, {"gemini-3-pro", "antigravity"}}, ModelSelection{"antigravity", "gemini-3-pro"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]ModelInfo(nil), tc.models...)
			var permutations func(int)
			permutations = func(i int) {
				if i == len(tc.models) {
					got, err := SelectModel(tc.models)
					if err != nil || got != tc.want {
						t.Fatalf("SelectModel(%v) = %+v, %v; want %+v", tc.models, got, err, tc.want)
					}
					return
				}
				for j := i; j < len(tc.models); j++ {
					tc.models[i], tc.models[j] = tc.models[j], tc.models[i]
					permutations(i + 1)
					tc.models[i], tc.models[j] = tc.models[j], tc.models[i]
				}
			}
			permutations(0)
			if !reflect.DeepEqual(original, tc.models) {
				t.Fatal("mutated catalog")
			}
			got, err := SelectModel(append(tc.models, tc.models...))
			if err != nil || got != tc.want {
				t.Fatalf("duplicates: %+v %v", got, err)
			}
		})
	}
}

func TestSelectModelRejectsUnsupported(t *testing.T) {
	for _, model := range []ModelInfo{
		{}, {"gpt-6", "OpenAI"}, {"gpt-6", "codex"}, {"claude-opus-5", "antigravity"},
		{"gpt-5.6-image", "openai"}, {"gpt-5.6-agent", "openai"}, {"gpt-5.6x", "openai"}, {"gpt-5..6", "openai"}, {"gpt-5.6.0.1", "openai"}, {"gpt-99999999999999999999999999999", "openai"},
		{"gpt-05.6", "openai"}, {"gpt-5.6\n", "openai"}, {"sol", "openai"},
		{"claude-opus-5-latest", "anthropic"}, {"claude-opus-5-20251301", "anthropic"},
		{"gemini-3-flash-image", "antigravity"}, {"gemini-3-flash-agent", "antigravity"}, {"gemini-3-flash-audio", "antigravity"}, {"gemini-3-embedding", "antigravity"},
	} {
		t.Run(model.OwnedBy+"/"+model.ID, func(t *testing.T) {
			got, err := SelectModel([]ModelInfo{model})
			if !errors.Is(err, ErrUnavailable) || got != (ModelSelection{}) {
				t.Fatalf("got %+v, %v", got, err)
			}
		})
	}
	if _, err := SelectModel(nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("empty: %v", err)
	}
}

func FuzzSelectModel(f *testing.F) {
	f.Add("gpt-5.6", "openai")
	f.Add("claude-opus-4-6", "anthropic")
	f.Add("gemini-3-flash-high", "antigravity")
	f.Fuzz(func(t *testing.T, id, owner string) {
		input := []ModelInfo{{id, owner}, {"gemini-2.5-pro", "antigravity"}}
		got, err := SelectModel(input)
		if err != nil {
			t.Fatal(err)
		}
		if got != (ModelSelection{owner, id}) && got != (ModelSelection{"antigravity", "gemini-2.5-pro"}) {
			t.Fatalf("invented selection %+v", got)
		}
		reverse, err := SelectModel([]ModelInfo{input[1], input[0], input[0]})
		if err != nil || reverse != got {
			t.Fatalf("order dependent: %+v %+v %v", got, reverse, err)
		}
	})
}
