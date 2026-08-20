#!/usr/bin/env python3
"""Print a word's New Oxford American Dictionary entry as flat text.

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
_cs.DCSCopyTextDefinition.restype = ctypes.c_void_p
_cs.DCSCopyTextDefinition.argtypes = [ctypes.c_void_p, ctypes.c_void_p, _CFRange]


def lookup(word):
    s = _cf.CFStringCreateWithCString(None, word.encode(), UTF8)
    if not s:
        return None
    try:
        res = _cs.DCSCopyTextDefinition(None, s, _CFRange(0, _cf.CFStringGetLength(s)))
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
    if len(sys.argv) != 2:
        print("usage: capture.py <word>", file=sys.stderr)
        sys.exit(2)
    text = lookup(sys.argv[1])
    if text is None:
        print(f"no entry: {sys.argv[1]}", file=sys.stderr)
        sys.exit(1)
    sys.stdout.write(text)
