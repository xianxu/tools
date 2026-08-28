package main_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// Mach-O (both endians, 32- and 64-bit, and universal) and ELF.
var executableMagics = [][]byte{
	{0xCF, 0xFA, 0xED, 0xFE}, {0xCE, 0xFA, 0xED, 0xFE},
	{0xFE, 0xED, 0xFA, 0xCF}, {0xFE, 0xED, 0xFA, 0xCE},
	{0xCA, 0xFE, 0xBA, 0xBE}, {0x7F, 'E', 'L', 'F'},
}

// The test is on MAGIC BYTES, not on a filename. An earlier version flagged
// "extensionless files in a source directory" and false-positived on a tracked
// symlink — the question is whether a file is an executable image, and that is
// answerable exactly.
func isExecutableImage(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	for _, m := range executableMagics {
		if bytes.Equal(b[:4], m) {
			return true
		}
	}
	return false
}

// Deliberately Fatal, never Skip. The previous version skipped on any git
// error, so it was a silent no-op in an exported tree or without git on PATH —
// a guard that reports nothing when it cannot run certifies nothing.
func git(t *testing.T, args ...string) []byte {
	t.Helper()
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git %s: %v (this guard must not be skipped)", strings.Join(args, " "), err)
	}
	return out
}

// `go test` runs with cwd set to the PACKAGE directory, so a bare `git ls-files`
// yields paths relative to it. Resolve the root explicitly; the first version of
// this test skipped the very artifact it was written for because of that.
func repoRoot(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(string(git(t, "rev-parse", "--show-toplevel")))
}

// scanForExecutables reads every blob named by want (sha -> path) out of git and
// returns the ones that are executable images.
//
// It asserts that it CONSUMED THE WHOLE LIST. That assertion is the point: the
// first version of this helper enumerated every object and then reported a clean
// result from a partial scan, because nothing compared the records read against
// the records requested and cat-file's exit status was deferred and dropped.
// Feeding it only the first five shas left the guard GREEN with a planted binary
// still reachable from HEAD — a guard carrying the exact defect it exists to
// catch. A guard that enumerates a work list must assert it reached the end of
// it, and must check the exit status of every process it depends on.
func scanForExecutables(t *testing.T, dir string, want map[string]string) []string {
	t.Helper()
	if len(want) == 0 {
		t.Fatal("nothing to scan; this test would pass vacuously")
	}

	var req strings.Builder
	for sha := range want {
		req.WriteString(sha + "\n")
	}
	cmd := exec.Command("git", "-C", dir, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(req.String())
	var errb bytes.Buffer
	cmd.Stderr = &errb
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("git cat-file: %v", err)
	}

	// Each record is "<sha> <type> <size>\n" then <size> bytes then a newline.
	var found []string
	seen := 0
	r := bufio.NewReader(pipe)
	for {
		header, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading cat-file stream: %v", err)
		}
		f := strings.Fields(header)
		if len(f) < 3 {
			// "<sha> missing": an object git was asked for and could not
			// produce. Never a reason to keep going quietly.
			t.Fatalf("git cat-file could not read %q", strings.TrimSpace(header))
		}
		size, err := strconv.Atoi(f[2])
		if err != nil {
			t.Fatalf("unparsable size in %q", header)
		}
		// The decision needs four bytes; the --batch protocol needs the whole
		// record consumed. Reading the object in full would make this guard's
		// memory scale with exactly the input class it exists to catch — a large
		// enough committed artifact would OOM it rather than be reported — and
		// the cost would grow with history forever, on every `go test`.
		n := 4
		if size < n {
			n = size
		}
		head := make([]byte, n)
		if _, err := io.ReadFull(r, head); err != nil {
			t.Fatalf("reading object %s: %v", f[0], err)
		}
		if _, err := io.CopyN(io.Discard, r, int64(size-n+1)); err != nil { // +1: record's trailing newline
			t.Fatalf("skipping object %s: %v", f[0], err)
		}
		seen++
		if f[1] == "blob" && isExecutableImage(head) {
			found = append(found, fmt.Sprintf("%s (blob %s, %d bytes)", want[f[0]], f[0][:8], size))
		}
	}

	// Both of these are load-bearing, and neither existed in the first version.
	if err := cmd.Wait(); err != nil {
		t.Fatalf("git cat-file exited %v (%s)", err, strings.TrimSpace(errb.String()))
	}
	if seen != len(want) {
		t.Fatalf("scanned %d of %d objects — a partial scan must not report a clean result", seen, len(want))
	}
	return found
}

// A compiled binary must never be tracked. `go build ./` inside cmd/define
// writes an extensionless binary named after the directory, and `git add -A`
// swept a 9.6 MB one in.
//
// This reads the INDEX — the state a .gitignore line keeps clean, and the last
// moment the mistake is free. It reads the index's BLOBS rather than the files
// on disk: the previous version opened each path and skipped anything it could
// not open, so a tracked file missing from the worktree was silently unexamined.
func TestNoCommittedBinaries(t *testing.T) {
	dir := repoRoot(t)

	want := map[string]string{}
	for _, line := range strings.Split(string(git(t, "-C", dir, "ls-files", "-s")), "\n") {
		meta, path, ok := strings.Cut(line, "\t") // "<mode> <sha> <stage>\t<path>"
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) < 2 {
			continue
		}
		want[f[1]] = path // identical content at two paths collapses; either name locates it
	}
	for _, hit := range scanForExecutables(t, dir, want) {
		t.Errorf("compiled binary is tracked: %s", hit)
	}
}

// The index is not where this class costs anything. Removing a binary in a
// FOLLOW-UP commit leaves the blob reachable, so every clone still pays: the
// 9.6 MB artifact took this branch's clone to 5.9 MB against main's 604 KB
// while `git ls-files` reported it zero times and the index guard was green.
// The fix is to rewrite the commit that adds it, and this is the test that says
// whether that worked.
func TestNoBinariesInHistory(t *testing.T) {
	dir := repoRoot(t)

	for _, hit := range scanForExecutables(t, dir, historyPaths(t, dir)) {
		t.Errorf("compiled binary in history: %s — reachable from HEAD, so it is fetched by "+
			"every clone. Rewrite the commit that adds it; deleting it in a later commit does not "+
			"remove the cost.", hit)
	}
}

// A deck is runtime state, never source. Three of its files reached commits
// because .gitignore anchored at the repo root while `go test` and the pty
// suite both run with cwd set to cmd/define/, so the deck they created matched
// no pattern.
//
// Checked against the INDEX by path, not by content: a deck file is perfectly
// valid YAML, so nothing about the file itself says it does not belong.
func TestNoTrackedRuntimeState(t *testing.T) {
	dir := repoRoot(t)
	files := strings.Split(string(git(t, "-C", dir, "ls-files", "-z")), "\x00")

	seen := 0
	for _, f := range files {
		if f == "" {
			continue
		}
		seen++
		segs := strings.Split(filepath.ToSlash(f), "/")
		for _, p := range segs {
			if isRuntimeDir(p) {
				t.Errorf("runtime deck state is tracked: %s", f)
				break
			}
		}
		// A runtime FILE, by basename. RuntimeDirs could only ever see a
		// directory, which is how user-model.md — inferred claims about the
		// learner, written into the working directory — reached none of the three
		// places RuntimeDirs was built to reach.
		//
		// This also makes the basename RESERVED, which is the point: .gitignore
		// hides these names un-anchored, so a tracked file that shares one is
		// silently un-addable after any git rm. testdata/golden/user-model.md was
		// exactly that, and is now user-model.golden.md.
		if isRuntimeFile(segs[len(segs)-1]) {
			t.Errorf("a runtime artifact's basename is tracked: %s — .gitignore hides that name "+
				"un-anchored, so this file is un-addable after a git rm. Rename it.", f)
		}
	}
	if seen == 0 {
		t.Fatal("no tracked files were examined; this test would pass vacuously")
	}
}

// historyPaths maps every path-bearing object reachable from HEAD to its path.
// rev-list --objects prints "<sha> <path>" for blobs and trees, bare shas for
// commits.
func historyPaths(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, line := range strings.Split(string(git(t, "-C", dir, "rev-list", "--objects", "HEAD")), "\n") {
		sha, path, ok := strings.Cut(line, " ")
		if !ok || path == "" {
			continue
		}
		if _, dup := out[sha]; dup {
			continue
		}
		out[sha] = path
	}
	return out
}

// The index is not where this costs anything, and TestNoTrackedRuntimeState
// checked only the index — which is the same half-fix that let a 9.6 MB binary
// stay reachable in #4 after its file was removed. Three deck files were
// committed across five commits here; removing them left every clone still
// fetching ten blobs of somebody's vocabulary until the commits were rewritten.
func TestNoRuntimeStateInHistory(t *testing.T) {
	dir := repoRoot(t)
	paths := historyPaths(t, dir)
	if len(paths) == 0 {
		t.Fatal("rev-list returned no path-bearing objects; this test would pass vacuously")
	}
	for _, path := range paths {
		segs := strings.Split(filepath.ToSlash(path), "/")
		for _, seg := range segs {
			if isRuntimeDir(seg) {
				t.Errorf("runtime deck state is reachable from HEAD: %s — rewrite the commit "+
					"that adds it; removing the file in a later commit does not remove the cost", path)
				break
			}
		}
		if isRuntimeFile(segs[len(segs)-1]) && !legacyRuntimeFilePaths[filepath.ToSlash(path)] {
			t.Errorf("a runtime artifact's basename is reachable from HEAD: %s — rewrite the "+
				"commit that adds it; removing the file in a later commit does not remove the cost", path)
		}
	}
}

// isRuntimeDir asks the store, rather than repeating its directory names.
//
// The names were hardcoded in both guards and listed again in .gitignore —
// three places with nothing keeping them in agreement, which is how #9's usage/
// reached none of them.
func isRuntimeDir(seg string) bool {
	for _, d := range store.RuntimeDirs {
		if seg == d {
			return true
		}
	}
	return false
}

// isRuntimeFile is the same question for a FILE, asked of the same store.
//
// By BASENAME, not by path segment: these are files, and a directory that
// happens to share the name is a different thing. That asymmetry is also why
// the persisted language is lang.txt rather than lang — an un-anchored
// `lang` in .gitignore would hide any DIRECTORY of that name too, which this
// guard structurally cannot see.
//
// filepath.Match because RuntimeFiles holds gitignore-style PATTERNS: two of its
// entries are families (per-language models, atomic-write shadows) rather than
// single names. Match's `?` and `*` agree with gitignore's for a single path
// element, which is all a basename is.
func isRuntimeFile(base string) bool {
	for _, pat := range store.RuntimeFiles {
		if ok, err := filepath.Match(pat, base); err == nil && ok {
			return true
		}
	}
	return false
}

// legacyRuntimeFilePaths is a RATCHET, not an exemption.
//
// The reserved-basename rule arrived with #23; this path predates it. The rename
// to user-model.golden.md clears the INDEX, but nothing clears history short of
// rewriting it, and these two blobs do not justify that: 995 bytes each of
// renderUserModel(sampleLearnerModel(), sampleMeta()) output — synthetic sample
// data, no learner content, which is the cost this guard exists to prevent.
//
// Pinned as an exact set rather than described in a comment, per the repo's own
// rule: a comment drifts, while an exact set fails the moment a SECOND path
// appears and can only ever be shortened. If history is ever rewritten for
// another reason, this map goes with it.
var legacyRuntimeFilePaths = map[string]bool{
	"cmd/define/testdata/golden/user-model.md": true,
}

// The .gitignore half, mirroring TestGitignoreCoversRuntimeDirs — including its
// anchoring rule, which is the part that has cost this repo review rounds.
//
// RuntimeDirs exists so a new runtime artifact reaches .gitignore, the index
// guard and the history guard together, and it covers directories only. So
// user-model.md slipped through all three: `git check-ignore -v user-model.md`
// matched nothing, while --reflect writes it into the CURRENT directory with
// inferred claims about the learner in it. Nothing leaked, but #23's language
// setting would have been the second instance — which is why this is a list and
// not two more lines in .gitignore.
func TestGitignoreCoversRuntimeFiles(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	lines := map[string]bool{}
	for _, l := range strings.Split(string(b), "\n") {
		lines[strings.TrimSpace(l)] = true
	}

	if len(store.RuntimeFiles) == 0 {
		t.Fatal("store.RuntimeFiles is empty; this test would pass vacuously")
	}
	for _, f := range store.RuntimeFiles {
		if !lines[f] {
			t.Errorf(".gitignore has no un-anchored %q entry — define writes it into the "+
				"working directory and a git add -A would commit it", f)
		}
		if lines["/"+f] {
			t.Errorf(".gitignore anchors %q to the repo root; go test runs in the package "+
				"directory, where an anchored pattern does not match", f)
		}
	}
}

// The loop the compiler cannot close: .gitignore is not Go, so nothing makes it
// follow store.RuntimeDirs. This does.
//
// Un-anchored patterns, deliberately — `go test` runs with cwd set to the
// PACKAGE directory, so a deck can appear at cmd/define/words/ where an anchored
// pattern would not match it. That fact has cost this repo three review rounds
// already; the comment in .gitignore records it and this test enforces it.
func TestGitignoreCoversRuntimeDirs(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	lines := map[string]bool{}
	for _, l := range strings.Split(string(b), "\n") {
		lines[strings.TrimSpace(l)] = true
	}

	for _, d := range store.RuntimeDirs {
		if !lines[d+"/"] {
			t.Errorf(".gitignore has no un-anchored %q entry — define writes there and a git add -A would commit it", d+"/")
		}
		if lines["/"+d+"/"] {
			t.Errorf(".gitignore anchors %q to the repo root; go test runs in the package directory, so that pattern misses cmd/define/%s/", d, d)
		}
	}
}

// A runtime artifact's filename is spelled in exactly ONE place in Go source.
//
// This is the mechanical form of a rule three boundary-review findings kept
// re-stating as prose: no output line, comment, or doc may spell a name the code
// owns — name the artifact ("the learner model") or derive it from its producer.
// The cost of the loose version was not cosmetic. `--reflect` printed "wrote
// user-model.md" while writing user-model.<lang>.md, and the README tells the
// learner to hand-edit that file's ## Corrections — so following the tool's own
// output put their corrections in a file UserModel() never reads.
//
// A ratchet, in this repo's established shape: it fails the moment a SECOND
// spelling appears, and it can only ever be tightened. Test files are exempt —
// a fixture or an assertion naming the concrete artifact is the point of the
// test, and store/lang_test.go derives the ones that matter anyway.
func TestRuntimeArtifactNamesAreSpelledOnceInSource(t *testing.T) {
	// Where the literal is DELIBERATE, with the reason. Everything else must
	// reach the name through store.UserModelName or store.RuntimeFiles.
	allowed := map[string]int{
		// The two consts that BUILD every learner-model name. Nothing else in
		// non-test Go may spell one — not even the doc comments here, which were
		// rewritten to describe the names rather than repeat them.
		"cmd/define/store/yaml.go": 2,
	}

	root := repoRoot(t)
	seen := 0
	for _, f := range strings.Split(string(git(t, "-C", root, "ls-files", "-z", "*.go")), "\x00") {
		if f == "" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		seen++
		// "user-model." with the dot: a FILENAME. usermodel.go writes
		// `type: user-model` as a frontmatter field value, which is the
		// artifact's type rather than its name and is not what drifts.
		n := strings.Count(string(b), "user-model.")
		if n > allowed[f] {
			t.Errorf("%s spells a runtime artifact's name %d time(s), allowed %d — name the "+
				"artifact (\"the learner model\") or derive it from store.UserModelName. "+
				"A second spelling is a second source, and the two drift.", f, n, allowed[f])
		}
	}
	if seen == 0 {
		t.Fatal("no Go source was examined; this test would pass vacuously")
	}
}

// The same rule, over the PROSE that is read as current truth.
//
// The Go ratchet above stops at `git ls-files '*.go'`, and the artifact-name
// rule was written for "no output line, comment, README line, atlas line or plan
// line". The half it did not reach was swept by hand and, predictably, left a
// live FALSE claim: the project file still said "a single user-model.md" after
// the model became one per language. Sixth finding in that family, and the
// reason it is mechanical now.
//
// SCOPE, which is the interesting decision. This binds the three artifact kinds
// a reader takes as describing the tool AS IT IS — README, atlas/ and the
// project portfolio view. It deliberately does NOT bind issues, plans, lessons
// or workshop/history/: those are dated RECORDS, and a Spec or a Log or a
// ## Revisions entry naming what was true when it was written is correct.
// Rewriting them to match today is the actual lie, which is why the rule's
// enforcement stops here rather than everywhere the string appears.
//
// A doc that spells a stale name is not a style problem: a learner who
// hand-edits the file the docs name loses their ## Corrections.
func TestProseDoesNotSpellStaleRuntimeArtifactNames(t *testing.T) {
	root := repoRoot(t)

	// Where naming the concrete file is deliberate, with the count. Layout
	// blocks, the migration's before/after, and repo-guards.md — whose SUBJECT
	// is this very naming rule, so it cannot state it without naming names.
	allowed := map[string]int{
		"README.md":            3,
		"atlas/define.md":      2,
		"atlas/repo-guards.md": 6,
	}
	binds := func(p string) bool {
		return p == "README.md" ||
			strings.HasPrefix(p, "atlas/") ||
			strings.HasPrefix(p, "workshop/projects/")
	}

	seen := 0
	for _, f := range strings.Split(string(git(t, "-C", root, "ls-files", "-z", "*.md")), "\x00") {
		p := filepath.ToSlash(f)
		if p == "" || !binds(p) || strings.HasPrefix(p, "workshop/projects/roadmap/") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		seen++
		text := currentTruthOnly(string(b))
		if n := strings.Count(text, "user-model."); n > allowed[p] {
			t.Errorf("%s spells a runtime artifact's name %d time(s), allowed %d — name the "+
				"artifact (\"the learner model\") unless the line is a current layout, the "+
				"migration's own subject, or this rule's own documentation. A stale name here "+
				"sends a learner's ## Corrections to a file nothing reads.", p, n, allowed[p])
		}
	}
	if seen == 0 {
		t.Fatal("no current-truth markdown was examined; this test would pass vacuously")
	}
}

// currentTruthOnly strips the parts of a markdown artifact that are RECORDS.
//
// A record is allowed — required, even — to name what was true when it was
// written; revising it to match today is the lie. Two shapes qualify, and both
// are self-identifying rather than listed by name, so a new one is covered
// without anyone remembering to add it:
//
//   - everything from a "## Revisions" or "## Log" heading onward;
//   - any "### " section carrying a "**closed:**" line, which is how a project
//     file marks a milestone detail block as finished.
func currentTruthOnly(text string) string {
	for _, marker := range []string{"\n## Revisions", "\n## Log"} {
		if i := strings.Index(text, marker); i >= 0 {
			text = text[:i]
		}
	}
	var kept []string
	for _, sec := range strings.Split(text, "\n### ") {
		if strings.Contains(sec, "**closed:**") {
			continue
		}
		kept = append(kept, sec)
	}
	return strings.Join(kept, "\n### ")
}
