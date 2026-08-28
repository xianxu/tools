// Package impure exists to be REJECTED. It is what a purity guard must catch.
package impure

import (
	"os"

	"github.com/xianxu/tools/cmd/define/store"
)

// Reaches the disk twice over: an os import, and store's YAML constructor —
// which the import allowlist alone would miss, because store is allowed.
func Bad() (string, store.Store) {
	return os.Getenv("HOME"), store.NewYAML("/tmp", store.DefaultLang, nil)
}
