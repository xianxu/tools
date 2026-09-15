//go:build !darwin

package main

type unavailableSpanishEnglishSource struct{}

func newSpanishEnglishSource() recordSource { return unavailableSpanishEnglishSource{} }
func (unavailableSpanishEnglishSource) Records(string) ([]bilingualRecord, error) {
	return nil, ErrBilingualUnsupported
}
