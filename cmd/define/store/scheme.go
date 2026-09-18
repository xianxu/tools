package store

import (
	"fmt"
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
