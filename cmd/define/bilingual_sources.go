package main

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
