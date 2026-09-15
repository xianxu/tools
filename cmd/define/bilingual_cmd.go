package main

import (
	"errors"
	"fmt"
	"strings"
)

const bilingualUsage = "Toggle bilingual definitions. With on or off, set it explicitly. On shows the selected language followed by English; off shows only the selected language. Default on; saved per deck, for this session otherwise."

func parseBilingualArgs(args []string, current bool) (bool, error) {
	if len(args) == 0 {
		return !current, nil
	}
	if len(args) == 1 {
		switch strings.ToLower(args[0]) {
		case "on":
			return true, nil
		case "off":
			return false, nil
		}
	}
	return current, fmt.Errorf("use /bilingual, /bilingual on, or /bilingual off")
}

func bilingualState(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func (d deps) bilingualEnabled() bool { return d.bilingual == nil || *d.bilingual }

// sessionSetBilingual persists before replacing the live value. A declined or
// unavailable store still permits an explicitly reported session-only setting.
func sessionSetBilingual(d *deps, persist func(bool) error) func(bool) (bool, error) {
	return func(on bool) (bool, error) {
		saved := false
		if persist != nil {
			err := persist(on)
			if err != nil && !errors.Is(err, errDeckDeclined) {
				return false, err
			}
			saved = err == nil
		}
		d.bilingual = &on
		return saved, nil
	}
}

func runBilingual(c commandCtx, args []string) int {
	on, err := parseBilingualArgs(args, c.bilingual)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: %v\n", err)
		return 2
	}
	if c.setBilingual == nil {
		fmt.Fprintln(c.stderr, "define: /bilingual needs a writable deck or an interactive session")
		return 2
	}
	saved, err := c.setBilingual(on)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /bilingual: %v\n", err)
		return 2
	}
	suffix := ""
	if !saved {
		suffix = " (session only; not saved)"
	}
	fmt.Fprintf(c.stdout, "  bilingual %s%s\n", bilingualState(on), suffix)
	return 0
}

func durableBilingualSetter(persist func(bool) error) func(bool) (bool, error) {
	if persist == nil {
		return nil
	}
	return func(on bool) (bool, error) { err := persist(on); return err == nil, err }
}
