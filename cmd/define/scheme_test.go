package main

import (
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestSchemeStateSequences(t *testing.T) {
	light, dark := store.SchemeLight, store.SchemeDark
	type step struct {
		apply func(schemeState) schemeState
		want  store.Scheme
		src   schemeSource
	}
	for _, tc := range []struct {
		name  string
		steps []step
	}{
		{"nothing known is dark by default", nil},
		{"a reply decides while nothing is chosen", []step{
			{func(s schemeState) schemeState { return s.withDetected(light) }, light, sourceDetected}}},
		{"a later reply overwrites an earlier one", []step{
			{func(s schemeState) schemeState { return s.withDetected(light) }, light, sourceDetected},
			{func(s schemeState) schemeState { return s.withDetected(dark) }, dark, sourceDetected}}},
		{"a choice outranks a reply that arrives after it", []step{
			{func(s schemeState) schemeState { return s.withChoice(dark, choiceFlag) }, dark, sourceFlag},
			{func(s schemeState) schemeState { return s.withDetected(light) }, dark, sourceFlag}}},
		{"clearing the choice reveals the reply kept underneath", []step{
			{func(s schemeState) schemeState { return s.withDetected(light) }, light, sourceDetected},
			{func(s schemeState) schemeState { return s.withChoice(dark, choiceSaved) }, dark, sourceSaved},
			{func(s schemeState) schemeState { return s.withoutChoice() }, light, sourceDetected}}},
		{"a session choice replaces a flag", []step{
			{func(s schemeState) schemeState { return s.withChoice(dark, choiceFlag) }, dark, sourceFlag},
			{func(s schemeState) schemeState { return s.withChoice(light, choiceSession) }, light, sourceSession}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s schemeState
			if v, src := s.effective(); v != dark || src != sourceDefault {
				t.Fatalf("zero state = %s/%v, want dark/default", v, src)
			}
			for i, st := range tc.steps {
				s = st.apply(s)
				if v, src := s.effective(); v != st.want || src != st.src {
					t.Fatalf("step %d: %s/%v, want %s/%v", i, v, src, st.want, st.src)
				}
			}
		})
	}
}

func TestSchemeHolder(t *testing.T) {
	var nilHolder *schemeHolder
	if got := nilHolder.Scheme(); got != store.SchemeDark {
		t.Errorf("nil holder paints %s, want dark", got)
	}
	if nilHolder.detect(store.SchemeLight) {
		t.Error("a nil holder is read-only; detect must report no change")
	}
	h := newSchemeHolder(schemeState{})
	if !h.detect(store.SchemeLight) || h.Scheme() != store.SchemeLight {
		t.Error("a reply on an undecided holder must change what is painted")
	}
	if h.detect(store.SchemeLight) {
		t.Error("the same reply twice is not a change — it must not force a repaint")
	}
	// A reply that DIFFERS from the one heard before, under a choice: nothing
	// visible changes. (Repeating the earlier reply could not tell a choice that
	// outranks a reply from one that does not.)
	if !h.choose(store.SchemeDark, choiceSaved) {
		t.Error("choosing dark over a detected light changes what is painted")
	}
	if h.detect(store.SchemeDark) || h.Scheme() != store.SchemeDark {
		t.Error("with a choice in force a reply changes nothing visible")
	}
	if h.choose(store.SchemeDark, choiceFlag) {
		t.Error("re-choosing the shade already in force changes nothing visible")
	}
	// The last reply was dark, so forgetting a dark choice paints nothing new...
	if h.forget() || h.Scheme() != store.SchemeDark {
		t.Error("forgetting a dark choice over a dark reply changes nothing painted")
	}
	// ...while forgetting a light choice reveals that dark reply.
	h.choose(store.SchemeLight, choiceSession)
	if !h.forget() || h.Scheme() != store.SchemeDark {
		t.Error("forgetting a light choice must reveal the dark reply underneath")
	}
}

// The holder's claim is concurrent safety; this is the driver that would fail
// without it (run under -race).
func TestSchemeHolderConcurrentReaders(t *testing.T) {
	h := newSchemeHolder(schemeState{})
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = h.Scheme()
				}
			}
		}()
	}
	for i := range 1000 {
		v := store.SchemeDark
		if i%2 == 0 {
			v = store.SchemeLight
		}
		h.detect(v)
	}
	close(stop)
	wg.Wait()
}

func TestParseSchemeArg(t *testing.T) {
	for in, want := range map[string]schemeArg{
		"auto": {auto: true}, "AUTO": {auto: true},
		"dark": {value: store.SchemeDark}, "light": {value: store.SchemeLight},
	} {
		if got, err := parseSchemeArg(in); err != nil || got != want {
			t.Errorf("parseSchemeArg(%q) = %+v, %v", in, got, err)
		}
	}
	if _, err := parseSchemeArg("sepia"); err == nil {
		t.Error("an unknown scheme must be refused")
	}
}

func TestParseTintFlag(t *testing.T) {
	for in, want := range map[string]bool{"on": true, "off": false, "ON": true} {
		if got, err := parseTintFlag(in); err != nil || got != want {
			t.Errorf("parseTintFlag(%q) = %v, %v", in, got, err)
		}
	}
	for _, old := range []string{"dark", "light"} {
		_, err := parseTintFlag(old)
		if err == nil || !strings.Contains(err.Error(), "-scheme "+old) {
			t.Errorf("-language-tint %s must be refused naming -scheme %s, got %v", old, old, err)
		}
	}
	if _, err := parseTintFlag("bogus"); err == nil {
		t.Error("an unknown value must be refused")
	}
}
