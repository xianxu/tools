//go:build darwin && cgo && conformance

package main

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/conformance"
)

// TestClipboardNativeLiteralConformance checks the real helper and native flavor
// inventory on a retained unique board; it never opens the user's clipboard.
func TestClipboardNativeLiteralConformance(t *testing.T) {
	name, read, closeBoard, err := newIsolatedClipboard()
	if err != nil {
		conformance.SkipOrFail(t, "isolated pasteboard unavailable", err)
		return
	}
	defer closeBoard()
	writer := processClipboardWriter{executable: builtBinary(t), target: name, timeout: 2 * time.Second}
	for _, text := range []string{"{\\rtf1 literal}", "%!PS literal\x00世界\né", ""} {
		if err := writer.Write(context.Background(), text); err != nil {
			t.Fatal(err)
		}
		got, flavors, err := read()
		if err != nil {
			t.Fatal(err)
		}
		if got != text || !slices.Contains(flavors, "public.utf8-plain-text") {
			t.Fatalf("text=%q flavors=%v, wanted %q literal text", got, flavors, text)
		}
		// macOS derives other plain-text encodings automatically. RTF/EPS-looking
		// prefixes must never cause it to advertise a rich or executable flavor.
		for _, flavor := range flavors {
			switch flavor {
			case "public.utf8-plain-text", "public.utf16-external-plain-text", "public.utf16-plain-text", "com.apple.traditional-mac-plain-text":
			default:
				t.Fatalf("unexpected non-literal flavor %q", flavor)
			}
		}
	}
}
