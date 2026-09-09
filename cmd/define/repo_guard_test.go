package main_test

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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

// compiledExtensions are compiled artifacts a magic-byte test cannot see.
//
// THE MAGIC TEST ALONE FAILED OPEN. It knows Mach-O and ELF, so a `.pyc`
// (magic `2b 0e 0d 0a`, version-dependent) sailed past it — and one did, into a
// now-public repo (#49 I-4). The comment below rejects filename heuristics, and
// rightly: "extensionless files in a source directory" false-positived on a
// symlink. But that is a rejection of GUESSING, not of naming. These extensions
// are EXACT — a file called `.pyc` is a compiled Python module, with no judgment
// involved — so they carry none of the imprecision that rule was written about.
var compiledExtensions = map[string]bool{
	".pyc": true, ".pyo": true, ".class": true,
	".o": true, ".a": true, ".so": true, ".dylib": true, ".wasm": true,
}

// acceptedCompiledBlobs is DEBT THIS GUARD HAS BEEN TOLD ABOUT, keyed by blob.
//
// Accepted rather than fixed, and the reason is in the guard's own error
// message: the only real remedy is rewriting the commit that adds it. Doing so
// here would move `7380263` — the commit `v0.1.0` tags and the formula's
// `sha256` pins — and strand every published install to buy back 7.9 KB. So the
// cost is recorded rather than paid.
//
// An entry is a decision, not a mute button: it names the blob, the size and the
// why, and TestAcceptedCompiledBlobsAreStillReachable fails if one stops
// existing, so a stale waiver cannot outlive the thing it waives.
var acceptedCompiledBlobs = map[string]string{
	"b1fc21e35796d1a4434f09c36f62e86e6b7e6879": "cmd/define/testdata/__pycache__/capture.cpython-314.pyc, " +
		"7892 bytes — #49 I-4. Deleted from the tree at 7380263 but still reachable; " +
		"rewriting would move the commit v0.1.0 tags.",
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

// splitReceiver separates a plan's qualified entity name into a receiver and a
// bare name: `store.RuntimeFiles` drops the PACKAGE qualifier when it matches the
// file's own directory, while `Entry.AlsoSpellings` keeps `Entry` as a TYPE
// qualifier so the method can be pinned to it.
//
// ONE locator, shared by both plan guards. It was written twice, and the two
// copies had ALREADY diverged — only one also accepted an assigned form. Same
// one-predicate-two-spellings family the audiourl fix closed earlier in this
// issue, with the second spelling landing in the commit that fixed the first.
func splitReceiver(name, path string) (recv, bare string) {
	qual, rest, ok := strings.Cut(name, ".")
	if !ok {
		return "", name
	}
	if filepath.Base(filepath.Dir(path)) == qual {
		return "", rest // a package qualifier the file is already inside
	}
	return qual, rest
}

// declRegexp matches a Go declaration of name, pinned to recv when there is one
// so `Entry.AlsoSpellings` cannot be satisfied by another type's method.
func declRegexp(name, recv string) *regexp.Regexp {
	if recv != "" {
		return regexp.MustCompile(`(?m)^func\s+\(\w+\s+\*?` +
			regexp.QuoteMeta(recv) + `\)\s*` + regexp.QuoteMeta(name) + `\b`)
	}
	return regexp.MustCompile(`(?m)^(func|type|var|const)\s+(\([^)]*\)\s*)?` +
		regexp.QuoteMeta(name) + `\b`)
}

// declaredInBlock reports whether name is declared as a member of a grouped
// `const (` or `var (` block — the form an iota enum takes, where the member
// carries no keyword and often no value.
//
// Scoped to the block rather than matched anywhere, because a bare identifier at
// the start of a line is also what a STRUCT FIELD looks like: `Kind KeyKind`
// inside `type Key struct` would otherwise satisfy a plan row naming `Kind`, and
// a guard a struct field can satisfy is not checking what the row claims.
func declaredInBlock(src, name string) bool {
	member := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(name) + `\b`)
	depth := 0
	for _, line := range strings.Split(src, "\n") {
		switch {
		case strings.HasPrefix(line, "const (") || strings.HasPrefix(line, "var ("):
			depth = 1
			continue
		case depth > 0 && strings.HasPrefix(line, ")"):
			depth = 0
			continue
		}
		if depth > 0 && member.MatchString(line) {
			return true
		}
	}
	return false
}

// declaredAsField reports whether `recv.name` is a FIELD of that struct.
//
// Only for a QUALIFIED row — `Region.Word`, `Key.Row` — because the type is what
// makes the check exact. An unqualified `Word` matched against every struct in
// the file would let a row name a field of something else entirely, which is the
// looseness declaredInBlock's comment refuses for the same reason.
//
// A field is a declaration, and plans name them: `Region.Word` carries the entry
// a click belongs to, which is a design fact a reader needs. A guard that could
// not see one was forcing rows to be vaguer than the design.
func declaredAsField(src, recv, name string) bool {
	if recv == "" {
		return false
	}
	open := regexp.MustCompile(`(?m)^type\s+` + regexp.QuoteMeta(recv) + `\s+struct\s*\{`)
	loc := open.FindStringIndex(src)
	if loc == nil {
		return false
	}
	body := src[loc[1]:]
	if end := strings.Index(body, "\n}"); end >= 0 {
		body = body[:end]
	}
	// A field line may declare several: `Row, Col int`.
	field := regexp.MustCompile(`(?m)^\s*(\w+\s*,\s*)*` + regexp.QuoteMeta(name) + `\b`)
	return field.MatchString(body)
}

// `go test` runs with cwd set to the PACKAGE directory, so a bare `git ls-files`
// yields paths relative to it. Resolve the root explicitly; the first version of
// this test skipped the very artifact it was written for because of that.
func repoRoot(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(string(git(t, "rev-parse", "--show-toplevel")))
}

// WHAT CONVICTS A BLOB, stated here because this is the mechanism that decides
// it and a decision rule restated elsewhere drifts from the code that runs it
// (#49 III): EITHER the first four bytes are an executable image's magic —
// Mach-O in all four flavours, universal, or ELF — OR the path carries an exact
// compiled extension from compiledExtensions. Magic alone failed open on a
// `.pyc`; the extension list is exact rather than heuristic, which is why it does
// not reintroduce the symlink false-positive that made an earlier
// filename-shaped check wrong. atlas/repo-guards.md links here rather than
// repeating this.
//
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
// `waived` is the debt this particular caller has been told about, and it is a
// PARAMETER rather than a package global for a reason the review measured
// (#49 I-1): both guards call this, but the waiver's justification —
// "rewriting would move the commit v0.1.0 tags" — is an argument about HISTORY
// and says nothing about the index. As a global it silenced both, so staging a
// NEW file carrying the same blob passed the index guard, which
// atlas/repo-guards.md calls "the last moment the mistake is free". The index
// guard passes nil.
func scanForExecutables(t *testing.T, dir string, want map[string]string, waived map[string]string) []string {
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
		// EITHER TEST CONVICTS: the magic bytes for an executable image, or an
		// exact compiled extension for the formats magic cannot generalise over.
		compiled := isExecutableImage(head) ||
			compiledExtensions[strings.ToLower(filepath.Ext(want[f[0]]))]
		if f[1] == "blob" && compiled && waived[f[0]] == "" {
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
	// nil: NOTHING is waived at the index. A blob already in history is a cost
	// already paid; the same bytes staged again today is a new mistake, and this
	// is the last moment it is free.
	for _, hit := range scanForExecutables(t, dir, want, nil) {
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

	for _, hit := range scanForExecutables(t, dir, historyPaths(t, dir), acceptedCompiledBlobs) {
		t.Errorf("compiled binary in history: %s — reachable from HEAD, so it is fetched by "+
			"every clone. Rewrite the commit that adds it; deleting it in a later commit does not "+
			"remove the cost.", hit)
	}
}

// A WAIVER MUST NOT OUTLIVE WHAT IT WAIVES. An accepted blob that is no longer
// reachable means the debt is gone (history was rewritten, or the object was
// gc'd), and a waiver left behind would silently permit the NEXT artifact that
// happens to hash the same way nothing — it would simply be dead prose claiming
// a cost the repo no longer carries.
func TestAcceptedCompiledBlobsAreStillReachable(t *testing.T) {
	dir := repoRoot(t)
	for sha, why := range acceptedCompiledBlobs {
		out, err := exec.Command("git", "-C", dir, "cat-file", "-t", sha).Output()
		if err != nil || strings.TrimSpace(string(out)) != "blob" {
			t.Errorf("accepted compiled blob %s is no longer a reachable blob (%q) — the "+
				"debt it records is gone, so delete the waiver rather than leaving a "+
				"guard hole open. Recorded as: %s", sha, strings.TrimSpace(string(out)), why)
		}
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
		"cmd/define/README.md": 3,
		"atlas/define.md":      2,
		"atlas/repo-guards.md": 6,
	}
	// A tool's own README binds too, and by SUFFIX rather than by the one path
	// this used to name. `define`'s user documentation moved from the repo
	// README to `cmd/define/README.md` — beside the binary it describes — and a
	// guard bound to a literal path silently stopped covering the very prose
	// that names `words/`, `events/` and `user-model.en.md`. Moving a document
	// out from under a check is a way of passing it that nobody notices.
	binds := func(p string) bool {
		return p == "README.md" ||
			strings.HasSuffix(p, "/README.md") ||
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
		text := currentTruthOnly(t, f, string(b))
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
//
// IT TAKES A T AND FAILS WHEN THE FILTER SWALLOWS THE ARTIFACT (R10). Truncating
// at the first record heading is only sound while records come LAST. `#40`'s
// plan grew a second "## Revisions" above "## Done when", and every guard
// reading this saw a file that stopped before the section it existed to check —
// four of them then reported "nothing to check" and skipped. A guard that
// certifies nothing while reporting success is worse than no guard, and this is
// the one place all of them pass through.
//
// So: a record heading must be the LAST top-level section. Anything after it is
// invisible, and invisible is the failure — not a state to tolerate.
func currentTruthOnly(t *testing.T, name, text string) string {
	for _, marker := range []string{"\n## Revisions", "\n## Log"} {
		i := strings.Index(text, marker)
		if i < 0 {
			continue
		}
		// What the filter is about to discard, minus the record section itself.
		rest := text[i+1:]
		if j := strings.Index(rest[1:], "\n## "); j >= 0 {
			t.Errorf("%s has a %q section with another top-level section after it. "+
				"Records are truncated at the first one, so everything below is invisible to every guard "+
				"that reads current truth — %s is unchecked. Records go LAST, once.",
				name, strings.TrimPrefix(marker, "\n## "), strings.SplitN(strings.TrimPrefix(rest[1+j:], "\n"), "\n", 2)[0])
		}
		text = text[:i]
	}
	// A DOCUMENT THAT DECLARES A SYMBOL DELETED IS THE RECORD OF THAT DELETION,
	// so its account of the symbol is a record too (#42).
	//
	// Two guards disagreed and neither could settle it alone:
	// `TestPlanTableStatusMatchesTheChangeWindow` REQUIRES a plan's Core-concepts
	// table to name what the window removed and checks it against the diff, while
	// `TestNoArtifactNamesARetiredSymbol` forbids a current-truth artifact from
	// naming a retired symbol. The plan cannot both name it and not.
	//
	// Settled HERE because "is this text a record" is this function's whole
	// question, so both guards inherit one answer instead of being taught it
	// separately. The exemption is SELF-LIMITING in two directions: only a
	// document carrying a `| deleted |` row gets it, and only for the symbol that
	// row names — so a plan cannot quiet a guard about anything it did not
	// declare, and no other artifact is exempted at all.
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "| ") || !strings.HasSuffix(strings.TrimSpace(line), "| deleted |") {
			continue
		}
		name := strings.Trim(strings.SplitN(line, "|", 3)[1], " `")
		if name == "" {
			continue
		}
		text = regexp.MustCompile("`?"+regexp.QuoteMeta(name)+"`?").ReplaceAllString(text, "")
	}
	// THE SECOND DISCARDING RULE, and R10's premise assertion applies to it too.
	//
	// Splitting on "\n### " means a closed section runs to the NEXT "### " — which
	// can swallow a following top-level "## " section whole, exactly as the
	// truncation above could. No live instance in the tree, which is why this is
	// the rule written down rather than a defect fixed: a filter with two ways to
	// discard needs the assertion on both, or the next one silently gets none.
	var kept []string
	for _, sec := range strings.Split(text, "\n### ") {
		if closedSection(sec) {
			if i := strings.Index(sec, "\n## "); i >= 0 {
				t.Errorf("%s has a closed \"### \" section with a top-level section after it (%q). "+
					"Closed sections are discarded up to the next \"### \", so that section is invisible "+
					"to every guard reading current truth. Close out the detail block before the next \"## \".",
					name, strings.SplitN(strings.TrimPrefix(sec[i:], "\n"), "\n", 2)[0])
			}
			continue
		}
		kept = append(kept, sec)
	}
	return strings.Join(kept, "\n### ")
}

// closedSection reports whether a "### " block carries a project file's
// `**closed:**` marker — the LINE, not the characters.
//
// `strings.Contains` was the first spelling and it was wrong in the way that
// matters for a filter: `atlas/repo-guards.md` DOCUMENTS this very rule, so the
// marker appears mid-sentence in its prose, and the whole top of that file — the
// guard inventory itself — was silently discarded from every guard reading
// current truth. It went unnoticed until R13 made discarding say so, which is
// the argument for the premise assertion in one line.
//
// A marker is written at the start of a line by whatever wrote the record. Prose
// about a marker is not a marker.
func closedSection(sec string) bool {
	for _, line := range strings.Split(sec, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "**closed:**") {
			return true
		}
	}
	return false
}

// A plan's Core-concepts table may not name an entity the tree does not have.
//
// Fourth recurrence of the symbol half of the artifact-name rule: round 1 named
// `deckDeps`, BR-6 named `MigrateFlatDeck`, and the close review found
// `dictChoice` and `dcsDictionaries` still listed. Every previous fix was a
// hand-sweep of the instances, so the family kept coming back — the ratchets
// this range added count filenames only, and a symbol is the other half of the
// rule they were written for.
//
// The table is a gift for this: it already states, in machine-readable form,
// "this identifier lives at this path". Making the plan a CONSUMER of the tree
// is the same move the README/prompt guard makes — a grep cannot fail a build,
// this can.
//
// Deliberately narrow. It checks the Name and "Lives in" cells of Core-concepts
// tables in ACTIVE plans, not prose, not history, and not rows whose file does
// not exist yet — a plan is written before the code, so an unbuilt row is a
// plan, while a WRONG row is a lie.
//
// The file-does-not-exist exemption assumed a plan CREATES the files it names,
// and #29 is the counterexample: seven of its entities are new symbols in files
// that already exist (parse.go, audiourl.go, main.go), so every unbuilt row read
// as a lie and the suite was red for the whole implementation phase — which
// costs exactly the regression signal a green suite between tasks exists to give.
//
// The STATUS cell is the missing signal, and the table already carries it. A row
// marked `new` says "this will be created here": a promise about the future, the
// same kind of statement the missing-file exemption already honours. A row
// marked `modified` / `unchanged` / `deleted` is a claim about the tree as it
// stands and stays checked. The promise is only honoured while the plan still
// has unticked steps — once every box is ticked the plan claims to be finished,
// and a finished plan naming a symbol nobody wrote is the lie this guard is for.
// `sdlc close`'s plan-unchecked gate is what makes that condition reachable.
func TestPlanTablesNameEntitiesThatExist(t *testing.T) {
	root := repoRoot(t)
	plans, err := filepath.Glob(filepath.Join(root, "workshop", "plans", "*-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) == 0 {
		// conformance:inapplicable — every plan is archived to workshop/history/,
		// which is a legitimate state between issues rather than a missing file.
		t.Skip("no active plans")
	}

	// `| `A` / `B` | `path` | ...` — the shape the plan template produces. A row
	// often names SEVERAL entities in its first cell; a first version captured
	// only the first, leaving 7 of 16 symbols unchecked in this repo's own plans.
	// The third cell is Status, in both the Pure-entities and Integration-points
	// tables. Optional in the pattern so a malformed row still gets CHECKED
	// rather than silently exempted — fail closed.
	row := regexp.MustCompile("^\\|([^|]*)\\|\\s*`([^`]+\\.go)`\\s*\\|?([^|]*)")
	nameCell := regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_.]*)`")
	checked := 0
	for _, plan := range plans {
		b, err := os.ReadFile(plan)
		if err != nil {
			t.Fatalf("reading %s: %v", plan, err)
		}
		body := currentTruthOnly(t, filepath.Base(plan), string(b))
		// An unticked step means the plan is still a plan. Scanned per plan, not
		// per row, because it is a property of the document.
		inProgress := strings.Contains(body, "- [ ] ")
		// SCOPED TO "## Core concepts", because the entity tables live there by
		// the plan template's own contract and nothing else in a plan is one.
		//
		// Unscoped, this matched any table row whose second cell was a
		// backticked `*.go` path — which is the shape of the "What this plan
		// asserts about the existing tree, verified" table, whose third column
		// is a verdict ("true — measured 2026-08-31") rather than a status word.
		// Three plans in a row tripped on it, and the workaround each time was
		// to bury a line number inside the backticks so the regex would stop
		// matching. That is a guard training authors to obfuscate their own
		// citations, which is worse than the false positive.
		for _, line := range strings.Split(coreConceptsSection(body), "\n") {
			m := row.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			path := m[2]
			status, ok := planStatus(m[3])
			if !ok {
				t.Errorf("%s: row %q has status %q, which is not one of %v — the status "+
					"column is a controlled vocabulary, and a cell outside it cannot be "+
					"checked at all", filepath.Base(plan), strings.TrimSpace(m[1]),
					strings.TrimSpace(m[3]), planStatuses)
				continue
			}
			if inProgress && status == "new" {
				continue
			}
			for _, nm := range nameCell.FindAllStringSubmatch(m[1], -1) {
				checkPlanName(t, root, filepath.Base(plan), nm[1], path, status, &checked)
			}
		}
	}
	if checked == 0 {
		// conformance:inapplicable — a plan legitimately precedes its code, so a
		// set of plans whose files do not exist yet is a design in progress, not
		// drift. Rows become checkable as their files land.
		t.Skip("no Core-concepts rows pointed at existing files")
	}
}

// coreConceptsSection is the part of a plan the entity tables live in: from the
// "## Core concepts" heading to the next "## " one.
//
// Returns the WHOLE body when the heading is absent, which fails toward
// checking too much rather than too little: a plan that has renamed its section
// should get noisy rows, not silent exemption.
func coreConceptsSection(body string) string {
	i := strings.Index(body, "## Core concepts")
	if i < 0 {
		return body
	}
	rest := body[i+len("## Core concepts"):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		return rest[:j]
	}
	return rest
}

// planStatuses is the Core-concepts status column's controlled vocabulary.
var planStatuses = []string{"new", "modified", "unchanged", "deleted"}

// planStatus normalises a status cell to that vocabulary, or reports that it is
// outside it.
//
// TWO heuristics failed here before this existed, in opposite directions, which
// is what a vocabulary is for. `Contains(lower(cell), "new")` also matched
// "renewed" and "newly"; the first-word fix then missed `**modified**`, and bold
// status cells are this repo's live convention — the #29 plan writes three of
// them. So the cell is normalised (emphasis stripped, trailing prose dropped)
// and matched against a closed set, and anything outside it FAILS LOUDLY rather
// than falling into whichever branch the heuristic happened to pick.
func planStatus(cell string) (string, bool) {
	fields := strings.Fields(strings.ToLower(cell))
	if len(fields) == 0 {
		return "", false
	}
	word := strings.Trim(fields[0], "*_`")
	return word, slices.Contains(planStatuses, word)
}

// Every TEST a plan names must exist too, wherever in the plan it is named.
//
// Third recurrence of `plan-table-incomplete`, and the previous two fixes were
// hand-edits of the rows, which is why it came back: the sibling guard above
// reads only the Core-concepts tables' first two cells, so a Done-when row's
// "pinned by" column — where a plan makes its most load-bearing claim, "this
// behaviour is defended by this test" — was unchecked. `aa4fe94` renamed
// `TestScreenFrameFitsTheTerminalInDisplayRows` and left row 1b naming it, with
// a green suite.
//
// A test name is self-identifying (`Test…`/`Fuzz…`/`Benchmark…` at the start of
// a backticked cell), so this needs no path column and no table shape — it reads
// the whole document, which is exactly the generality the family was missing.
// Subtests are addressed as `Parent/case name`; only the parent is checked,
// since the case name is prose by design.
func TestPlanNamedTestsExist(t *testing.T) {
	root := repoRoot(t)
	plans, err := filepath.Glob(filepath.Join(root, "workshop", "plans", "*-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) == 0 {
		// conformance:inapplicable — every plan is archived to workshop/history/
		// at close, a legitimate state between issues rather than a missing file.
		t.Skip("no active plans")
	}
	// Test names as they appear in Go source, anywhere under cmd/define —
	// SUBPACKAGES INCLUDED.
	//
	// This used to Glob `cmd/define/*_test.go`, which is flat, so a plan pinning
	// a test in `play/`, `store/`, `schedule/` or `puretest/` was told the test
	// "does not exist". #7 hit it with six at once: form 2.3's selection is pure
	// and its tests live in `play/` by design, which is exactly where this
	// repo's plans are SUPPOSED to put them (ARCH-PURE). A guard that fails on
	// the arrangement the architecture asks for teaches people to weaken the
	// guard, so the fix is here rather than in the plan.
	declared := map[string]bool{}
	decl := regexp.MustCompile(`(?m)^func ((?:Test|Fuzz|Benchmark)[A-Za-z0-9_]*)\(`)
	err = filepath.WalkDir(filepath.Join(root, "cmd", "define"), func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range decl.FindAllStringSubmatch(string(b), -1) {
			declared[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(declared) == 0 {
		t.Fatal("no test declarations found: the guard would certify nothing")
	}

	named := regexp.MustCompile("`((?:Test|Fuzz|Benchmark)[A-Za-z0-9_]*)(?:/[^`]*)?`")
	checked := 0
	for _, plan := range plans {
		b, err := os.ReadFile(plan)
		if err != nil {
			t.Fatalf("reading %s: %v", plan, err)
		}
		// Scoped to the unit of work the plan actually declares, which is what
		// makes this both safe and useful. Unticked tasks mean the unit is still
		// being built, so the tests its Done-when names are promises — the same
		// exemption the sibling guard gives a `new` row. A unit claiming to be
		// finished while naming a test nobody wrote is the lie this guard exists
		// for.
		//
		// The UNIT is the milestone where a plan has milestones, and the whole
		// document where it does not. Per-milestone alone was the first version
		// and it was wrong for single-pass plans (AGENTS.md §3: work with one
		// boundary takes plain checkboxes, no `Mx`): there the tasks and the
		// Done-when table are separate `##` sections, so the table looked
		// finished from the moment the plan was written. Document-level alone is
		// wrong the other way — it would exempt a milestone for exactly as long
		// as that milestone was being built, which is when #30's miss happened.
		body := currentTruthOnly(t, filepath.Base(plan), string(b))
		units := planSections(body)
		if !strings.Contains(body, "## Milestone ") {
			units = []string{body}
		}
		for _, section := range units {
			if strings.Contains(section, "- [ ] ") {
				continue
			}
			for _, m := range named.FindAllStringSubmatch(section, -1) {
				checked++
				if !declared[m[1]] {
					t.Errorf("%s names the test %q, which no *_test.go declares. A plan's "+
						"\"pinned by\" column is its most load-bearing claim — that a behaviour "+
						"is DEFENDED — so a name that resolves to nothing claims coverage that "+
						"does not exist. Update the row when the test renames.",
						filepath.Base(plan), m[1])
				}
			}
		}
	}
	if checked == 0 {
		// conformance:inapplicable — a plan whose milestones are all still in
		// progress is a design being built, the same state the sibling guard's
		// `new`-row exemption honours.
		t.Skip("no finished unit of work names a test")
	}
}

// planSections splits a plan at its `## …` headings, so a claim can be judged
// against the progress of the milestone that makes it rather than the document's.
func planSections(body string) []string {
	var out []string
	var cur strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") && cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
		cur.WriteString(line + "\n")
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// checkPlanName asserts one Name cell resolves to a declaration at the stated path.
func checkPlanName(t *testing.T, root, plan, name, path, status string, checked *int) {
	t.Helper()
	{
		recv, name := splitReceiver(name, path)
		src, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			// The file does not exist yet: a plan legitimately precedes its
			// code. Only a row pointing at a REAL file makes a checkable claim.
			return
		}
		*checked++
		declared := declRegexp(name, recv)
		assigned := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `\s*:?=`)
		// A bare member of a grouped const or var block — `KeyPageUp` under an
		// iota — is a declaration with no keyword and no `=` of its own, so
		// neither pattern above can see it. Without this the guard calls a
		// perfectly real enum member missing, which is a guard telling a lie
		// about the tree it exists to check.
		// DECLARED or ASSIGNED only. A first version also accepted any
		// occurrence anywhere in the file, which admitted COMMENTS — and a
		// stale `newDeck` row stayed green solely because one comment still
		// mentioned the old name. A guard that a comment can satisfy is not
		// checking the tree.
		present := declared.Match(src) || assigned.Match(src) ||
			declaredInBlock(string(src), name) || declaredAsField(string(src), recv, name)

		// A `deleted` row asserts the OPPOSITE, and until this it could never
		// pass: the check demanded every named entity be declared, so the moment
		// the deletion actually happened the row went red for doing what it
		// said. That made `deleted` — one of the four words in the status
		// vocabulary — unusable, and the only way to keep a plan green was to
		// leave the row lying about the tree.
		//
		// Inverted, the row becomes a real pin: it fails if the entity is still
		// there, which is exactly the claim "deleted" makes.
		if status == "deleted" {
			if present {
				t.Errorf("%s marks %q at %s as deleted, but it is still declared there — "+
					"either the deletion did not happen or the row is wrong.", plan, name, path)
			}
			return
		}
		if !present {
			t.Errorf("%s names %q at %s, which does not declare it — a plan is the one "+
				"artifact a reader trusts to describe the design, so a stale entity name "+
				"there is worse than none. Update the row when the code renames.",
				plan, name, path)
		}
	}
}

// retiredSymbolNames maps a name the tree no longer declares to what replaced
// it. A rename adds a row here; the guard below then fails if the old name
// survives anywhere a reader would take as current.
//
// This is the SYMBOL half of the artifact-name rule, which recurred nine times
// while only the filename half was mechanical. The plan-table guard closed the
// tables; comments and prose stayed hand-swept, and the very commit that added
// that guard left three `newDeck` comments behind.
//
// A rename cannot be detected automatically — only the person doing it knows the
// old name — so this is the one place the rule needs a human to write something
// down. Everything after that is mechanical, and the list can only shrink as
// history archives.
var retiredSymbolNames = map[string]string{
	// #42 split selection in two: the box partition moved into todaysQuestions'
	// one loop (it must run BEFORE the lookup, so it cannot be a function over a
	// parsed entry) and the packing half became packBoards.
	//
	// NO ROW FOR `Recall`, the form deleted in the same window, and the omission
	// is deliberate: "recall" is ordinary English this program uses constantly for
	// the editor's up-arrow history, so a row would fire on `history.go`,
	// `repl.go` and half the atlas over text that has nothing to do with the form.
	// What IS retired there is the routing CLAIM, and `retiredPhrases` carries it
	// — a phrase, not a word, which is the distinction that map's own doc makes.
	"boardsFor":       "packBoards, plus the box partition inside todaysQuestions",
	"deckDeps":        "newLangDeps",
	"newDeck":         "newLangDeps",
	"MigrateFlatDeck": "MigrateToLanguages",
	"dictChoice":      "dictMeta",
	// #46 dissolved the decorator into the seam: deps.audio is a *audioSeam, so
	// there is no unwrapped source to wrap and no wrap site to remember. Four
	// plan-gate rounds went into naming which functions must call the old
	// constructor, and all four answers were wrong — the type is what stopped the
	// question being askable.
	"newCachingAudioSource": "newAudioSeam (the memo IS the seam now)",
	"cachingAudioSource":    "audioSeam",
	"dcsDictionaries":       "installedDictionaries",
	// #27 renamed this when the atlas joined the README as a consumer. The
	// rename skipped this row and left a stale mention in voice.go — the human
	// half of this mechanism failing, in the same commit that widened the test
	// it names. Adding the row is the whole discipline; everything after is
	// mechanical.
	"TestREADMEQuotesTheLocaleHelp": "TestDocsQuoteTheLocaleHelp",
	// #29 narrowed this name: with -pron the loop deliberately asks for another
	// language first, so "only" needed a "when none was named". atlas/define.md
	// cited the old name, which is exactly the stale mention this map exists to
	// catch — and the row is what turns "I should sweep the docs" into a build
	// failure.
	"TestTheFetchLoopAsksOnlyForTheSessionsLanguage": "TestTheFetchLoopAsksOnlyForTheSessionsLanguageWhenNoneWasNamed",
	// #31 merged the Spanish and Italian notation tests into one table and
	// deleted a sweep the widened strong invariant subsumes. Neither row was
	// added at the time, so the atlas and the plan named tests the tree does not
	// declare and this guard stayed green over them — the SECOND time the human
	// half of this mechanism failed (see the #27 note above), which is why
	// TestARemovedDeclarationIsSweptOrRetired now checks it mechanically.
	"TestSpanishEntriesCarryNoPronunciationNotation": "TestNonEnglishEntriesCarryNoPronunciationNotation",
	"TestRenderLosesNothingInEveryCapturedLanguage":  "TestRenderLosesNothing (widened to every captured language)",
}

// No current-truth artifact names a symbol the tree has retired.
//
// Scope matches TestProseDoesNotSpellStaleRuntimeArtifactNames plus non-test Go:
// docs that describe the tool as it IS, and the code itself. Records — issues,
// history, ## Revisions, ## Log — legitimately name what was true when written.
// Test files are exempt because a test-local variable may reuse a plain name for
// unrelated reasons.
func TestNoArtifactNamesARetiredSymbol(t *testing.T) {
	root := repoRoot(t)
	for _, f := range currentTruthFiles(t, root) {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		text := currentTruthOnly(t, f, string(b))
		for old, now := range retiredSymbolNames {
			// Word-boundaried: migrateFlatDeck must not match MigrateFlatDeck,
			// and a longer identifier containing the old name is not the old name.
			if regexp.MustCompile(`\b` + regexp.QuoteMeta(old) + `\b`).MatchString(text) {
				t.Errorf("%s names the retired symbol %q; the tree declares %q. A rename "+
					"sweeps every restatement in the SAME commit — nine findings in this "+
					"family say the hand-sweep does not hold.", f, old, now)
			}
		}
	}
}

// retiredPhrases is the CONCEPT half of the rule above: a phrase naming a DRAWN
// element the tool no longer has, mapped to what states that fact now.
//
// `retiredSymbolNames` closes the half a compiler could almost have caught — a
// name the tree stops declaring. This is the half nothing can: R11 deleted the
// board's footer toggle row and moved the live mark onto the prompt row, and
// six months of comments went on describing a row that is not drawn. No symbol
// was renamed (`toggleLine` was unexported, which `isCitableName` filters out on
// purpose), so every existing guard stayed green while five comments, one of
// them contradicting its own owner twenty lines below, told a reader to look for
// a row that is gone. It took five rounds of one boundary review to enumerate.
//
// The trigger the finding asked for, made mechanical: when a window deletes a
// drawn element, the sweep set is `grep` for what it was CALLED over
// currentTruthFiles, and this map is where that grep gets written down so it
// runs on every later commit too. A key is a PHRASE, not a word — "toggle" is
// still a live verb for what Tab does, and the row it used to name is not.
var retiredPhrases = map[string]string{
	"toggle row":      "the prompt row — `Board.Keys` states the live mark (#40 R11)",
	"footer's toggle": "the prompt row — `Board.Keys` states the live mark (#40 R11)",
	// #42 deleted form 2.1. The FORM's name is not banned — several comments
	// legitimately explain why something exists by naming the form it was built
	// for, and the atlas keeps the argument that produced `Keys()`. What is
	// retired is the ROUTING CLAIM: nothing falls back to it any more, because it
	// is not there to fall back to.
	//
	// The phrase, not the word, exactly as the two rows above are phrases. It was
	// worth writing down because the tree-wide sweep this issue ran caught four
	// files and missed five more — the instance, not the class (#42 BR-3).
	"back to form 2.1": "a word no test can be built for is TRIAGED on a board (#42)",
	// TENSE IS THE DISCRIMINATOR, and it is what closes this class (#42 BR-12).
	//
	// Round 1 declined a bare ban on the form's NAME because ~8 historical mentions
	// would redden — correctly: a comment that explains why something exists by
	// naming the form it was built for is right, and the atlas keeps the argument
	// that produced `Keys()`. But declining the ban left the class open, and three
	// rounds later seven production doc comments still said form 2.1 IS, HAS or
	// CANNOT — present tense about a form the tree does not contain, one of them
	// three lines from a block a previous round had rewritten.
	//
	// So the rows are keyed on the tense rather than the name. "was", "used to"
	// and "before #42" are untouched by construction, which is exactly the
	// distinction the sweep needed and the bare name could not express.
	"form 2.1 is":              "form 2.1 WAS — #42 retired it; a word no test fits is triaged on a board",
	"form 2.1 has":             "form 2.1 HAD — #42 retired it",
	"form 2.1 cannot":          "form 2.1 COULD NOT — #42 retired it",
	"form 2.1 grades":          "form 2.1 GRADED — #42 retired it",
	"form 2.1 shows":           "form 2.1 SHOWED — #42 retired it",
	"form 2.1 answer":          "form 2.1 ANSWERED — #42 retired it",
	"2.1 or 2.3":               "form 2.3 alone — #42 retired 2.1",
	"falls back to 2.1":        "a word no test can be built for is TRIAGED on a board (#42)",
	"back to Recall":           "a word no test can be built for is TRIAGED on a board (#42)",
	"fallback to form":         "a word no test can be built for is TRIAGED on a board (#42)",
	"falls back to the recall": "a word no test can be built for is TRIAGED on a board (#42)",
}

// No current-truth artifact describes a drawn element the tool no longer draws.
//
// Same scope and same reasoning as TestNoArtifactNamesARetiredSymbol — records
// legitimately say what was true when written, so this reads currentTruthOnly.
// Case-insensitive: these are prose phrases, and the comments that went stale
// spelled the element in three different cases.
func TestNoArtifactDescribesARetiredDrawnElement(t *testing.T) {
	root := repoRoot(t)
	for _, f := range currentTruthFiles(t, root) {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		text := currentTruthOnly(t, f, string(b))
		for old, now := range retiredPhrases {
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(old) + `\b`)
			if loc := re.FindStringIndex(text); loc != nil {
				t.Errorf("%s describes %q, a drawn element the tool no longer has; %s. "+
					"A deleted element is swept in the SAME commit — grep for what it "+
					"was called, and add the phrase here so the next commit is swept too.",
					f, text[loc[0]:loc[1]], now)
			}
		}
	}
}

// A plan's status column is a claim about the DIFF, and it is checkable.
//
// "unchanged" / "modified" are not opinions about behaviour — they say whether
// this window touched the symbol, which git already knows. Four of the #29
// plan's twenty rows were wrong across two review rounds (`fakeCDN`,
// `fakeDictionary`, `rebasedSource`, `AudioCandidates`), and the round that
// fixed them BY HAND caught three of four: the fourth had been sitting in the
// table the whole time, claiming "unchanged" about a symbol the same plan's own
// Task 8 rewrote. Hand-sweeping is what keeps failing, so this is the mechanism.
//
// DECLARATION-LEVEL, not file-level, and that distinction is the whole design:
// voice.go is modified in this window while voiceFor / localeFor /
// defaultLocale / applyVoice genuinely are not, and that row is correct. A
// file-level check would call it a lie.
//
// The doc comment counts as part of the declaration. #29's AudioCandidates row
// is the case: its body changed by one line, but ten lines of its comment were
// rewritten by the plan's own doc sweep — and a plan that says "unchanged" about
// a symbol whose documentation this window rewrote is misleading in exactly the
// way that matters to a reader.
func TestPlanTableStatusMatchesTheChangeWindow(t *testing.T) {
	root := repoRoot(t)
	base, ok := changeWindowBase(t)
	if !ok {
		// This is the ONLY skip in this guard: git being absent or broken is
		// Fatal through git(), because a guard that reports nothing when it
		// cannot run certifies nothing.
		//
		// conformance:inapplicable — on main there is no window, and a plan
		// describing no changes cannot contradict one.
		t.Skip("no change window: HEAD is the merge-base with main")
	}
	plans, err := filepath.Glob(filepath.Join(root, "workshop", "plans", "*-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) == 0 {
		// conformance:inapplicable — every plan is archived to workshop/history/
		// at close, a legitimate state between issues rather than a missing file.
		// Same reasoning as the sibling guard above.
		t.Skip("no active plans")
	}

	row := regexp.MustCompile("^\\|([^|]*)\\|\\s*`([^`]+\\.go)`\\s*\\|?([^|]*)")
	nameCell := regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_.]*)`")
	checked := 0
	for _, plan := range plans {
		// ONLY the plan this window belongs to. Another issue in flight has its
		// own branch, and its `modified` rows describe work that happened there —
		// judging them against THIS window would fail them for being someone
		// else's. The window touching the plan file is what identifies it.
		rel, err := filepath.Rel(root, plan)
		if err != nil || changedLines(t, base, rel) == nil {
			continue
		}
		b, err := os.ReadFile(plan)
		if err != nil {
			t.Fatalf("reading %s: %v", plan, err)
		}
		for _, line := range strings.Split(currentTruthOnly(t, filepath.Base(plan), string(b)), "\n") {
			m := row.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			status, ok := planStatus(m[3])
			if !ok || status == "new" || status == "deleted" {
				// `new` is checked by the sibling test (does it exist yet);
				// `deleted` has no declaration left to locate.
				continue
			}
			touched := changedLines(t, base, m[2])
			if touched == nil {
				continue // the file itself is untouched by this window
			}
			for _, nm := range nameCell.FindAllStringSubmatch(m[1], -1) {
				checkPlanStatus(t, root, filepath.Base(plan), nm[1], m[2], status, touched, &checked)
			}
		}
	}
	if checked == 0 {
		// conformance:inapplicable — a window editing only prose, or only files
		// no row names, has no status claim to contradict. Reachable on a
		// docs-only commit, which is a normal state rather than drift.
		t.Skip("no unchanged/modified rows pointed at files this window touched")
	}
}

// A PLAN'S `pinned by` COLUMN IS A CLAIM ABOUT THE TREE, and it is checkable.
//
// The sibling above checks the STATUS column — "did this window touch that
// symbol" — and reads only the `name | file.go | status` rows, which left the
// Done-when table's `pinned by` column unguarded. `#40`'s boundary review found
// it: three rows cited four tests that had never been written, and the shipped
// pins appeared only in the issue's Log. A Done-when row naming a test that does
// not exist is worse than one naming none, because it reads as evidence.
//
// The issue's own Log had recorded the lesson one round earlier — "a hand-sweep
// of a plan's tables does not hold, and this repo already knew it" — and the
// hand-sweep failed again in the column the guard could not see. So the guard
// grows rather than the discipline.
//
// EVERY backticked `Test*` identifier in an active plan, not only the Done-when
// table's: a plan citing a test anywhere is making the same claim.
func TestPlanCitesTestsThatExist(t *testing.T) {
	root := repoRoot(t)
	plans, err := filepath.Glob(filepath.Join(root, "workshop", "plans", "*-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) == 0 {
		// conformance:inapplicable — every plan is archived at close, so no
		// active plan is a legitimate state between issues.
		t.Skip("no active plans")
	}
	// The tree's test functions, by name, read once.
	declared := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range regexp.MustCompile(`func (Test[A-Za-z0-9_]*)\(`).FindAllStringSubmatch(string(b), -1) {
			declared[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(declared) == 0 {
		t.Fatal("found no test functions in the tree; this guard would pass vacuously")
	}

	cite := regexp.MustCompile("`(Test[A-Za-z0-9_]*)`")
	checked, donewhen := 0, 0
	for _, plan := range plans {
		b, err := os.ReadFile(plan)
		if err != nil {
			t.Fatalf("reading %s: %v", plan, err)
		}
		body := currentTruthOnly(t, filepath.Base(plan), string(b))
		if strings.Contains(body, "## Done when") {
			donewhen++
		}
		for _, m := range cite.FindAllStringSubmatch(body, -1) {
			checked++
			if !declared[m[1]] {
				t.Errorf("%s cites `%s` and no such test exists. A plan naming a test that was "+
					"never written reads as evidence; write it, or cite the one that shipped.",
					filepath.Base(plan), m[1])
			}
		}
	}
	if checked == 0 && donewhen > 0 {
		// NOT A SKIP. `checked == 0` beside a Done-when table means the citations
		// were swallowed — which is exactly how this guard passed while `#40`'s
		// plan cited four tests that did not exist (R10). A guard reading a
		// FILTERED view has to assert its premise about that view and fail, not
		// report success about a file it never saw.
		t.Errorf("%d plan(s) have a Done-when table and not one cites a test. Either the rows "+
			"name no pin — which is what the table is for — or currentTruthOnly truncated them away.", donewhen)
	}
	if checked == 0 && donewhen == 0 {
		// conformance:inapplicable — a plan may legitimately name no test yet,
		// which is the normal state before implementation begins.
		t.Skip("no active plan has a Done-when table")
	}
}

// changeWindowBase is the commit this branch diverged from main.
//
// Through git(), which is FATAL, not Skip. That helper's own comment carries the
// reason — "a guard that reports nothing when it cannot run certifies nothing" —
// and the first version of this function swallowed every git error into a skip,
// conflating "on main" (genuinely inapplicable) with "git is broken" (the guard
// did not run). Only the first is reported here.
func changeWindowBase(t *testing.T) (string, bool) {
	t.Helper()
	base := strings.TrimSpace(string(git(t, "merge-base", "main", "HEAD")))
	head := strings.TrimSpace(string(git(t, "rev-parse", "HEAD")))
	return base, head != base
}

// changedLines returns the line numbers this window touched in path, or nil when
// it touched none. Uses the NEW-side hunk headers, which is what maps onto the
// file as it stands.
// Also through git(): returning nil on an error would be the same value as "this
// window did not touch the file", so a git failure would silently downgrade every
// row to unchecked — a check that cannot fail reading as green.
//
// ":/" makes the pathspec repo-root-relative. git() runs with cwd set to the
// PACKAGE directory, so a bare repo-relative path matches nothing and every row
// reads as untouched — the same trap repoRoot's comment records for `git
// ls-files`, and it made this guard silently skip on its first run.
func changedLines(t *testing.T, base, path string) map[int]bool {
	t.Helper()
	out := git(t, "diff", "--unified=0", base+"..HEAD", "--", ":/"+path)
	if len(out) == 0 {
		return nil
	}
	hunk := regexp.MustCompile(`(?m)^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)
	touched := map[int]bool{}
	for _, m := range hunk.FindAllStringSubmatch(string(out), -1) {
		start := atoiTest(t, m[1])
		n := 1
		if m[2] != "" {
			n = atoiTest(t, m[2])
		}
		for i := 0; i < n; i++ {
			touched[start+i] = true
		}
	}
	if len(touched) == 0 {
		return nil
	}
	return touched
}

func atoiTest(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("bad hunk number %q: %v", s, err)
	}
	return n
}

// checkPlanStatus holds one row's claim against the window.
func checkPlanStatus(t *testing.T, root, plan, name, path, status string, touched map[int]bool, checked *int) {
	t.Helper()
	recv, name := splitReceiver(name, path)
	src, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return
	}
	lo, hi, ok := declarationRegion(string(src), name, recv)
	if !ok {
		return // the sibling test owns "this symbol does not exist"
	}
	*checked++
	inWindow := false
	for ln := lo; ln <= hi; ln++ {
		if touched[ln] {
			inWindow = true
			break
		}
	}
	switch {
	case status == "unchanged" && inWindow:
		t.Errorf("%s calls %q unchanged, but this window edits its declaration "+
			"(%s:%d-%d). The status column is a claim about the DIFF, and a reader "+
			"trusts the plan to describe what changed.", plan, name, path, lo, hi)
	case status == "modified" && !inWindow:
		t.Errorf("%s calls %q modified, but this window does not touch its "+
			"declaration (%s:%d-%d) — the row describes work that did not happen.",
			plan, name, path, lo, hi)
	}
}

// declarationRegion locates a symbol's declaration INCLUDING its doc comment,
// which is part of what a reader means by "unchanged".
func declarationRegion(src, name, recv string) (lo, hi int, ok bool) {
	decl := declRegexp(name, recv)
	next := regexp.MustCompile(`^(func|type|var|const)\s`)
	lines := strings.Split(src, "\n")
	start := -1
	for i, l := range lines {
		if decl.MatchString(l) {
			start = i
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	lo = start
	for lo > 0 && strings.HasPrefix(strings.TrimSpace(lines[lo-1]), "//") {
		lo-- // the doc comment belongs to the symbol
	}
	hi = len(lines) - 1
	for i := start + 1; i < len(lines); i++ {
		if next.MatchString(lines[i]) {
			hi = i - 1
			// back off over the NEXT symbol's doc comment
			for hi > start && strings.HasPrefix(strings.TrimSpace(lines[hi]), "//") {
				hi--
			}
			break
		}
	}
	return lo + 1, hi + 1, true // 1-indexed, matching git
}

// planStatus is the third spelling of one parse, and the first two shipped
// broken — so this time every branch is entered by a fixture.
//
// `Contains(lower(cell), "new")` also matched "renewed". `Fields(cell)[0] ==
// "new"` then missed `**new**`, and bold cells are this repo's convention. The
// vocabulary version was written to end that, and the close review found the
// emphasis-stripping — the entire substance of the fix — was GREEN WHEN REMOVED,
// because no plan in the tree happens to write a bolded status today.
//
// The rule that generalises: the "observed red when the wiring is removed"
// discipline the plan applies to Done-when cells applies to EVERY fix delivered
// in answer to a finding. A finding-fix with no test that reddens without it is
// not addressed, however plausible the diff.
func TestPlanStatusNormalisesToTheVocabulary(t *testing.T) {
	for _, tc := range []struct {
		name, cell, want string
		ok               bool
	}{
		{"plain", "new", "new", true},
		{"padded", "  modified  ", "modified", true},
		{"trailing prose", "modified — gains `pron store.Lang`", "modified", true},
		{"an em-dash note after unchanged", "unchanged — D3", "unchanged", true},
		{"deleted", "deleted", "deleted", true},
		// The emphasis cells. No plan in the tree writes one today, which is
		// exactly why they must be fixtures: removing the Trim leaves every
		// other row here green.
		{"bold", "**new**", "new", true},
		{"bold with prose", "**modified** — keys on `EscapedPath`", "modified", true},
		{"italic", "*unchanged*", "unchanged", true},
		{"code-quoted", "`deleted`", "deleted", true},
		// And the loud-failure branch, likewise unreached by any real plan.
		{"a word that merely contains new", "renewed", "renewed", false},
		{"another", "newly added", "newly", false},
		{"empty", "   ", "", false},
		{"outside the vocabulary", "reused", "reused", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := planStatus(tc.cell)
			if got != tc.want || ok != tc.ok {
				t.Errorf("planStatus(%q) = %q, %v; want %q, %v", tc.cell, got, ok, tc.want, tc.ok)
			}
		})
	}
}

// A declaration this window REMOVED is either registered as retired or mentioned
// nowhere that reads as current truth.
//
// `retiredSymbolNames` is the right mechanism and its own comment says why it
// cannot be automatic: "only the person doing it knows the old name". True of
// the MAPPING — but not of the trigger. Whether a declaration disappeared in
// this window is a fact git holds, and the human half has now failed twice:
// `#27` renamed a doc-sync test and left a stale mention in `voice.go`, and `#31`
// merged two notation tests and deleted a sweep, leaving stale names in the
// atlas, its own plan, and `#30` — the live consumer that issue exists to
// unblock. Both times the guard stayed green over prose it was written to catch,
// because nobody added the row.
//
// So this is the trigger, and it asks for one of two things rather than
// guessing: register the rename, or leave no current-truth mention behind. A
// symbol that was moved rather than removed is not reported — the tree still
// declares it.
func TestARemovedDeclarationIsSweptOrRetired(t *testing.T) {
	root := repoRoot(t)
	base, ok := changeWindowBase(t)
	if !ok {
		// conformance:inapplicable — on main there is no window, so no removal
		// to sweep. git being broken is Fatal through git(), as everywhere here.
		t.Skip("no change window: HEAD is the merge-base with main")
	}

	// Removed top-level declarations, from the OLD side of the diff.
	removed := regexp.MustCompile(`(?m)^-func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)`)
	diff := string(git(t, "diff", base+"..HEAD", "--", ":/*.go"))
	var gone []string
	for _, m := range removed.FindAllStringSubmatch(diff, -1) {
		name := m[1]
		// Only names that cannot plausibly BE prose. `ids` is a real helper in
		// this package, and searching artifacts for it hits "for-bids"; the
		// failure message would then advise a retiredSymbolNames row that makes
		// the SIBLING guard permanently red on every file containing the word.
		// Exported and Test* names are the ones an artifact actually cites.
		if !isCitableName(name) {
			continue
		}
		// Still declared somewhere? Then it moved, which is not a removal.
		//
		// Scanned rather than `git grep`, which exits 1 for "no match" — a
		// perfectly ordinary answer that git() would treat as a failure and
		// Fatal on. Asking a tool a question whose negative answer is an error
		// code is how a guard ends up unable to run.
		if treeDeclares(t, root, name) {
			continue
		}
		if _, registered := retiredSymbolNames[name]; registered {
			continue // the sibling guard sweeps its mentions
		}
		if !slices.Contains(gone, name) {
			gone = append(gone, name)
		}
	}
	if len(gone) == 0 {
		// conformance:inapplicable — a window that removes no declaration has
		// nothing to sweep. Reachable on most windows, which is normal.
		t.Skip("this window removed no top-level declaration")
	}

	for _, name := range gone {
		for _, f := range currentTruthFiles(t, root) {
			b, err := os.ReadFile(filepath.Join(root, f))
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			// WORD-BOUNDARY, the rule the sibling guard already uses. Contains
			// was the first spelling and it is wrong the same way it was wrong
			// for the doc check two commits earlier — a substring hit is not a
			// mention, and this guard was written after that fix.
			if regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(currentTruthOnly(t, f, string(b))) {
				t.Errorf("%s names %q, which this window REMOVED and which is not in "+
					"retiredSymbolNames. Either add the row — the mapping is the part only "+
					"you know — or sweep the mention. A guard that depends on someone "+
					"remembering has now been remembered late twice.", f, name)
			}
		}
	}
}

// currentTruthFiles is the artifact set a stale name misleads a reader in:
// production Go, the README, the atlas, and active plans. Shared with
// TestNoArtifactNamesARetiredSymbol so the two guards cannot disagree about
// what "current truth" means.
func currentTruthFiles(t *testing.T, root string) []string {
	t.Helper()
	self := "cmd/define/repo_guard_test.go"
	binds := func(p string) bool {
		switch {
		case strings.HasSuffix(p, "_test.go"):
			return false
		case strings.HasSuffix(p, ".go"):
			return true
		case p == "README.md", strings.HasSuffix(p, "/README.md"), strings.HasPrefix(p, "atlas/"):
			return true
		case strings.HasPrefix(p, "workshop/plans/") && strings.HasSuffix(p, "-plan.md"):
			return true
		}
		return false
	}
	var out []string
	for _, f := range strings.Split(string(git(t, "-C", root, "ls-files", "-z")), "\x00") {
		p := filepath.ToSlash(f)
		if p == "" || p == self || !binds(p) {
			continue
		}
		if info, err := os.Stat(filepath.Join(root, f)); err != nil || info.IsDir() {
			continue
		}
		out = append(out, f)
	}
	// HERE, not in one caller. The inline copy this replaced carried the
	// assertion and the extracted helper did not, so the newer guard would have
	// passed over an empty file set if `binds` ever stopped matching. The two
	// copies had diverged before the second one was a day old.
	if len(out) == 0 {
		t.Fatal("no current-truth artifacts were examined; every guard over this set " +
			"would pass vacuously")
	}
	return out
}

// treeDeclares reports whether any Go file still declares name at top level.
func treeDeclares(t *testing.T, root, name string) bool {
	t.Helper()
	decl := regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s*)?` + regexp.QuoteMeta(name) + `\b`)
	for _, f := range strings.Split(string(git(t, "-C", root, "ls-files", "-z", "*.go")), "\x00") {
		if f == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue // a deleted-but-tracked path during a rewrite; not a declaration
		}
		if decl.Match(b) {
			return true
		}
	}
	return false
}

// isCitableName reports whether a removed declaration is one an artifact would
// actually name: a Test, a Fuzz target, an exported identifier, or an unexported
// camelCase one with an interior capital.
//
// THE LAST CLAUSE IS #42's, and it is the second finding in its family. This
// filter read "exported or Test*", on the reasoning that unexported helpers are
// not cited in prose. That is true of `ids` and `binds` and false of exactly the
// helpers this codebase argues about: `boardsFor` was deleted alongside `Recall`
// and stayed the current account of selection in six current-truth sites,
// including two atlas paragraphs — so the atlas held two contradictory accounts
// of the very rule the issue existed to change, and the guard written to catch
// that could not see it.
//
// The interior capital is what separates the two populations. A prose-cited
// unexported helper in this repo is a compound (`boardsFor`, `choiceFor`,
// `optionCandidates`); a single lowercase word (`ids`, `binds`, `paint`) is both
// uncited AND a substring of ordinary English, which is the noise the original
// clause was avoiding. So the noise is still excluded and the citations are not.
func isCitableName(name string) bool {
	if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz") {
		return true
	}
	if name == "" {
		return false
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return true
	}
	return strings.ContainsFunc(name[1:], func(r rune) bool { return r >= 'A' && r <= 'Z' })
}

// A DOC COMMENT'S FIRST WORD IS THE NAME OF THE DECLARATION IT SITS ON.
//
// Second finding in this family, so the deliverable is the check rather than the
// five edits. `go vet` does not look at this, and the exported-comment linters do
// not reach unexported declarations — which is most of this package. The failure
// is quiet and specific: a comment block written for one function acquires
// another when a declaration is inserted between them, so the new one arrives
// undocumented and the old one's prose now describes its neighbour. All five
// instances found at once were of exactly that shape, two of them inserted by the
// window under review.
//
// Named FUNCTIONS and SINGLE-SPEC const/var declarations. A method's receiver is
// not in the comment by convention, and a MULTI-spec `var`/`const` block's doc
// legitimately describes the group rather than any member — but a lone
// `const chromeGap = 1` is named by its own doc exactly as a function is.
//
// The const arm was added after this guard watched the failure it exists to catch
// go past it (#44): a new constant was inserted between `Paint`'s doc block and
// `Paint`, so `go doc` printed the constant with "Paint draws one whole frame…"
// and left `Paint` undocumented — invisible here, because the reparented block's
// new owner was not a FuncDecl. A guard whose scope is narrower than the failure
// mode is the shape of the thing it is guarding against.
func TestADocCommentNamesWhatItSitsOn(t *testing.T) {
	root := repoRoot(t)
	fset := token.NewFileSet()
	checked := 0
	for _, dir := range []string{"cmd/define", "cmd/define/play", "cmd/define/schedule", "cmd/define/store"} {
		pkgs, err := parser.ParseDir(fset, filepath.Join(root, dir), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", dir, err)
		}
		for _, pkg := range pkgs {
			for path, f := range pkg.Files {
				for _, d := range f.Decls {
					name, doc := documentedDecl(d)
					if name == "" || doc == nil {
						continue
					}
					// A TEST's doc legitimately opens with the name of what it
					// tests — "seedFor is GOLDEN…" over TestSeedForIsPinned is
					// the repo's convention and reads correctly. The failure
					// this catches is a PRODUCTION declaration wearing its
					// neighbour's prose.
					if strings.HasPrefix(name, "Test") ||
						strings.HasPrefix(name, "Fuzz") ||
						strings.HasPrefix(name, "Benchmark") {
						continue
					}
					first := strings.Fields(doc.Text())
					if len(first) == 0 {
						continue
					}
					checked++
					// The name, or the name with punctuation ("Keys names…",
					// "Grade:"), or a leading article the repo also writes
					// ("The frame is…" reads about the return value, not about
					// a neighbour). Only a first word that is ANOTHER
					// declaration's name is the failure this catches.
					word := strings.Trim(first[0], "`:,.—-")
					if word == name {
						continue
					}
					// SAME FILE only, because that is the shape this catches: a
					// declaration inserted between a comment and the function it
					// was written for. A first word naming something in another
					// file is prose — `newStoreCapturer`'s doc opens by naming
					// its `vocab` PARAMETER, which collides with a function two
					// files away and is perfectly correct.
					if !declaredIn(f)[word] {
						continue
					}
					t.Errorf("%s: the doc comment on %s opens with %q, which is another "+
						"declaration in this package — a comment block acquires the wrong "+
						"owner when a declaration is inserted between them, leaving one "+
						"undocumented and the other described by its neighbour's prose",
						filepath.Base(path), name, word)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no documented declarations parsed, so this guard checked nothing")
	}
}

// documentedDecl is the NAME a doc comment sits on, and the doc, for the two
// declaration shapes whose comment names one thing: a named function, and a
// single-spec const/var.
//
// A multi-spec `const (…)` block is excluded because its doc describes the GROUP
// — "The two prompt lines livePrompt returns…" over a pair of constants is
// correct prose that this guard must not fire on. A lone declaration has no group
// to describe, so its doc names it or names its neighbour.
func documentedDecl(d ast.Decl) (string, *ast.CommentGroup) {
	name := declName(d)
	if name == "" {
		return "", nil
	}
	switch d := d.(type) {
	case *ast.FuncDecl:
		return name, d.Doc
	case *ast.GenDecl:
		return name, d.Doc
	}
	return "", nil
}

// declName is the name a declaration would be documented UNDER, whether or not it
// currently has a doc.
//
// SEPARATE FROM THE DOC, and that separation is the finding rather than a tidy-up
// (#44 I1). `declaredIn` first derived its neighbour set from `documentedDecl`,
// which requires a doc — and the failure being caught is a declaration that has
// just LOST its doc to a neighbour inserted above it. So the very name the guard
// needed to recognise was the one name excluded, and the mutation still passed.
// The owner set and the neighbour set must be the same SHAPES, and only the owner
// set may require a doc.
func declName(d ast.Decl) string {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Name == nil {
			return ""
		}
		return d.Name.Name
	case *ast.GenDecl:
		// Single-spec only: a multi-spec `const (…)` block's doc describes the
		// GROUP, and firing on that would be wrong.
		if len(d.Specs) != 1 {
			return ""
		}
		vs, ok := d.Specs[0].(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 {
			return ""
		}
		return vs.Names[0].Name
	}
	return ""
}

// declaredIn is every declaration name in ONE file, so the guard can tell a
// neighbour's name from ordinary prose. Without it the check would fire on every
// comment that happens to open with an identifier-shaped word.
//
// DERIVED FROM declName, so the OWNER set and the NEIGHBOUR set are the
// same set of declaration shapes BY CONSTRUCTION. They were not: `documentedDecl`
// was widened to single-spec const/var and this was left at `FuncDecl`, so a doc
// block whose first word is a CONSTANT's name stayed invisible — insert a
// function between `// boardRefusal is the prompt row…` and `const boardRefusal`
// and the guard passes while `go doc` shows the function wearing the constant's
// prose. That is the same failure one mirror over, and widening one side of a
// two-sided rule is how it survived being fixed once.
func declaredIn(f *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, d := range f.Decls {
		if name := declName(d); name != "" {
			out[name] = true
		}
	}
	return out
}

// isCitableName's own fixture table (#42 BR-11).
//
// The 2ND finding in the `pin-that-cannot-fail` family, and the same shape as the
// one eight lines above `TestPlanStatusNormalisesToTheVocabulary`: the widening
// that answered BR-7 — letting the removed-declaration sweep see unexported
// compound names — was GREEN WHEN REVERTED. `boardsFor` is swept from the tree
// now, so no artifact in the repo exercises the new clause, and the fix that
// closed a class was itself unpinned.
//
// A FIXTURE TABLE is the answer for exactly that reason: the input the rule cares
// about is one this tree no longer contains, so it has to be supplied rather than
// found. The rows below are the two populations the clause separates — prose-cited
// compounds, and the single lowercase words that are substrings of ordinary
// English.
func TestIsCitableNameSeparatesCitedNamesFromNoise(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		// The population the widening exists for: unexported, compound, and cited
		// in prose. `boardsFor` is the one that went stale in two atlas paragraphs
		// while the guard could not see it.
		{"boardsFor", true},
		{"choiceFor", true},
		{"optionCandidates", true},
		{"todaysQuestions", true},
		// The population the original clause was right to exclude: a single
		// lowercase word is both uncited and a substring of ordinary English, so
		// sweeping for it reports noise rather than findings.
		{"ids", false},
		{"binds", false},
		{"paint", false},
		{"ask", false},
		// Unchanged by the widening: exported names and test entry points.
		{"Marks", true},
		{"TestSomething", true},
		{"FuzzSomething", true},
		// Degenerate.
		{"", false},
		{"a", false},
	} {
		if got := isCitableName(tc.name); got != tc.want {
			t.Errorf("isCitableName(%q) = %v, want %v — a name in the cited population that "+
				"this returns false for is a stale mention no guard will find, and a noise "+
				"word it returns true for is a sweep that reports substrings",
				tc.name, got, tc.want)
		}
	}
}

// currentTruthOnly exempts a `| deleted |` row's OWN symbol, and nothing else
// (#42 BR-11).
//
// The other half of the round-2 fix that was green when reverted: the exemption
// resolves a genuine conflict between two guards, and the tree exercises only the
// passing case. Its risk is over-reach — an exemption that quieted a guard about
// symbols the document never declared would silently widen every artifact's
// licence — so both directions are supplied here rather than hoped for.
func TestADeletedRowExemptsOnlyItsOwnSymbol(t *testing.T) {
	const doc = "| `boardsFor` | `x.go` | deleted |\n\n" +
		"Prose naming boardsFor, which the row above declares deleted.\n" +
		"Prose naming somethingElse, which it does not.\n"
	got := currentTruthOnly(t, "workshop/plans/fixture-plan.md", doc)

	if strings.Contains(got, "boardsFor") {
		t.Errorf("the declared symbol survived the filter, so the guard it exists to quiet "+
			"still fires on the document that recorded the deletion:\n%s", got)
	}
	if !strings.Contains(got, "somethingElse") {
		t.Errorf("an undeclared symbol was stripped too — the exemption must be limited to "+
			"what a `| deleted |` row NAMES, or a plan quiets every guard at once:\n%s", got)
	}
}
