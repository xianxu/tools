package main

import (
	"errors"
	"fmt"
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
)

type definitionSection struct {
	label        string
	language     store.Lang
	entries      []string
	documents    []bilingualDocument
	presentation store.Lang
	formatErr    error
	err          error
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
	first := definitionSection{entries: []string{primary}, language: dictionarySourceLanguage(dict), err: primaryErr}
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

func renderDefinitionOutput(set definitionSet, opt RenderOpts) renderedOutput {
	var out strings.Builder
	var regions []Region
	var paints []rowPaint
	for index, section := range set.sections {
		startLine := strings.Count(out.String(), "\n")
		role := section.language
		if section.presentation != "" {
			role = section.presentation
		}
		if section.formatErr != nil {
			role = ""
		}
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
			}
			entry := ParseEntry(raw)
			var rendered string
			var rs []Region
			if entryIndex < len(section.documents) && section.documents[entryIndex].root != nil {
				rendered, rs = renderBilingualDocument(section.documents[entryIndex], ro)
			} else if section.formatErr != nil {
				rendered = wrapWritten(raw, opt.Width) + "\n"
			} else {
				rendered, rs = Render(entry, ro)
			}
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
		if section.formatErr != nil {
			out.WriteString("[Oxford formatting unavailable; showing source text]\n")
		}
		endLine := strings.Count(out.String(), "\n")
		for len(paints) < endLine {
			paints = append(paints, rowPaint{})
		}
		if opt.Color && role != "" && normalizedLang(role) == normalizedLang(opt.Tint.lang) {
			for i := startLine; i < endLine; i++ {
				paints[i].tinted = opt.Tint.on
			}
		}
	}
	return layoutOutput(renderedOutput{text: out.String(), regions: regions, rows: paints}, opt.Width)
}

type spanishDefinitions struct {
	Dictionary
	english recordSource
}

func (d spanishDefinitions) primaryLabel() string { return "Spanish — Larousse Diccionario General" }
func (d spanishDefinitions) supplement(word, primary string) definitionSection {
	section := definitionSection{label: "English — Oxford Spanish–English", presentation: "en"}
	records, err := d.english.Records(word)
	if err == nil {
		var selected []bilingualRecord
		selected, err = selectedSpanishRecords(records, word, entryIdentity(ParseEntry(primary)))
		for _, record := range selected {
			section.entries = append(section.entries, record.Text)
			doc, parseErr := parseBilingualDocument(record)
			section.documents = append(section.documents, doc)
			if parseErr != nil {
				section.formatErr = parseErr
			}
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

func (d spanishDefinitions) primarySourceLanguage() store.Lang {
	return dictionarySourceLanguage(d.Dictionary)
}
func (d untranslatedDefinitions) primarySourceLanguage() store.Lang {
	return dictionarySourceLanguage(d.Dictionary)
}
