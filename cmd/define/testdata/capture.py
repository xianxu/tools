#!/usr/bin/env python3
"""Print a word's dictionary entry as flat text.

Calls CoreServices' DCSCopyTextDefinition through ctypes -- deliberately NOT
through the Go binary this corpus is captured for, so the fixtures can exist
before any Go code does. The same call is made from cgo in dict_darwin.go; the
duplication is one-directional and intentional.

MUST run outside a sandbox: DCSCopyTextDefinition needs real access to
/System/Library/AssetsV2 and returns nothing (not an error) without it.
"""
import ctypes
import ctypes.util
import sys

UTF8 = 0x08000100

_cf = ctypes.cdll.LoadLibrary(ctypes.util.find_library("CoreFoundation"))
_cs = ctypes.cdll.LoadLibrary(
    "/System/Library/Frameworks/CoreServices.framework/CoreServices"
)


class _CFRange(ctypes.Structure):
    _fields_ = [("location", ctypes.c_long), ("length", ctypes.c_long)]


_cf.CFStringCreateWithCString.restype = ctypes.c_void_p
_cf.CFStringCreateWithCString.argtypes = [ctypes.c_void_p, ctypes.c_char_p, ctypes.c_uint32]
_cf.CFStringGetLength.restype = ctypes.c_long
_cf.CFStringGetLength.argtypes = [ctypes.c_void_p]
_cf.CFStringGetCString.argtypes = [ctypes.c_void_p, ctypes.c_char_p, ctypes.c_long, ctypes.c_uint32]
_cf.CFRelease.argtypes = [ctypes.c_void_p]
_cf.CFRetain.restype = ctypes.c_void_p
_cf.CFRetain.argtypes = [ctypes.c_void_p]
_cs.DCSCopyTextDefinition.restype = ctypes.c_void_p
_cs.DCSCopyTextDefinition.argtypes = [ctypes.c_void_p, ctypes.c_void_p, _CFRange]

# The PRIVATE surface. Undocumented and absent from the SDK header, which
# declares only two functions and says of the dictionary argument "not supported
# for Leopard. You should always pass NULL." That is true of the HEADER and false
# of the framework: these resolve, and the refs they return are accepted by
# DCSCopyTextDefinition. Measured 2026-08-28 — 87 dictionaries on this machine.
_cs.DCSCopyAvailableDictionaries.restype = ctypes.c_void_p
_cs.DCSCopyAvailableDictionaries.argtypes = []
_cs.DCSDictionaryGetIdentifier.restype = ctypes.c_void_p
_cs.DCSDictionaryGetIdentifier.argtypes = [ctypes.c_void_p]

# A SET, not an array. Measured: treating the result as a CFArray does not return
# garbage, it CRASHES with "-[__NSCFSet objectAtIndex:]: unrecognized selector".
_cf.CFSetGetCount.restype = ctypes.c_long
_cf.CFSetGetCount.argtypes = [ctypes.c_void_p]
_cf.CFSetGetValues.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.c_void_p)]


def _cfstr(ref):
    """Decode a CFStringRef we do not own."""
    if not ref:
        return None
    n = _cf.CFStringGetLength(ref)
    buf = ctypes.create_string_buffer((n + 1) * 4)
    if not _cf.CFStringGetCString(ref, buf, len(buf), UTF8):
        return None
    return buf.value.decode()


def dictionary_by_id(identifier):
    """Find an installed dictionary by its reverse-DNS identifier.

    Matched on the IDENTIFIER, never on the display name: the name is localised
    and ambiguous ("Espa" matches both Larousse and the bilingual Oxford), while
    DCSDictionaryGetIdentifier is stable.

    RETAINS before returning. The ref is borrowed from the copied set, and an
    earlier version released that set in a `finally` and then handed the ref to
    DCSCopyTextDefinition -- a use-after-release that only worked because
    CoreServices happens to keep the dictionaries alive. dict_darwin.go gets the
    same sequence right by doing the lookup before CFRelease; this is the other
    correct way, and the two implementations of one boundary should not disagree
    about handle lifetime.
    """
    dicts = _cs.DCSCopyAvailableDictionaries()
    if not dicts:
        return None
    try:
        n = _cf.CFSetGetCount(dicts)
        values = (ctypes.c_void_p * n)()
        _cf.CFSetGetValues(dicts, values)
        for ref in values:
            if _cfstr(_cs.DCSDictionaryGetIdentifier(ref)) == identifier:
                return _cf.CFRetain(ref)
    finally:
        _cf.CFRelease(dicts)
    return None


def lookup(word, dictionary=None):
    s = _cf.CFStringCreateWithCString(None, word.encode(), UTF8)
    if not s:
        return None
    try:
        res = _cs.DCSCopyTextDefinition(dictionary, s, _CFRange(0, _cf.CFStringGetLength(s)))
        if not res:
            return None
        try:
            n = _cf.CFStringGetLength(res)
            buf = ctypes.create_string_buffer((n + 1) * 4)
            if not _cf.CFStringGetCString(res, buf, len(buf), UTF8):
                return None
            return buf.value.decode()
        finally:
            _cf.CFRelease(res)
    finally:
        _cf.CFRelease(s)


if __name__ == "__main__":
    if len(sys.argv) not in (2, 3):
        print("usage: capture.py <word> [dictionary-identifier]", file=sys.stderr)
        sys.exit(2)
    ref = None
    if len(sys.argv) == 3:
        ref = dictionary_by_id(sys.argv[2])
        if ref is None:
            print(f"no dictionary with identifier {sys.argv[2]}", file=sys.stderr)
            sys.exit(3)
    try:
        text = lookup(sys.argv[1], ref)
    finally:
        if ref is not None:
            _cf.CFRelease(ref)
    if text is None:
        print(f"no entry: {sys.argv[1]}", file=sys.stderr)
        sys.exit(1)
    sys.stdout.write(text)
