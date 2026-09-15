# Native Oxford `rendir` capture

`rendir.json` is the exact, unedited native record capture promoted from
`/tmp/define66-rendir-native.json` on 2026-09-15 for tools#66. It contains the
installed `com.apple.dictionary.OxfordSpanish` Spanish-to-English entry
`s_b-es-en0040662`: 11,404 UTF-8 HTML bytes and 1,960 UTF-8 Text bytes.
SHA-256: `2aabe21f7d073b9598dcabd3394291274f4fe7fa9565c77683e7a1881c99c771`.

The record preserves the native XML declaration and public XHTML DTD reference;
parsing never fetches that DTD.

Regenerate on macOS with Oxford Spanish installed, using the same native seam
as `TestBilingualNativeDirection`: call
`newSpanishEnglishSource().Records("rendir")`, then JSON-encode the returned
`[]bilingualRecord` with `json.MarshalIndent(records, "", "  ")` and append a
newline. The provider uses `DCSCopyRecordsForSearchString` and
`DCSRecordCopyData` formats 0 (HTML) and 3 (Text). Capture the returned values
without editing their content. If an installed dictionary update changes the
record, review the source and expected hierarchy before replacing this fixture.

Strict live verification:

```sh
CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestBilingualNative' -count=1
```

The regression deliberately checks the literal A/B/C group hierarchy, the A2
and A4 lettered sub-senses, and inline Spanish-example/English-translation rows.
Other existing native captures exercise idioms and repeated text. Tests may
mutate native XML to exercise corruption, wrappers, and emphasis; those
mutations are adversarial parser inputs, not claimed native service responses.

To verify actual native `rendir` structure and dictionary-section backgrounds,
and optionally capture the production renderer's output at widths 32 and 80
for both target languages and themes:

```sh
CONFORMANCE_STRICT=1 DEFINE_LAYOUT_CAPTURE_PREFIX=/tmp/define66-actual \
  go test -tags conformance ./cmd/define \
  -run '^TestBilingualNativeRendirLayout$' -count=1 -v
```

The capture writes `.ansi` terminal output and unpainted `.txt` alongside it.
Both come from the assembled native dictionary and production
`renderDefinitionOutput`/`serializeOutput` path. The test independently checks
every terminal cell's background, including short-row fill and blank rows.
