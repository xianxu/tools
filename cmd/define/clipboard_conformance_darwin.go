//go:build darwin && cgo && conformance

package main

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

static char *define_clipboard_string(CFStringRef value) {
 CFIndex length = CFStringGetMaximumSizeForEncoding(CFStringGetLength(value), kCFStringEncodingUTF8) + 1;
 char *text = malloc(length);
 if (text && !CFStringGetCString(value,text,length,kCFStringEncodingUTF8)) {free(text);return NULL;}
 return text;
}
static OSStatus define_clipboard_isolated(PasteboardRef *board, char **name) {
 OSStatus status = PasteboardCreate(NULL, board);
 if (status != noErr) return status;
 CFStringRef board_name = NULL;
 status = PasteboardCopyName(*board, &board_name);
 if (status == noErr) {
  *name = define_clipboard_string(board_name);
  CFRelease(board_name);
  if (!*name) status = memFullErr;
 }
 if (status != noErr) {CFRelease(*board);*board=NULL;}
 return status;
}
static OSStatus define_clipboard_read(PasteboardRef board, CFDataRef *data, CFArrayRef *flavors) {
 PasteboardSynchronize(board);
 ItemCount count=0;
 OSStatus status=PasteboardGetItemCount(board,&count);
 if (status != noErr) return status;
 if (count != 1) return badPasteboardItemErr;
 PasteboardItemID item=NULL;
 status=PasteboardGetItemIdentifier(board,1,&item);
 if (status != noErr) return status;
 status=PasteboardCopyItemFlavors(board,item,flavors);
 if (status != noErr) return status;
 status=PasteboardCopyItemFlavorData(board,item,CFSTR("public.utf8-plain-text"),data);
 if (status != noErr) {CFRelease(*flavors);*flavors=NULL;}
 return status;
}
static char *define_clipboard_flavor(CFArrayRef flavors, CFIndex i) {
 return define_clipboard_string((CFStringRef)CFArrayGetValueAtIndex(flavors,i));
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// newIsolatedClipboard retains a uniquely named native board for subprocess
// conformance. Read and close are serialized; ordinary builds omit this harness.
func newIsolatedClipboard() (name string, read func() (string, []string, error), closeBoard func(), err error) {
	var board C.PasteboardRef
	var cName *C.char
	if status := C.define_clipboard_isolated(&board, &cName); status != 0 {
		return "", nil, nil, fmt.Errorf("create isolated pasteboard: %d", int(status))
	}
	name = C.GoString(cName)
	C.free(unsafe.Pointer(cName))
	if name == "" || name == clipboardName {
		C.CFRelease(C.CFTypeRef(board))
		return "", nil, nil, errors.New("native board was not isolated")
	}
	var mu sync.Mutex
	read = func() (string, []string, error) {
		mu.Lock()
		defer mu.Unlock()
		if board == 0 {
			return "", nil, errors.New("isolated pasteboard closed")
		}
		var data C.CFDataRef
		var flavors C.CFArrayRef
		if status := C.define_clipboard_read(board, &data, &flavors); status != 0 {
			return "", nil, fmt.Errorf("read isolated pasteboard: %d", int(status))
		}
		defer C.CFRelease(C.CFTypeRef(data))
		defer C.CFRelease(C.CFTypeRef(flavors))
		text := string(C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(data)), C.int(C.CFDataGetLength(data))))
		var names []string
		for i := C.CFIndex(0); i < C.CFArrayGetCount(flavors); i++ {
			s := C.define_clipboard_flavor(flavors, i)
			if s == nil {
				return "", nil, errors.New("read pasteboard flavor name")
			}
			names = append(names, C.GoString(s))
			C.free(unsafe.Pointer(s))
		}
		return text, names, nil
	}
	closeBoard = func() {
		mu.Lock()
		defer mu.Unlock()
		if board != 0 {
			C.PasteboardClear(board)
			C.CFRelease(C.CFTypeRef(board))
			board = 0
		}
	}
	return name, read, closeBoard, nil
}
