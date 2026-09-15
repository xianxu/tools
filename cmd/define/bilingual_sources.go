package main

import "fmt"

// spanishDictionaryFromInstalled is the portable factory used by the native
// shell. makePrimary constructs the selected-ID adapter without fetching a word;
// english owns its separate record API and remains usable independently.
func spanishDictionaryFromInstalled(installed []dictMeta, makePrimary func([]string) Dictionary, english recordSource) (Dictionary, string) {
	ids, name := spanishDictionarySources(installed)
	var primary Dictionary = unavailableDictionary{fmt.Errorf("Spanish dictionary unavailable: enable Spanish (Larousse Diccionario General) in Dictionary → Settings and wait for the download")}
	if installed == nil {
		primary = unavailableDictionary{fmt.Errorf("Spanish dictionary availability unknown: dictionary-selection API unavailable: %w", ErrLookupFailed)}
	} else if len(ids) > 0 {
		primary = makePrimary(ids)
	}
	return spanishDefinitions{Dictionary: primary, english: english}, name
}

// spanishDictionarySources derives both construction and its displayed status
// from the same installed set. A missing metadata API means unknown availability,
// not evidence that a dictionary is absent. Oxford's misleading language pairs
// do not prevent recognizing its installed identifier.
func spanishDictionarySources(installed []dictMeta) (ids []string, name string) {
	primaryName, englishName := "Larousse Diccionario General", "Oxford Spanish–English"
	if installed == nil {
		return nil, primaryName + " (availability unknown); English: " + englishName + " (availability unknown)"
	}
	chosen, _ := chooseDictionary(installed, "es")
	for _, dictionary := range chosen {
		ids = append(ids, dictionary.ID)
	}
	if len(ids) == 0 {
		primaryName += " (unavailable)"
	}
	englishInstalled := false
	for _, dictionary := range installed {
		if dictionary.ID == spanishEnglishDictionaryID {
			englishInstalled = true
			break
		}
	}
	if !englishInstalled {
		englishName += " (unavailable)"
	}
	return ids, primaryName + "; English: " + englishName
}
