package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordArgs(t *testing.T) {
	for _, tc := range []struct {
		o    recordOpts
		cmd  []string
		want string
	}{
		{recordOpts{Cols: 95, Rows: 36}, []string{"./parley_app", "--demo"},
			"rec --return --output-format asciicast-v3 --window-size 95x36 --command ./parley_app --demo out.cast"},
		{recordOpts{Cols: 80, Rows: 24, CaptureInput: true, IdleLimit: 1.5, Headless: true}, []string{"couch", "a b", "it's"},
			"rec --return --output-format asciicast-v3 --window-size 80x24 --capture-input --idle-time-limit 1.5 --headless --command couch 'a b' 'it'\\''s' out.cast"},
	} {
		got := recordArgs("out.cast", tc.o, tc.cmd)
		// --command is one argv element: the joined, quoted command.
		if i := indexOf(got, "--command"); i < 0 || got[i+1] != shellJoin(tc.cmd) {
			t.Errorf("--command not a single element: %q", got)
		}
		if s := strings.Join(got, " "); s != tc.want {
			t.Errorf("recordArgs:\n got %s\nwant %s", s, tc.want)
		}
	}
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}

func TestShellJoinRoundTripsThroughSh(t *testing.T) {
	words := []string{"plain", "with space", "it's", `"dq"`, "$HOME", "a;b", "", "*"}
	got := shellJoin(words)
	want := "plain 'with space' 'it'\\''s' '\"dq\"' '$HOME' 'a;b' '' '*'"
	if got != want {
		t.Errorf("shellJoin = %s\nwant       %s", got, want)
	}
}

func TestNextTake(t *testing.T) {
	dir := t.TempDir()
	if got, _ := nextTake(filepath.Join(dir, "missing")); filepath.Base(got) != "take-01.cast" {
		t.Errorf("empty dir: %s", got)
	}
	for _, f := range []string{"take-01.cast", "take-03.cast", "notes.txt", "take-07.captions.txt", "take-x.cast"} {
		os.WriteFile(filepath.Join(dir, f), nil, 0o644)
	}
	if got, _ := nextTake(dir); filepath.Base(got) != "take-04.cast" {
		t.Errorf("got %s, want take-04.cast", got)
	}
	os.WriteFile(filepath.Join(dir, "take-99.cast"), nil, 0o644)
	if got, _ := nextTake(dir); filepath.Base(got) != "take-100.cast" {
		t.Errorf("after 99: %s", got)
	}
}

// fakeAsciinema models asciinema 3's `rec` closely enough to stand in for it:
// it runs --command through sh, writes a v3 cast with the window size, an `o`
// event and an `x` exit event, logs its argv, and exits with the command's
// status only under --return (asciinema 3 exits 0 otherwise).
const fakeAsciinema = `#!/bin/sh
printf '%s\n' "$@" > "$CASTCUT_FAKE_LOG"
shift # rec
ret=0 size=80x24 cmd=
while [ $# -gt 1 ]; do
  case "$1" in
    --return) ret=1 ;;
    --window-size) size=$2; shift ;;
    --output-format|--idle-time-limit) shift ;;
    --command) cmd=$2; shift ;;
  esac
  shift
done
out=$1
[ -e "$out" ] && { echo "file exists" >&2; exit 1; }
text=$(sh -c "$cmd"); code=$?
printf '{"version":3,"term":{"cols":%s,"rows":%s}}\n[0.1, "o", "%s"]\n[0.1, "x", "%s"]\n' "${size%x*}" "${size#*x}" "$text" "$code" > "$out"
[ $ret = 1 ] && exit $code
exit 0
`

// recordContract is what castcut record must do with either asciinema.
func recordContract(t *testing.T, dir string, extra ...string) {
	t.Helper()
	out := filepath.Join(dir, "sub", "t.cast")
	args := append([]string{"record", "-o", out}, extra...)
	args = append(args, "--", "sh", "-c", "printf hi; exit 3")
	code, _, stderr := runCLI(t, args...)
	if code != 3 {
		t.Fatalf("exit %d, want the command's 3\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "castcut: recording sh -c 'printf hi; exit 3' to "+out+" with asciinema (95x36)") {
		t.Errorf("does not say what it is doing:\n%s", stderr)
	}
	c := mustParse(t, out)
	var term struct{ Cols, Rows int }
	if err := json.Unmarshal(c.Header["term"], &term); err != nil || term.Cols != 95 || term.Rows != 36 {
		t.Errorf("term = %s (%v)", c.Header["term"], err)
	}
	var text string
	json.Unmarshal(c.Events[0].Data, &text)
	if c.Events[0].Kind != "o" || !strings.Contains(text, "hi") {
		t.Errorf("first event = %+v", c.Events[0])
	}
	// A second take at the same path is refused before asciinema runs.
	if code, _, stderr := runCLI(t, args...); code != 1 || !strings.Contains(stderr, "never overwrites") {
		t.Errorf("overwrite: exit %d %q", code, stderr)
	}
}

func installFake(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "asciinema"), []byte(fakeAsciinema), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	log := filepath.Join(t.TempDir(), "argv")
	t.Setenv("CASTCUT_FAKE_LOG", log)
	return log
}

func TestRecordThroughTheFake(t *testing.T) {
	log := installFake(t)
	recordContract(t, t.TempDir())
	argv := strings.Split(strings.TrimSpace(string(mustRead(t, log))), "\n")
	if argv[0] != "rec" || indexOf(argv, "--return") < 0 || argv[indexOf(argv, "--command")+1] != "sh -c 'printf hi; exit 3'" {
		t.Errorf("argv = %q", argv)
	}
}

func TestRecordDefaultsToTheNextTake(t *testing.T) {
	installFake(t)
	t.Chdir(t.TempDir())
	os.MkdirAll("recordings", 0o755)
	os.WriteFile("recordings/take-02.cast", nil, 0o644)
	code, _, stderr := runCLI(t, "record", "--", "true")
	if code != 0 || !strings.Contains(stderr, "take saved to recordings/take-03.cast") ||
		!strings.Contains(stderr, "next: castcut annotate recordings/take-03.cast") {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	if _, err := os.Stat("recordings/take-03.cast"); err != nil {
		t.Error(err)
	}
}

func TestRecordUsageAndMissingAsciinema(t *testing.T) {
	for _, args := range [][]string{
		{"record"}, {"record", "--"}, {"record", "stray", "--", "true"}, {"record", "--cols", "0", "--", "true"},
	} {
		if code, _, _ := runCLI(t, args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	t.Setenv("PATH", t.TempDir())
	code, _, stderr := runCLI(t, "record", "-o", filepath.Join(t.TempDir(), "t.cast"), "--", "true")
	if code != 1 || !strings.Contains(stderr, "asciinema not found on PATH (brew install asciinema)") {
		t.Errorf("exit %d %q", code, stderr)
	}
}
