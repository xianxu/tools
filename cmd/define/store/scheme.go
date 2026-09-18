package store

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Scheme names the terminal's background. A closed set: anything else read from
// a file, a flag or a command is refused at the boundary (ARCH-SECURE), so every
// value past ParseScheme is one of these two.
type Scheme string

const (
	SchemeDark  Scheme = "dark"
	SchemeLight Scheme = "light"
)

// ParseScheme accepts either value, trimmed and in any case.
func ParseScheme(s string) (Scheme, error) {
	switch v := Scheme(strings.ToLower(strings.TrimSpace(s))); v {
	case SchemeDark, SchemeLight:
		return v, nil
	}
	return "", fmt.Errorf("%q is not a colour scheme; use dark or light", s)
}

const schemeFileName = "scheme"
const maxSchemeSettingBytes = 64

// ReadScheme reads the saved scheme in dir (define's USER config directory, not a
// deck). found is false with a nil error when nothing is saved. Anything
// unreadable, oversized or outside the enum is an ERROR naming the file, so the
// caller can warn once and treat it as unset (ARCH-SECURE: a hand-edited or
// truncated file is untrusted input, parsed into the closed enum here).
func ReadScheme(dir string) (Scheme, bool, error) {
	path := filepath.Join(dir, schemeFileName)
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxSchemeSettingBytes+1))
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", path, err)
	}
	if len(b) > maxSchemeSettingBytes {
		return "", false, fmt.Errorf("%s is larger than a scheme name", path)
	}
	s, err := ParseScheme(string(b))
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", path, err)
	}
	return s, true, nil
}

// WriteScheme saves s atomically, creating dir if needed.
func WriteScheme(dir string, s Scheme) error {
	return writeBytesAtomic(filepath.Join(dir, schemeFileName), []byte(string(s)+"\n"))
}

// ClearScheme forgets the saved scheme, then the directory if that left it
// empty (ARCH-FUNERAL: the residue is at most this file and its directory).
//
// It removes only what is OURS to remove. The file is the saved choice itself,
// link or not, so forgetting the choice removes it. The directory goes only if
// it is a REAL directory that is now empty: os.Remove unlinks a symlink even
// when its target is full, which would break a dotfile manager's link (stow,
// chezmoi) — so Lstat first, and a non-empty real directory refuses on its own.
func ClearScheme(dir string) error {
	if err := os.Remove(filepath.Join(dir, schemeFileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if fi, err := os.Lstat(dir); err == nil && fi.IsDir() {
		_ = os.Remove(dir) // ENOTEMPTY when something else lives there: kept
	}
	return nil
}
