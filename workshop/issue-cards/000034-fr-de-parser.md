---
id: 000034
status: open
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
github_issue:
---

# French and German need parser work before curating: the entry collapses into one blob

## Problem

Split from `#31`, which narrowed to Italian once the three books were measured
rather than assumed. `#29`'s Spec had called all three "one line each"; that is
true of the `curated` map and false of the reading experience.

Measured 2026-08-29 by querying each dictionary directly through
`selectedDictionary{ids: …}`, five common words each:

| dictionary | senses the parser finds | longest single undifferentiated blob |
|---|---|---|
| `NOAD` (English, baseline) | 153 | 144 runes |
| `it.Devoto-Oli` (shipped in `#31`) | 245 | 645 runes |
| **`fr.Multi`** | **6** | **2,811 runes** |
| **`de.DDDSI`** | **5** | **4,404 runes** |

`Haus` puts its grammar table, its `TYPISCHE VERBINDUNGEN`, its whole `SYNONYME`
list and its `HERKUNFT` etymology into ONE example string. That is not a
definition a person reads; it is the raw entry with quote marks around it.
