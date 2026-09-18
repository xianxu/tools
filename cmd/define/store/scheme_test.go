package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseScheme(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Scheme
		ok   bool
	}{
		{"dark", SchemeDark, true}, {"light", SchemeLight, true},
		{" Light\n", SchemeLight, true}, {"DARK", SchemeDark, true},
		{"", "", false}, {"auto", "", false}, {"solarized", "", false},
	} {
		got, err := ParseScheme(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseScheme(%q) = %q, %v; want %q, ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
}

// The saved scheme lives in the USER's config directory (#70). These run
// against t.TempDir(), never a real config.
func TestSchemeSettingRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "define") // does not exist yet
	if s, found, err := ReadScheme(dir); s != "" || found || err != nil {
		t.Fatalf("nothing saved: got %q, %v, %v", s, found, err)
	}
	if err := WriteScheme(dir, SchemeLight); err != nil {
		t.Fatal(err)
	}
	if s, found, err := ReadScheme(dir); s != SchemeLight || !found || err != nil {
		t.Fatalf("after save: got %q, %v, %v", s, found, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "scheme"))
	if err != nil || string(b) != "light\n" {
		t.Fatalf("saved %q (%v), want \"light\\n\"", b, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("a save left %d entries (%v), want exactly the setting", len(entries), err)
	}
}

func TestSchemeSettingParsesLooselyAndRefusesTheRest(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "scheme"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	if s, found, err := ReadScheme(write(t, "  DARK \n")); s != SchemeDark || !found || err != nil {
		t.Fatalf("trimmed, any case: got %q, %v, %v", s, found, err)
	}
	// THE CAP, with a VALID word so only the cap can refuse it.
	if s, _, err := ReadScheme(write(t, "light"+strings.Repeat(" ", 59))); s != SchemeLight || err != nil {
		t.Fatalf("64 bytes is within the cap: got %q, %v", s, err)
	}
	for _, body := range []string{"light" + strings.Repeat(" ", 60), "sepia", ""} {
		dir := write(t, body)
		s, found, err := ReadScheme(dir)
		if s != "" || found || err == nil {
			t.Fatalf("%d-byte %q: got %q, %v, %v — a garbled file is an error, never a silent default", len(body), strings.TrimSpace(body), s, found, err)
		}
		if !strings.Contains(err.Error(), filepath.Join(dir, "scheme")) {
			t.Errorf("the error must name the file so the warning says where to fix it: %v", err)
		}
	}
}

func TestClearSchemeRemovesOnlyWhatIsOurs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "define")
	if err := WriteScheme(dir, SchemeDark); err != nil {
		t.Fatal(err)
	}
	if err := ClearScheme(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("an emptied config dir is ours and goes: %v", err)
	}
	if err := ClearScheme(dir); err != nil {
		t.Fatalf("clearing nothing is not an error: %v", err)
	}

	shared := t.TempDir()
	if err := os.WriteFile(filepath.Join(shared, "foreign"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteScheme(shared, SchemeLight); err != nil {
		t.Fatal(err)
	}
	if err := ClearScheme(shared); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(shared, "foreign")); err != nil {
		t.Fatal("a directory with something else in it stays, and so does the something")
	}

	// A dotfile manager's SYMLINKED config dir: os.Remove would unlink the link
	// even though its target is full.
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "foreign"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "define")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteScheme(link, SchemeLight); err != nil {
		t.Fatal(err)
	}
	if err := ClearScheme(link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("the symlinked config dir was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "foreign")); err != nil {
		t.Fatal("the link's target lost its other file")
	}
	if _, err := os.Stat(filepath.Join(target, "scheme")); !os.IsNotExist(err) {
		t.Fatal("the setting itself must go")
	}
}
