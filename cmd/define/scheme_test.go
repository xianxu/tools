package main

import (
	"errors"
	"path/filepath"
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
		"auto": {}, "AUTO": {},
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

func TestConfigDirFrom(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
		ok   bool
	}{
		{"xdg wins", map[string]string{"XDG_CONFIG_HOME": "/x", "HOME": "/h"}, "/x/define", true},
		{"home fallback", map[string]string{"HOME": "/h"}, "/h/.config/define", true},
		{"relative xdg is ignored", map[string]string{"XDG_CONFIG_HOME": "rel", "HOME": "/h"}, "/h/.config/define", true},
		{"nothing usable", map[string]string{"HOME": "rel"}, "", false},
		{"empty", nil, "", false},
	} {
		if got, ok := configDirFrom(env(tc.env)); got != tc.want || ok != tc.ok {
			t.Errorf("%s: got %q,%v want %q,%v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// The production wiring, in process (lessons: adding a field is not wiring it).
func TestRealDepsConfigDirReadsXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	d := realDeps()
	if d.configDir == nil {
		t.Fatal("realDeps supplies no config directory, so a saved scheme is never read")
	}
	if got, ok := d.configDir(); !ok || got != filepath.Join(dir, "define") {
		t.Fatalf("configDir() = %q, %v; want %q", got, ok, filepath.Join(dir, "define"))
	}
}

// fakePersister is the durable half of /scheme, stateful: it holds what was
// saved, and can be told to fail.
type fakePersister struct {
	saved   store.Scheme
	failing error
	garbled error // what load reports for a file it cannot parse
	loads   int
}

func (f *fakePersister) load() (store.Scheme, bool, error) {
	f.loads++
	if f.garbled != nil {
		return "", false, f.garbled
	}
	return f.saved, f.saved != "", nil
}

func (f *fakePersister) save(s store.Scheme) error {
	if f.failing != nil {
		return f.failing
	}
	f.saved = s
	return nil
}

func (f *fakePersister) clear() error {
	if f.failing != nil {
		return f.failing
	}
	f.saved = ""
	return nil
}

func TestApplyScheme(t *testing.T) {
	light, dark := store.SchemeLight, store.SchemeDark
	boom := errors.New("disk full")
	type want struct {
		v     store.Scheme
		src   schemeSource
		saved store.Scheme
		err   error // compared with errors.Is; nil means none
	}
	for _, tc := range []struct {
		name    string
		start   schemeState
		arg     schemeArg
		p       *fakePersister // nil: nowhere to save
		prior   store.Scheme   // what the fake holds before
		session bool
		want    want
	}{
		{"save light", schemeState{}, schemeArg{value: light}, &fakePersister{}, "", true, want{light, sourceSaved, light, nil}},
		{"a failed save changes nothing", schemeState{}, schemeArg{value: light}, &fakePersister{failing: boom}, "", true, want{dark, sourceDefault, "", boom}},
		{"nowhere to save, in a session", schemeState{}, schemeArg{value: light}, nil, "", true, want{light, sourceSession, "", nil}},
		{"nowhere to save, one-shot", schemeState{}, schemeArg{value: light}, nil, "", false, want{dark, sourceDefault, "", errNowhereToSave}},
		{"a flag choice is replaced", schemeState{}.withChoice(light, choiceFlag), schemeArg{value: dark}, &fakePersister{}, "", true, want{dark, sourceSaved, dark, nil}},
		{"auto reveals the reply", schemeState{}.withDetected(dark).withChoice(light, choiceSaved), schemeArg{}, &fakePersister{}, light, true, want{dark, sourceDetected, "", nil}},
		{"a failed clear changes nothing", schemeState{}.withChoice(light, choiceSaved), schemeArg{}, &fakePersister{failing: boom}, light, true, want{light, sourceSaved, light, boom}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newSchemeHolder(tc.start)
			var p schemePersister
			if tc.p != nil {
				tc.p.saved = tc.prior
				p = tc.p
			}
			st, err := applyScheme(h, tc.arg, p, tc.session)
			if (tc.want.err == nil) != (err == nil) || tc.want.err != nil && !errors.Is(err, tc.want.err) {
				t.Fatalf("err = %v, want %v", err, tc.want.err)
			}
			for _, got := range []schemeState{st, h.Load()} {
				if v, src := got.effective(); v != tc.want.v || src != tc.want.src {
					t.Fatalf("effective = %s/%v, want %s/%v", v, src, tc.want.v, tc.want.src)
				}
			}
			if tc.p != nil && tc.p.saved != tc.want.saved {
				t.Fatalf("saved %q, want %q", tc.p.saved, tc.want.saved)
			}
		})
	}
	if _, err := applyScheme(nil, schemeArg{value: light}, &fakePersister{}, true); !errors.Is(err, errNoScheme) {
		t.Fatalf("a nil holder must refuse: %v", err)
	}
}

func TestDescribeScheme(t *testing.T) {
	light, dark := store.SchemeLight, store.SchemeDark
	for _, tc := range []struct {
		st         schemeState
		fullScreen bool
		want       string
	}{
		{schemeState{}.withChoice(light, choiceSaved), true, "light (saved)"},
		{schemeState{}.withDetected(dark), true, "dark (detected)"},
		{schemeState{}.withChoice(light, choiceFlag), false, "light (-scheme flag)"},
		{schemeState{}.withChoice(light, choiceSession), true, "light (session only; not saved)"},
		{schemeState{}, true, "dark (default: the terminal has not reported its background)"},
		{schemeState{}, false, "dark (default: detected only in a full-screen session)"},
	} {
		if got := describeScheme(tc.st, tc.fullScreen); got != tc.want {
			t.Errorf("describeScheme = %q, want %q", got, tc.want)
		}
	}
}

// The startup precedence, pure: no pty, no filesystem (the M2 review's
// ARCH-PURE finding — this order used to live in run() glue, pinned only
// through a terminal).
func TestInitialSchemeState(t *testing.T) {
	light, dark := store.SchemeLight, store.SchemeDark
	garbled := errors.New("scheme: \"sepia\" is not a colour scheme")
	for _, tc := range []struct {
		name     string
		flag     schemeArg
		p        *fakePersister // nil: no config directory
		want     store.Scheme
		src      schemeSource
		warnings int
		loads    int
	}{
		{"flag beats saved, without reading it", schemeArg{value: dark}, &fakePersister{saved: light, garbled: garbled}, dark, sourceFlag, 0, 0},
		{"saved when no flag", schemeArg{}, &fakePersister{saved: light}, light, sourceSaved, 0, 1},
		{"nothing saved", schemeArg{}, &fakePersister{}, dark, sourceDefault, 0, 1},
		{"garbled warns once and is unset", schemeArg{}, &fakePersister{garbled: garbled}, dark, sourceDefault, 1, 1},
		{"no config directory", schemeArg{}, nil, dark, sourceDefault, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var warn strings.Builder
			var p schemePersister
			if tc.p != nil {
				p = tc.p
			}
			st := initialSchemeState(tc.flag, p, &warn)
			if v, src := st.effective(); v != tc.want || src != tc.src {
				t.Fatalf("effective = %s/%v, want %s/%v", v, src, tc.want, tc.src)
			}
			if got := strings.Count(warn.String(), "define: ignoring saved scheme:"); got != tc.warnings {
				t.Fatalf("%d warnings, want %d: %q", got, tc.warnings, warn.String())
			}
			if tc.p != nil && tc.p.loads != tc.loads {
				t.Fatalf("loaded %d times, want %d", tc.p.loads, tc.loads)
			}
		})
	}
	if !(schemeArg{}).auto() {
		t.Fatal("the zero schemeArg must be auto, so it forgets rather than saving a blank")
	}
}
