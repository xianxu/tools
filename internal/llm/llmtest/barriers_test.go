package llmtest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func barrierWait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("barrier not reached")
	}
}

func TestReplyBarrierReleasePreservesCapture(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "stream"}[streaming], func(t *testing.T) {
			f := NewFake(t)
			started, release := make(chan struct{}), make(chan struct{})
			textStarted, textRelease := make(chan struct{}), make(chan struct{})
			afterText, finish := make(chan struct{}), make(chan struct{})
			capture := "message-thinking.json"
			if streaming {
				capture = "stream-sample.sse"
			}
			f.Script("", Reply{Capture: capture, Started: started, Release: release, TextStarted: textStarted, TextRelease: textRelease, AfterText: afterText, FinishRelease: finish})
			done := make(chan struct{})
			var got string
			go func() {
				defer close(done)
				resp, err := http.Post(f.URL, "application/json", strings.NewReader(`{"stream":`+map[bool]string{false: "false", true: "true"}[streaming]+`}`))
				if err != nil {
					return
				}
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				got = string(b)
			}()
			barrierWait(t, started)
			select {
			case <-done:
				t.Fatal("response escaped barrier")
			default:
			}
			close(release)
			if streaming {
				barrierWait(t, textStarted)
				close(textRelease)
				barrierWait(t, afterText)
				select {
				case <-done:
					t.Fatal("stream completed before release")
				default:
				}
				close(finish)
			}
			barrierWait(t, done)
			if got != string(Capture(t, capture)) {
				t.Fatal("barriers altered capture")
			}
		})
	}
}

func TestReplyBarrierCancellationReleasesHandler(t *testing.T) {
	f := NewFake(t)
	started := make(chan struct{})
	f.Script("", Reply{Started: started, Release: make(chan struct{}), Text: "ok"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{}`)).WithContext(ctx)
	done := make(chan struct{})
	go func() { defer close(done); f.serve(httptest.NewRecorder(), req) }()
	barrierWait(t, started)
	cancel()
	barrierWait(t, done)
}
