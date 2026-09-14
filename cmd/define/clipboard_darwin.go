//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

static OSStatus define_clipboard_write(const UInt8 *name, CFIndex name_len, const UInt8 *text, CFIndex text_len) {
 CFStringRef target = CFStringCreateWithBytes(NULL, name, name_len, kCFStringEncodingUTF8, false);
 if (!target) return memFullErr;
 CFDataRef data = CFDataCreate(NULL, text, text_len);
 if (!data) { CFRelease(target); return memFullErr; }
 PasteboardRef board = NULL;
 OSStatus status = PasteboardCreate(target, &board);
 if (status == noErr) status = PasteboardClear(board);
 if (status == noErr) status = PasteboardPutItemFlavor(board, (PasteboardItemID)1, CFSTR("public.utf8-plain-text"), data, 0);
 if (board) CFRelease(board);
 CFRelease(data);
 CFRelease(target);
 return status;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// writeNativeClipboard publishes actual UTF-8 bytes with an explicit text flavor.
func writeNativeClipboard(target, text string) error {
	name := []byte(target)
	data := []byte(text)
	status := C.define_clipboard_write((*C.UInt8)(unsafe.Pointer(unsafe.SliceData(name))), C.CFIndex(len(name)), (*C.UInt8)(unsafe.Pointer(unsafe.SliceData(data))), C.CFIndex(len(data)))
	if status != 0 {
		return fmt.Errorf("macOS clipboard status %d", int(status))
	}
	return nil
}
