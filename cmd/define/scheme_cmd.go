package main

import (
	"fmt"
	"strings"
)

// schemeUsage is /scheme's row in the registry. M2's wording: detection (M3)
// adds "so define follows what the terminal reports" to auto.
const schemeUsage = "With nothing, the colour scheme in use and where it came from. light or dark sets it and saves it for every session; auto forgets the saved choice. The scheme picks the shade of the language tint: dark grey on a dark background, light grey on a light one."

// runScheme is /scheme (#70): report, choose, or forget the colour scheme.
func runScheme(c commandCtx, args []string) int {
	if len(args) > 1 {
		fmt.Fprintf(c.stderr, "define: /scheme takes one of light, dark or auto, not %q\n", strings.Join(args, " "))
		return 2
	}
	if len(args) == 0 {
		fmt.Fprintf(c.stdout, "  scheme %s\n", describeScheme(c.scheme.Load(), c.loop == loopEditor))
		return 0
	}
	arg, err := parseSchemeArg(args[0])
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /scheme: %v\n", err)
		return 2
	}
	st, err := applyScheme(c.scheme, arg, c.schemePersister, c.loop != loopOneShot)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /scheme: %v\n", err)
		return 2
	}
	fmt.Fprintf(c.stdout, "  scheme %s\n", describeScheme(st, c.loop == loopEditor))
	return 0
}
