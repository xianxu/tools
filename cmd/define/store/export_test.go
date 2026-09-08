package store

// Test-only aliases, in an _test.go file so they are not production API.
//
// prune's rule is asserted from the package's EXTERNAL test, where every other
// invariant of this surface is asserted, and the first version exported a
// permanent symbol to get there. A test file alias reaches the same place
// without widening what consumers can call.
var PruneForTest = prune

// PerWordDirsForTest exposes the classification the Forget guard checks, on ALL
// THREE axes: the directory's path, whether it is language-scoped, and whether a
// word owns SEVERAL files there.
//
// The third arrived with #46 and the helper's doc said "both axes" for a while
// after — which is the same drift the axes themselves exist to catch. A guard
// that reports fewer axes than the classification carries is a guard that cannot
// see the one it is missing.
func PerWordDirsForTest(y *YAML) []PerWordDirView {
	var out []PerWordDirView
	for _, d := range y.perWordDirs() {
		out = append(out, PerWordDirView{Path: d.path, Scoped: d.scoped, Many: d.many})
	}
	return out
}

// PerWordDirView is one directory's classification, as the guard sees it.
type PerWordDirView struct {
	Path   string
	Scoped bool
	Many   bool
}

// DigestForTest exposes the key's digest so a test can name the files a word
// owns — the only way to damage one, which is what the degrade rows do.
func (k AudioKey) DigestForTest() string { return k.Digest }
