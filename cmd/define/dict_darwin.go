//go:build darwin

package main

/*
#cgo LDFLAGS: -framework CoreServices -framework CoreFoundation
#include <CoreServices/CoreServices.h>
#include <stdlib.h>

// DCSCopyTextDefinition is declared by the DictionaryServices SDK header
// included above -- do NOT redeclare it here, that is a compile error.
//
// Only two functions are public in that header; structured markup is not among
// them, which is why the caller parses flat text.
// status: 0 = found, 1 = no entry, 2 = internal failure. Collapsing these into
// a bare NULL would report a genuine CoreFoundation failure as "no entry".
char *noad_lookup(const char *word, int *status) {
    *status = 2;
    CFStringRef s = CFStringCreateWithCString(NULL, word, kCFStringEncodingUTF8);
    if (!s) return NULL;
    CFRange r = CFRangeMake(0, CFStringGetLength(s));
    CFStringRef def = DCSCopyTextDefinition(NULL, s, r);
    CFRelease(s);
    if (!def) { *status = 1; return NULL; }
    CFIndex max = CFStringGetMaximumSizeForEncoding(CFStringGetLength(def), kCFStringEncodingUTF8) + 1;
    char *buf = malloc(max);
    if (buf && !CFStringGetCString(def, buf, max, kCFStringEncodingUTF8)) { free(buf); buf = NULL; }
    CFRelease(def);
    if (buf) *status = 0;
    return buf;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// ErrLookupFailed separates a CoreFoundation malfunction from a word the
// dictionary simply does not have.
var ErrLookupFailed = errors.New("dictionary lookup failed")

// noadDictionary reads the New Oxford American Dictionary bundled with macOS --
// the same dictionary Google licenses for its US definition panel, which is why
// the notation matches character-for-character.
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

func systemDictionary() Dictionary { return noadDictionary{} }
