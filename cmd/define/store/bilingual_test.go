package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBilingualSettingRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if !ReadBilingual(dir) {
		t.Fatal("missing setting must default on")
	}
	for _, on := range []bool{false, true, false} {
		if err := WriteBilingual(dir, on); err != nil {
			t.Fatal(err)
		}
		// A fresh read has no session state to inherit, just as after a restart.
		if got := ReadBilingual(dir); got != on {
			t.Fatalf("reload = %v, want %v", got, on)
		}
		b, err := os.ReadFile(filepath.Join(dir, BilingualFileName()))
		if err != nil {
			t.Fatal(err)
		}
		want := "off\n"
		if on {
			want = "on\n"
		}
		if string(b) != want {
			t.Fatalf("saved %q, want %q", b, want)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("setting changes left %d files, want one", len(entries))
	}
}

func TestBilingualSettingMalformedDefaultsOn(t *testing.T) {
	for _, body := range []string{"", " ", "false", "0", "off on", "off\x00", "off" + strings.Repeat(" ", 1024*1024)} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, BilingualFileName()), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if !ReadBilingual(dir) {
			t.Fatalf("malformed setting (%d bytes) disabled bilingual", len(body))
		}
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, BilingualFileName()), 0700); err != nil {
		t.Fatal(err)
	}
	if !ReadBilingual(dir) {
		t.Fatal("unreadable setting must default on")
	}
}

func TestBilingualSettingHumanWhitespace(t *testing.T) {
	for _, body := range []string{"off", "off\n", " OFF \r\n"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, BilingualFileName()), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if ReadBilingual(dir) {
			t.Fatalf("saved %q must disable bilingual", body)
		}
	}
}

func TestBilingualSettingFailedReplacementPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, BilingualFileName())
	// A directory at the destination forces rename to fail, regardless of the
	// user's privileges. Keep a saved off setting inside it to detect damage.
	if err := WriteBilingual(destination, false); err != nil {
		t.Fatal(err)
	}
	if err := WriteBilingual(dir, true); err == nil {
		t.Fatal("replacement of a nonempty directory succeeded")
	}
	if ReadBilingual(destination) {
		t.Fatal("failed replacement damaged the existing destination")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != BilingualFileName() || !entries[0].IsDir() {
		t.Fatalf("failed replacement left temporary files or changed destination: %v", entries)
	}
}

func FuzzBilingualSetting(f *testing.F) {
	for _, s := range []string{"on", "off\n", " OFF ", "", "off\x00", strings.Repeat("x", 64)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := parseBilingualSetting([]byte(s))
		// Independent language of accepted off values; everything else defaults on.
		trimmed := strings.TrimSpace(s)
		isOff := len(trimmed) == 3 && (trimmed[0] == 'o' || trimmed[0] == 'O') && (trimmed[1] == 'f' || trimmed[1] == 'F') && (trimmed[2] == 'f' || trimmed[2] == 'F')
		want := !(len(s) <= 32 && isOff)
		if got != want {
			t.Fatalf("parse(%q) = %v, want %v", s, got, want)
		}
	})
}
