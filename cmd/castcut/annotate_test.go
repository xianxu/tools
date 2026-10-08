package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serveTake runs the real handler on a real 127.0.0.1 listener, built with the
// bound port exactly as runAnnotate builds it.
func serveTake(t *testing.T) (base, castPath string) {
	t.Helper()
	dir := copyFixture(t)
	castPath = filepath.Join(dir, "take.cast")
	os.Remove(sidecarPath(castPath))
	s := httptest.NewUnstartedServer(nil)
	s.Config.Handler = newAnnotateHandler(castPath, s.Listener.Addr().(*net.TCPAddr).Port)
	s.Start()
	t.Cleanup(s.Close)
	return s.URL, castPath
}

func do(t *testing.T, method, url, body string, mod func(*http.Request)) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if mod != nil {
		mod(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestAnnotateServesTheTakeAndItsNotes(t *testing.T) {
	base, castPath := serveTake(t)
	if code, body := do(t, "GET", base+"/", "", nil); code != 200 || !strings.Contains(body, "fetch('/notes', { method: 'PUT'") ||
		!strings.Contains(body, `<textarea id="notes" disabled`) {
		t.Errorf("GET /: %d", code)
	}
	if code, body := do(t, "GET", base+"/cast", "", nil); code != 200 || body != string(mustRead(t, castPath)) {
		t.Errorf("GET /cast: %d, %d bytes", code, len(body))
	}
	if code, body := do(t, "GET", base+"/session", "", nil); code != 200 || body != `{"cast":"take.cast","notes":"take.captions.txt"}`+"\n" {
		t.Errorf("GET /session: %d %s", code, body)
	}
	if code, body := do(t, "GET", base+"/notes", "", nil); code != 200 || body != "" {
		t.Errorf("GET /notes before any save: %d %q", code, body)
	}
	notes := "~0:03.0  first\n~0:12.5  second\n"
	if code, _ := do(t, "PUT", base+"/notes", notes, func(r *http.Request) { r.Header.Set("Origin", base) }); code != 204 {
		t.Fatalf("PUT /notes: %d", code)
	}
	if got := string(mustRead(t, sidecarPath(castPath))); got != notes {
		t.Errorf("sidecar = %q", got)
	}
	if _, body := do(t, "GET", base+"/notes", "", nil); body != notes {
		t.Errorf("GET /notes after save = %q", body)
	}
	// What annotate saves, cut reads.
	if _, err := parseCaptions("", mustRead(t, sidecarPath(castPath))); err != nil {
		t.Errorf("saved notes do not parse as captions: %v", err)
	}
	if fi, _ := os.Stat(sidecarPath(castPath)); fi.Mode().Perm() != 0o644 {
		t.Errorf("sidecar mode %v, want 0644", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(castPath))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestAnnotateGuards(t *testing.T) {
	base, castPath := serveTake(t)
	sidecar := sidecarPath(castPath)
	os.WriteFile(sidecar, []byte("keep me\n"), 0o644)

	for _, route := range []string{"/", "/cast", "/notes", "/session"} {
		if code, _ := do(t, "GET", base+route, "", func(r *http.Request) { r.Host = "evil.example:80" }); code != 403 {
			t.Errorf("GET %s with a foreign Host: %d, want 403", route, code)
		}
	}
	for name, mod := range map[string]func(*http.Request){
		"foreign host":   func(r *http.Request) { r.Host = "localhost" + strings.TrimPrefix(base, "http://127.0.0.1") },
		"foreign origin": func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") },
		"origin on another port": func(r *http.Request) {
			r.Header.Set("Origin", "http://127.0.0.1:1")
		},
	} {
		if code, _ := do(t, "PUT", base+"/notes", "overwritten", mod); code != 403 {
			t.Errorf("PUT with %s: %d, want 403", name, code)
		}
	}
	if code, _ := do(t, "PUT", base+"/notes", strings.Repeat("x", maxNotes+1), nil); code != 413 {
		t.Errorf("oversized PUT: %d, want 413", code)
	}
	if got := string(mustRead(t, sidecar)); got != "keep me\n" {
		t.Errorf("a refused write changed the sidecar: %q", got)
	}
	if code, _ := do(t, "DELETE", base+"/notes", "", nil); code != 405 {
		t.Errorf("DELETE /notes: %d", code)
	}
	if code, _ := do(t, "GET", base+"/etc/passwd", "", nil); code != 404 {
		t.Errorf("unknown path: %d", code)
	}
}

func TestAnnotateCLIRefusesWhatCutCouldNotRead(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "v2.cast")
	os.WriteFile(bad, []byte(`{"version": 2}`+"\n[0, \"o\", \"x\"]\n"), 0o644)
	if code, _, stderr := runCLI(t, "annotate", "--no-open", bad); code != 1 || !strings.Contains(stderr, "expected asciicast v3") {
		t.Errorf("exit %d %q", code, stderr)
	}
	if code, _, _ := runCLI(t, "annotate"); code != 2 {
		t.Errorf("no take: exit %d", code)
	}
}

func TestAnnotateNext(t *testing.T) {
	for _, tc := range []struct {
		isCut, notes bool
		want         string
	}{
		{false, true, "next: castcut cut rec/take-01.cast -o rec/take-01-cut.cast\n      then castcut annotate rec/take-01-cut.cast"},
		{false, false, "next: stamp captions with Alt+T; they save to rec/take-01.captions.txt"},
		{true, true, "next: embed it — castcut --help, section EMBEDDING"},
	} {
		if got := annotateNext("rec/take-01.cast", tc.isCut, tc.notes); !strings.HasPrefix(got, tc.want) {
			t.Errorf("annotateNext(cut=%v, notes=%v) = %q\nwant prefix %q", tc.isCut, tc.notes, got, tc.want)
		}
	}
}
