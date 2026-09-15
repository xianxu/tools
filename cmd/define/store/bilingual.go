package store

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const bilingualFileName = "bilingual.txt"
const maxBilingualSettingBytes = 32

// BilingualFileName is the per-deck bilingual display setting's filename.
func BilingualFileName() string { return bilingualFileName }

// ReadBilingual defaults on for missing, unreadable, or malformed settings.
// Limit the read before parsing: a corrupted setting must not allocate an
// arbitrarily large buffer during startup.
func ReadBilingual(dir string) bool {
	f, err := os.Open(filepath.Join(dir, bilingualFileName))
	if err != nil {
		return true
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxBilingualSettingBytes+1))
	if err != nil {
		return true
	}
	return parseBilingualSetting(b)
}

func parseBilingualSetting(b []byte) bool {
	if len(b) > maxBilingualSettingBytes {
		return true
	}
	return !strings.EqualFold(strings.TrimSpace(string(b)), "off")
}

// WriteBilingual atomically replaces the per-deck setting before callers change
// their live session value.
func WriteBilingual(dir string, on bool) error {
	value := "off\n"
	if on {
		value = "on\n"
	}
	return writeBytesAtomic(filepath.Join(dir, bilingualFileName), []byte(value))
}
