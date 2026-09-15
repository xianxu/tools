package main

import (
	"errors"
	"fmt"
	"strings"
)

type definitionSection struct {
	label   string
	entries []string
	source  []languageText
	err     error
}
type definitionSet struct {
	sections []definitionSection
	err      error
	labeled  bool
}

type supplementalDictionary interface {
	supplement(word, primary string) definitionSection
	primaryLabel() string
}

// definitionsFor never re-fetches the primary entry. Raw and disabled callers
// keep the original single-source contract, including error identity.
func definitionsFor(dict Dictionary, word, primary string, primaryErr error, on bool) definitionSet {
	first := definitionSection{entries: []string{primary}, err: primaryErr}
	set := definitionSet{sections: []definitionSection{first}, err: primaryErr}
	provider, ok := dict.(supplementalDictionary)
	if !on || !ok {
		return set
	}
	set.labeled = true
	set.sections[0].label = provider.primaryLabel()
	second := provider.supplement(word, primary)
	set.sections = append(set.sections, second)
	if second.err == nil && len(second.entries) > 0 {
		set.err = nil
	} else if primaryErr != nil {
		set.err = errors.Join(primaryErr, second.err)
	}
	return set
}

// renderDefinitions preserves per-section language ownership of word actions.
// Regions are complete here, so callers must not run an unscoped vocabulary
// pass over the composed bilingual string afterward.
func renderDefinitions(set definitionSet, opt RenderOpts) (string, []Region) {
	var out strings.Builder
	var regions []Region
	for index, section := range set.sections {
		if set.labeled {
			if index > 0 {
				out.WriteString("\n")
			}
			p := newPalette(opt.Color)
			out.WriteString(p.sect + wrapText(section.label, opt.Width, 0) + p.off + "\n")
		}
		if section.err != nil {
			if set.labeled {
				out.WriteString(wrapText(section.err.Error(), opt.Width, 0) + "\n")
			}
			continue
		}
		for entryIndex, raw := range section.entries {
			ro := opt
			if index > 0 {
				ro.Vocab = nil
				ro.Language = ""
			}
			entry := ParseEntry(raw)
			if entryIndex < len(section.source) && section.source[entryIndex].text == raw {
				entry.source = section.source[entryIndex]
			}
			rendered, rs := Render(entry, ro)
			if index == 0 {
				rs = mergeRegions(rs, wordRegions(rendered, opt.Vocab))
			}
			offset := strings.Count(out.String(), "\n")
			for _, r := range rs {
				r.Line += offset
				regions = append(regions, r)
			}
			out.WriteString(rendered)
		}
	}
	return out.String(), regions
}

type spanishDefinitions struct {
	Dictionary
	english recordSource
}

func (d spanishDefinitions) primaryLabel() string { return "Spanish — Larousse Diccionario General" }
func (d spanishDefinitions) supplement(word, primary string) definitionSection {
	section := definitionSection{label: "English — Oxford Spanish–English"}
	records, err := d.english.Records(word)
	if err == nil {
		var selected []bilingualRecord
		selected, err = selectedSpanishRecords(records, word, entryIdentity(ParseEntry(primary)))
		for _, record := range selected {
			section.entries = append(section.entries, record.Text)
			section.source = append(section.source, bilingualLanguageText(record))
		}
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrBilingualUnavailable):
			err = fmt.Errorf("English dictionary unavailable: enable Spanish–English (Oxford) in Dictionary → Settings and wait for the download")
		case errors.Is(err, ErrNoEntry):
			err = fmt.Errorf("no English explanation for %q in Oxford Spanish–English", word)
		default:
			err = fmt.Errorf("English explanation unavailable: %w", err)
		}
	}
	section.err = err
	return section
}

type unavailableDictionary struct{ err error }

func (d unavailableDictionary) Lookup(string) (string, error) { return "", d.err }

// wordRegionsOutside adds actions only outside a render whose complete regions
// were supplied by its owner. This preserves section languages through reveals.
func wordRegionsOutside(text, already string, v Vocabulary) []Region {
	rs := wordRegions(text, v)
	if already == "" {
		return rs
	}
	at := strings.Index(text, already)
	if at < 0 {
		return rs
	}
	start := strings.Count(text[:at], "\n")
	end := start + strings.Count(already, "\n")
	startCol := visibleCells(text[strings.LastIndex(text[:at], "\n")+1 : at])
	endAt := at + len(already)
	endCol := visibleCells(text[strings.LastIndex(text[:endAt], "\n")+1 : endAt])
	out := rs[:0]
	for _, r := range rs {
		before := r.Line < start || (r.Line == start && r.Col+r.Width <= startCol)
		after := r.Line > end || (r.Line == end && r.Col >= endCol)
		if before || after {
			out = append(out, r)
		}
	}
	return out
}

type untranslatedDefinitions struct {
	Dictionary
	language string
}

func (d untranslatedDefinitions) primaryLabel() string { return d.language }
func (d untranslatedDefinitions) supplement(string, string) definitionSection {
	return definitionSection{label: "English", err: fmt.Errorf("English explanations are not supported for %s yet", d.language)}
}
