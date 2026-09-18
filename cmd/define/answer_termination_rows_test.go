package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// Cancel only after a complete owned segment in the captured wire stream. This
// exercises the real decoder, final row flush, and runAsk's closing plain write.
type cancelAfterOwnedAnswer struct {
	llm.Client
	cancel context.CancelFunc
}

func (c cancelAfterOwnedAnswer) Stream(ctx context.Context, r llm.Request, delta func(string)) (llm.Response, error) {
	var raw strings.Builder
	return c.Client.Stream(ctx, r, func(s string) {
		if ctx.Err() != nil {
			return
		}
		raw.WriteString(s)
		delta(s)
		if strings.Contains(raw.String(), "good morning[/lang]") {
			c.cancel()
		}
	})
}

func TestAskFinalOwnedRowSurvivesTermination(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		for _, live := range []bool{false, true} {
			name := "success/append"
			if cancelled {
				name = "cancel/append"
			}
			if live {
				name = strings.Replace(name, "append", "live", 1)
			}
			t.Run(name, func(t *testing.T) {
				d, fake, _, _ := askRig(t)
				d.lang = "en"
				d.scheme = holderFor(store.SchemeDark) // the 236 asserted below
				fake.Script("", llmtest.Reply{Capture: "stream-language.sse"})
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if cancelled {
					d.newLLM = func(cfg llm.Config) llm.Client { return cancelAfterOwnedAnswer{llm.New(cfg), cancel} }
				}
				var appendOut, errOut bytes.Buffer
				var out io.Writer = &appendOut
				var screen *liveScreen
				if live {
					screen = newLiveScreen(io.Discard, 8, 20)
					screen.attachScheme(d.scheme)
					defer screen.Stop()
					out = screen
				}
				sess := &session{}
				if code := runAsk(ctx, d, options{color: true, width: 20, tintOn: true}, sess, question{text: "Explain buenos días"}, out, &errOut); code != 0 {
					t.Fatalf("code %d: %s", code, &errOut)
				}
				if len(sess.turns) != 1 {
					t.Fatalf("history %+v", sess.turns)
				}
				ending := "the -a ending."
				if cancelled {
					ending = "good morning"
				}
				if !strings.HasSuffix(sess.turns[0].Answer, ending) || strings.Contains(sess.turns[0].Answer, "[lang") {
					t.Fatalf("history not finalized cleanly: %q", sess.turns[0].Answer)
				}
				painted := appendOut.String()
				if live {
					painted = screen.PaintedTranscript()
				}
				rows := strings.Split(strings.TrimSuffix(painted, "\n"), "\n")
				last := rows[len(rows)-1]
				cells, _ := rowTestCells(t, last, 20)
				for col, c := range cells {
					if c.bg != 236 {
						t.Fatalf("final target row lost fill at column %d: %q", col, last)
					}
				}
			})
		}
	}
}

func TestPartialOwnedRowPlainTermination(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		owned      bool
	}{
		{"newline", "\n", true}, {"crlf", "\r\n", true}, {"newline then next row", "\nnext", true}, {"substantive append", " hello\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLiveScreen(io.Discard, 8, 20)
			defer l.Stop()
			if err := l.WriteOutput(renderedOutput{text: "hola", rows: []rowPaint{{tinted: true}}}); err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(l, tc.text); err != nil {
				t.Fatal(err)
			}
			first := strings.Split(l.PaintedTranscript(), "\n")[0]
			cells, _ := rowTestCells(t, first, 20)
			want := -1
			if tc.owned {
				want = 236
			}
			for col, c := range cells {
				if c.bg != want {
					t.Fatalf("column %d background %d want %d, row %q", col, c.bg, want, first)
				}
			}
		})
	}
}

func TestPartialOwnedRowStructuredTermination(t *testing.T) {
	for _, tc := range []struct {
		name   string
		output renderedOutput
		owned  bool
	}{
		// Each fragment's metadata is the OPPOSITE of the row's expected paint, so
		// a partial row that took it would fail. (Before #70 the fragments carried
		// the other shade; a row now keeps one bit, so the contrast is the bit.)
		{"newline metadata", renderedOutput{text: "\n", rows: []rowPaint{{}}}, true},
		{"crlf metadata", renderedOutput{text: "\r\n", rows: []rowPaint{{}}}, true},
		{"newline no metadata", renderedOutput{text: "\n"}, true},
		{"substantive no metadata", renderedOutput{text: " hello\n"}, false},
		{"substantive metadata", renderedOutput{text: " hello\n", rows: []rowPaint{{tinted: true}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLiveScreen(io.Discard, 8, 20)
			defer l.Stop()
			if err := l.WriteOutput(renderedOutput{text: "hola", rows: []rowPaint{{tinted: true}}}); err != nil {
				t.Fatal(err)
			}
			if err := l.WriteOutput(tc.output); err != nil {
				t.Fatal(err)
			}
			first := strings.Split(l.PaintedTranscript(), "\n")[0]
			cells, _ := rowTestCells(t, first, 20)
			want := -1
			if tc.owned {
				want = 236
			}
			for col, c := range cells {
				if c.bg != want {
					t.Fatalf("column %d background %d want %d, row %q", col, c.bg, want, first)
				}
			}
		})
	}
}
