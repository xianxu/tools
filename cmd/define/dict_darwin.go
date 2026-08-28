//go:build darwin

package main

/*
#cgo LDFLAGS: -framework CoreServices -framework CoreFoundation
#include <CoreServices/CoreServices.h>
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>

// DCSCopyTextDefinition is declared by the DictionaryServices SDK header
// included above -- do NOT redeclare it here, that is a compile error.
//
// Everything else this file needs is PRIVATE: absent from the header, resolved
// at run time with dlsym, and free to vanish on an OS update. That is the whole
// reason for the shape below -- a missing symbol degrades to the NULL search
// this tool did before #23, rather than failing to link or crashing.
typedef void *DCSRef;
typedef CFSetRef (*copy_available_fn)(void);
typedef CFStringRef (*get_identifier_fn)(DCSRef);
typedef CFArrayRef (*get_languages_fn)(DCSRef);

static copy_available_fn f_copy_available;
static get_identifier_fn f_get_identifier;
static get_languages_fn f_get_languages;
static int resolved; // 0 = not tried, 1 = all present, -1 = something missing

// dcs_resolve loads the private surface once. Returns 1 if every symbol needed
// to SELECT a dictionary is present.
//
// RTLD_DEFAULT rather than dlopen: CoreServices is already linked in by the
// #cgo LDFLAGS above, so the symbols are in the process image if they exist at
// all. dlopening the framework again would be a second handle to the same code.
static int dcs_resolve(void) {
    if (resolved) return resolved == 1;
    f_copy_available = (copy_available_fn)dlsym(RTLD_DEFAULT, "DCSCopyAvailableDictionaries");
    f_get_identifier = (get_identifier_fn)dlsym(RTLD_DEFAULT, "DCSDictionaryGetIdentifier");
    f_get_languages  = (get_languages_fn)dlsym(RTLD_DEFAULT, "DCSDictionaryGetLanguages");
    resolved = (f_copy_available && f_get_identifier && f_get_languages) ? 1 : -1;
    return resolved == 1;
}

static char *cfstring_dup(CFStringRef s) {
    if (!s) return NULL;
    CFIndex max = CFStringGetMaximumSizeForEncoding(CFStringGetLength(s), kCFStringEncodingUTF8) + 1;
    char *buf = malloc(max);
    if (buf && !CFStringGetCString(s, buf, max, kCFStringEncodingUTF8)) { free(buf); buf = NULL; }
    return buf;
}

// dcs_count returns how many dictionaries are installed, or -1 when the private
// surface is unavailable.
static CFIndex dcs_count(void) {
    if (!dcs_resolve()) return -1;
    CFSetRef set = f_copy_available();
    if (!set) return -1;
    CFIndex n = CFSetGetCount(set);
    CFRelease(set);
    return n;
}

// dcs_describe fills identifier + language pairs for the i'th dictionary.
//
// A CFSet, NOT a CFArray. Measured 2026-08-28: CFArrayGetValueAtIndex on this
// result does not return garbage, it raises "-[__NSCFSet objectAtIndex:]:
// unrecognized selector" -- an uncaught ObjC exception in a cgo frame, where the
// cause is not remotely obvious. CFSetGetValues is the only correct read.
//
// Iteration order is therefore UNSPECIFIED, which is why selection is by
// identifier and never by position or name (chooseDictionary is pinned
// order-independent for the same reason).
static char *dcs_describe(CFIndex want, char **langs_out) {
    *langs_out = NULL;
    if (!dcs_resolve()) return NULL;
    CFSetRef set = f_copy_available();
    if (!set) return NULL;

    CFIndex n = CFSetGetCount(set);
    char *id_out = NULL;
    if (want >= 0 && want < n) {
        const void **values = malloc(sizeof(void *) * n);
        if (values) {
            CFSetGetValues(set, values);
            DCSRef d = (DCSRef)values[want];
            id_out = cfstring_dup(f_get_identifier(d));

            // Language pairs, flattened to "index>description," repeated. A flat
            // string rather than a nested structure because the Go side parses
            // it once into typed values -- keeping the cgo boundary to plain
            // C strings is what stops this file from growing a data model.
            CFArrayRef langs = f_get_languages(d);
            if (langs) {
                CFMutableStringRef acc = CFStringCreateMutable(NULL, 0);
                CFIndex ln = CFArrayGetCount(langs);
                for (CFIndex j = 0; j < ln; j++) {
                    CFDictionaryRef pair = (CFDictionaryRef)CFArrayGetValueAtIndex(langs, j);
                    if (!pair) continue;
                    CFStringRef idx = (CFStringRef)CFDictionaryGetValue(pair, CFSTR("DCSDictionaryIndexLanguage"));
                    CFStringRef des = (CFStringRef)CFDictionaryGetValue(pair, CFSTR("DCSDictionaryDescriptionLanguage"));
                    if (!idx || !des) continue;
                    CFStringAppend(acc, idx);
                    CFStringAppend(acc, CFSTR(">"));
                    CFStringAppend(acc, des);
                    CFStringAppend(acc, CFSTR(","));
                }
                *langs_out = cfstring_dup(acc);
                CFRelease(acc);
            }
            free(values);
        }
    }
    CFRelease(set);
    return id_out;
}

// dcs_lookup_in searches ONE dictionary, found by identifier.
//
// status: 0 = found, 1 = no entry, 2 = internal failure, 3 = no such dictionary.
// 1 and 3 are different answers and must not collapse: "this word is not Spanish"
// is a correct result, while "the Spanish dictionary is not installed" means the
// caller should fall back rather than report an absence it cannot vouch for.
static char *dcs_lookup_in(const char *identifier, const char *word, int *status) {
    *status = 2;
    if (!dcs_resolve()) { *status = 3; return NULL; }

    CFSetRef set = f_copy_available();
    if (!set) { *status = 3; return NULL; }
    CFIndex n = CFSetGetCount(set);
    const void **values = malloc(sizeof(void *) * n);
    if (!values) { CFRelease(set); return NULL; }
    CFSetGetValues(set, values);

    DCSRef found = NULL;
    for (CFIndex i = 0; i < n && !found; i++) {
        char *id = cfstring_dup(f_get_identifier((DCSRef)values[i]));
        if (id && strcmp(id, identifier) == 0) found = (DCSRef)values[i];
        free(id);
    }
    free(values);
    if (!found) { CFRelease(set); *status = 3; return NULL; }

    CFStringRef s = CFStringCreateWithCString(NULL, word, kCFStringEncodingUTF8);
    if (!s) { CFRelease(set); return NULL; }
    CFRange r = CFRangeMake(0, CFStringGetLength(s));
    CFStringRef def = DCSCopyTextDefinition(found, s, r);
    CFRelease(s);
    CFRelease(set);
    if (!def) { *status = 1; return NULL; }
    char *buf = cfstring_dup(def);
    CFRelease(def);
    if (buf) *status = 0;
    return buf;
}

// noad_lookup is the pre-#23 path: NULL means "search every ACTIVE dictionary".
// Kept verbatim because it is the fallback -- when the private surface is gone,
// this is exactly what the tool did before, which is a working English
// dictionary rather than a crash.
char *noad_lookup(const char *word, int *status) {
    *status = 2;
    CFStringRef s = CFStringCreateWithCString(NULL, word, kCFStringEncodingUTF8);
    if (!s) return NULL;
    CFRange r = CFRangeMake(0, CFStringGetLength(s));
    CFStringRef def = DCSCopyTextDefinition(NULL, s, r);
    CFRelease(s);
    if (!def) { *status = 1; return NULL; }
    char *buf = cfstring_dup(def);
    CFRelease(def);
    if (buf) *status = 0;
    return buf;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unsafe"

	"github.com/xianxu/tools/cmd/define/store"
)

// ErrLookupFailed separates a CoreFoundation malfunction from a word the
// dictionary simply does not have.
var ErrLookupFailed = errors.New("dictionary lookup failed")

// installedDictionaries reads the private surface and returns what it says.
//
// nil means the surface is unavailable — every caller treats that as "fall back
// to the NULL search", never as "no dictionaries exist".
func installedDictionaries() []dictMeta {
	n := C.dcs_count()
	if n < 0 {
		return nil
	}
	out := make([]dictMeta, 0, int(n))
	for i := C.CFIndex(0); i < n; i++ {
		var langs *C.char
		id := C.dcs_describe(i, &langs)
		if id == nil {
			continue
		}
		m := dictMeta{ID: C.GoString(id)}
		C.free(unsafe.Pointer(id))
		if langs != nil {
			m.Langs = parseLangPairs(C.GoString(langs))
			C.free(unsafe.Pointer(langs))
		}
		out = append(out, m)
	}
	return out
}

// parseLangPairs reads the flat "index>description," encoding the C side emits.
//
// Pure, so the encoding is testable without CoreServices — which matters more
// than it sounds: this is the one place a malformed pair could silently make a
// bilingual dictionary look monolingual.
func parseLangPairs(s string) []langPair {
	var out []langPair
	for _, field := range strings.Split(s, ",") {
		idx, desc, ok := strings.Cut(field, ">")
		if !ok {
			continue
		}
		// Apple writes both "en" and "en_US"; the region is not a language.
		i, err := store.ParseLang(baseLang(idx))
		if err != nil {
			continue
		}
		d, err := store.ParseLang(baseLang(desc))
		if err != nil {
			continue
		}
		out = append(out, langPair{Index: i, Description: d})
	}
	return out
}

func baseLang(s string) string {
	if i := strings.IndexAny(s, "_-"); i >= 0 {
		return s[:i]
	}
	return s
}

// selectedDictionary looks a word up in ONE chosen dictionary.
type selectedDictionary struct{ id string }

func (d selectedDictionary) Lookup(word string) (string, error) {
	cid, cw := C.CString(d.id), C.CString(word)
	defer C.free(unsafe.Pointer(cid))
	defer C.free(unsafe.Pointer(cw))
	var status C.int
	res := C.dcs_lookup_in(cid, cw, &status)
	if res == nil {
		switch status {
		case 1:
			// The word is not in THIS language. A real answer, and the one the
			// mode makes possible: sycophantic has no Spanish entry, and saying
			// so beats answering from English.
			return "", ErrNoEntry
		case 3:
			return "", fmt.Errorf("%w: dictionary %s is unavailable", ErrLookupFailed, d.id)
		}
		return "", ErrLookupFailed
	}
	defer C.free(unsafe.Pointer(res))
	return C.GoString(res), nil
}

// noadDictionary is the pre-#23 path: DCSCopyTextDefinition with a NULL
// dictionary, meaning "search every ACTIVE dictionary" — not NOAD specifically.
//
// It stays because it is the FALLBACK. The nine symbols this file resolves are
// private and undocumented; when one disappears on an OS update the tool
// degrades to exactly what it shipped before #23 — a working English dictionary
// — rather than to a crash or a link failure.
type noadDictionary struct{}

func (noadDictionary) Lookup(word string) (string, error) {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))
	var status C.int
	res := C.noad_lookup(cw, &status)
	if res == nil {
		if status == 1 {
			return "", ErrNoEntry
		}
		return "", ErrLookupFailed
	}
	defer C.free(unsafe.Pointer(res))
	return C.GoString(res), nil
}

// systemDictionary picks the dictionary for a language.
//
// The three outcomes, and each is a deliberate answer rather than a default:
//
//   - a curated, installed, monolingual dictionary — use it, and SAY SO, because
//     on a machine with a different set installed a wrong pick should be visible
//     rather than puzzling.
//   - the private surface is gone (an OS update) — fall back to the NULL search,
//     which is what the tool did before #23. Degrading to the previous version's
//     behaviour is the point of resolving these symbols at run time at all.
//   - the surface works but nothing curated matches the language — ALSO the NULL
//     search, and say why. Guessing among a thesaurus, an accessibility
//     dictionary and a general one is how a deterministic tiebreak picks wrong.
func systemDictionary(lang store.Lang, warn io.Writer) (Dictionary, string) {
	installed := installedDictionaries()
	if installed == nil {
		// LOUD, because it is the surprising one: the private surface moved under
		// us, and the tool has silently become its pre-#23 self.
		warnf(warn, "the dictionary-selection API is unavailable; searching every active dictionary")
		return noadDictionary{}, everyActiveDictionary
	}
	m, ok := chooseDictionary(installed, lang)
	if !ok {
		warnf(warn, "no known %s dictionary is installed; searching every active dictionary", lang)
		return noadDictionary{}, everyActiveDictionary
	}
	// SILENT on the happy path, and the name is reported by /lang instead. A line
	// per lookup saying the expected thing happened is noise; a learner on a
	// machine with a different set installed asks the question once, and /lang is
	// where the answer belongs.
	return selectedDictionary{id: m.ID}, m.ID
}

// everyActiveDictionary is what the NULL search is called when /lang reports it.
// Not an identifier, because it is not one dictionary — it is the host's whole
// active set, which is why results depend on Dictionary.app's configuration.
const everyActiveDictionary = "every active dictionary"

func warnf(w io.Writer, format string, args ...any) {
	if w != nil {
		fmt.Fprintf(w, "define: "+format+"\n", args...)
	}
}
