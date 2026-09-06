package store

// Test-only aliases, in an _test.go file so they are not production API.
//
// prune's rule is asserted from the package's EXTERNAL test, where every other
// invariant of this surface is asserted, and the first version exported a
// permanent symbol to get there. A test file alias reaches the same place
// without widening what consumers can call.
var PruneForTest = prune

// PerWordDirsForTest exposes the classification the Forget guard checks.
func PerWordDirsForTest(y *YAML) []string { return y.perWordDirs() }
