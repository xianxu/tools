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
// dcs_has_symbol lets a test check ONE name without going through the resolver,
// so the list below can be verified member by member rather than as a boolean.
static int dcs_has_symbol(const char *name) {
    return dlsym(RTLD_DEFAULT, name) != NULL;
}

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

// dcs_describe_all returns every installed dictionary as one flat record set:
//   <identifier>\t<index>>lang,<index>>lang,\n
//
// ONE copy of the set, described in a single pass. An earlier version called a
// per-index describe N times, each re-copying the set -- unsound by this file's
// own comment: CFSet enumeration order is UNSPECIFIED, so two copies may
// enumerate differently and the result gains duplicates and drops entries. A
// dropped Larousse degrades Spanish to the NULL search silently.
//
// A CFSet, NOT a CFArray. Measured 2026-08-28: CFArrayGetValueAtIndex on this
// result does not return garbage, it raises "-[__NSCFSet objectAtIndex:]:
// unrecognized selector" -- an uncaught ObjC exception in a cgo frame, where the
// cause is not remotely obvious. CFSetGetValues is the only correct read.
//
// A flat string rather than a nested structure because the Go side parses it
// once into typed values -- keeping the cgo boundary to plain C strings is what
// stops this file from growing a data model.
static char *dcs_describe_all(void) {
    if (!dcs_resolve()) return NULL;
    CFSetRef set = f_copy_available();
    if (!set) return NULL;

    CFIndex n = CFSetGetCount(set);
    const void **values = malloc(sizeof(void *) * n);
    if (!values) { CFRelease(set); return NULL; }
    CFSetGetValues(set, values);

    CFMutableStringRef acc = CFStringCreateMutable(NULL, 0);
    for (CFIndex i = 0; i < n; i++) {
        DCSRef d = (DCSRef)values[i];
        CFStringRef id = f_get_identifier(d);
        if (!id) continue;
        CFStringAppend(acc, id);
        CFStringAppend(acc, CFSTR("\t"));

        CFArrayRef langs = f_get_languages(d);
        if (langs) {
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
        }
        CFStringAppend(acc, CFSTR("\n"));
    }
    free(values);
    CFRelease(set);
    char *out = cfstring_dup(acc);
    CFRelease(acc);
    return out;
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

// dcsPrivateSymbols names every symbol dcs_resolve looks up.
//
// ONE producer for the list, because a COUNT written into prose is a
// restatement with nothing keeping it true — "the nine symbols" was repeated in
// four documents and was wrong in all four (nine is how many the issue's survey
// FOUND; this file resolves three). The docs now say "the private symbols" and
// the conformance test walks this slice, so there is no number to drift.
//
// DCSCopyTextDefinition is deliberately absent: it is the one PUBLIC call, comes
// from the SDK header, and is not at risk in the way these are.
var dcsPrivateSymbols = []string{
	"DCSCopyAvailableDictionaries",
	"DCSDictionaryGetIdentifier",
	"DCSDictionaryGetLanguages",
}

// hasPrivateSymbol reports whether one private symbol resolves.
func hasPrivateSymbol(name string) bool {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return C.dcs_has_symbol(cs) != 0
}

// installedDictionaries reads the private surface and returns what it says.
//
// nil means the surface is unavailable — every caller treats that as "fall back
// to the NULL search", never as "no dictionaries exist".
func installedDictionaries() []dictMeta {
	raw := C.dcs_describe_all()
	if raw == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(raw))
	return parseDictRecords(C.GoString(raw))
}

// parseDictRecords reads the flat encoding the C side emits.
//
// Pure, so the whole cgo boundary's format is testable without CoreServices —
// which matters because this is where a malformed record could silently drop the
// one dictionary a language depends on.
func parseDictRecords(s string) []dictMeta {
	out := []dictMeta{}
	for _, line := range strings.Split(s, "\n") {
		id, langs, ok := strings.Cut(line, "\t")
		if !ok || id == "" {
			continue
		}
		out = append(out, dictMeta{ID: id, Langs: parseLangPairs(langs)})
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

// selectedDictionary looks a word up in the chosen dictionaries FOR ONE
// LANGUAGE, in curated order, and reports the first entry.
//
// A list rather than one, because "the English dictionary" is not one book: NOAD
// answers ordinary words, Apple Dictionary answers iPhone. Both index en->en, so
// walking them cannot leak another language — which is the property that
// distinguishes this from the NULL search over every ACTIVE dictionary.
type selectedDictionary struct{ ids []string }

func (d selectedDictionary) Lookup(word string) (string, error) {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))

	// The FIRST non-absence error wins, not the last error seen. Status 3 on the
	// primary followed by status 1 on the next would otherwise report ErrNoEntry
	// — "this word does not exist in English" when the truth is "the primary
	// dictionary vanished". dcs_lookup_in keeps those two statuses distinct
	// precisely so this layer does not collapse them.
	var lastErr error = ErrNoEntry
	for _, id := range d.ids {
		cid := C.CString(id)
		var status C.int
		res := C.dcs_lookup_in(cid, cw, &status)
		C.free(unsafe.Pointer(cid))
		if res != nil {
			out := C.GoString(res)
			C.free(unsafe.Pointer(res))
			return out, nil
		}
		switch status {
		case 1:
			// Not in THIS dictionary. Keep going — the next one may have it —
			// and if none do, that absence is the answer the mode makes
			// possible: sycophantic has no Spanish entry, and saying so beats
			// answering from English.
			lastErr = ErrNoEntry
		case 3:
			// The dictionary itself is gone, which is NOT the same as the word
			// being absent. Kept, and NOT overwritten by a later absence.
			if errors.Is(lastErr, ErrNoEntry) {
				lastErr = fmt.Errorf("%w: dictionary %s is unavailable", ErrLookupFailed, id)
			}
		default:
			if errors.Is(lastErr, ErrNoEntry) {
				lastErr = ErrLookupFailed
			}
		}
	}
	return "", lastErr
}

// noadDictionary is the pre-#23 path: DCSCopyTextDefinition with a NULL
// dictionary, meaning "search every ACTIVE dictionary" — not NOAD specifically.
//
// It stays because it is the FALLBACK. The symbols this file resolves are
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
// A thin IO shell over dictionaryFor: read the metadata, apply the policy, build
// the seam. The policy is pure and lives in dictselect.go, which is what lets
// its three outcomes be tested on a machine with no dictionaries at all.
func systemDictionary(lang store.Lang, warn io.Writer) (Dictionary, string) {
	ids, name, complaint := dictionaryFor(installedDictionaries(), lang)
	if complaint != "" {
		warnTo(warn, "%s", complaint)
	}
	if len(ids) == 0 {
		return noadDictionary{}, name
	}
	return selectedDictionary{ids: ids}, name
}
