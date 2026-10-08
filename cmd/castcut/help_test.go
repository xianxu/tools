package main

import (
	"regexp"
	"strings"
	"testing"
)

// The manual documents every command and every flag; the flag list comes from
// each command's own -h, so a new flag without a manual entry fails here.
func TestHelpCoversEveryCommandAndFlag(t *testing.T) {
	code, help, _ := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("--help exit %d", code)
	}
	if _, bare, _ := runCLI(t); bare != help {
		t.Error("bare castcut does not print the manual")
	}
	for _, want := range []string{"OUTPUT CONTRACT", "header `captions`", "EMBEDDING", "cast-embed", "CastEmbed.astro"} {
		if !strings.Contains(help, want) {
			t.Errorf("manual lacks %q", want)
		}
	}
	flagRe := regexp.MustCompile(`(?m)^  -(\S+)`)
	for _, cmd := range []string{"record", "annotate", "cut"} {
		if !strings.Contains(help, "castcut "+cmd+" ") {
			t.Errorf("manual lacks a `castcut %s` usage line", cmd)
		}
		code, _, usage := runCLI(t, cmd, "-h")
		if code != 0 {
			t.Errorf("%s -h: exit %d", cmd, code)
		}
		flags := flagRe.FindAllStringSubmatch(usage, -1)
		if len(flags) == 0 {
			t.Fatalf("%s -h listed no flags:\n%s", cmd, usage)
		}
		for _, f := range flags {
			name := f[1]
			if name == "o" || name == "out" || name == "headless" {
				continue // -o is in the usage lines; --headless is for tests and CI
			}
			if !strings.Contains(help, "--"+name) {
				t.Errorf("manual does not document %s --%s", cmd, name)
			}
		}
	}
}
