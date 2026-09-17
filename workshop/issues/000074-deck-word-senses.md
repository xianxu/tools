---
id: 000074
status: open
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
---

# define: a deck word is a set of senses, not one word

## Problem

The deck models a word as one thing. It is keyed by `store.Key(word)`;
`store.WordFacts` holds exactly one `Band` and one `Domain` per word;
`schedule.Progress` is a `map[string]Progress` keyed by that same word key; and
`store.ReviewEvent` has no sense field.

But learning a word means learning a *sense* of it. NOAD puts a river's edge, a
financial institution, a tier of oars, a shot in pool and an aircraft's tilt in
one `bank` entry. Two consequences follow, and the second is not benign.

**Practice can test a sense the learner never met.** Surfaced while designing
#68: `senseFacts` commits to the FIRST usable gloss and the first domain label in
document order, and that gloss is what `renderAuthorPrompt` hands the model — so
an authored item is internally consistent. `entryUsages` does the opposite: it
walks every block and sense and flattens their examples into one list, discarding
which sense each came from. Measured over the 34 committed entry fixtures, 24
words have examples, 242 in total — `run` 40, `set` 36, `read` 26, `bank` 6
across at least four unrelated senses. Pick a stem from that flattened pool and
the item's stem, gloss, band and domain can each be anchored to a different part
of the entry.

**All senses collapse into one Progress.** A learner who knows one of `bank`'s
senses and meets the other four in review produces four wrong answers, and
`schedule.Progress` cannot tell them apart from four failures on one meaning. The
scheduler reads "does not know `bank`" and moves it up the queue, then re-tests
senses the learner has never encountered. This is invisible to the learner and it
is what would make polysemous words feel punishing.

## Spec

### Record every sense, not the learned one

Operator decision, 2026-09-17. **At lookup time the signal does not exist:**
`define <word>` returns the whole entry and nothing says which sense sent the
learner there. A mask inferred from a guess would silently filter out the very
stems that teach the sense they needed, and the failure would be undetectable.
So a deck word carries the entry's full sense set.

The accepted consequence is that practice may offer a sense the learner has not
met. That is a learning event rather than a failure — they re-check the entry,
which is how the word gets learned. What is NOT acceptable is the scheduling
effect above, which the learner cannot see and cannot correct; that is why
per-sense progress below is in scope rather than deferred.

### Arcane senses drop out with machinery that exists

`readGloss` already reports a gloss's AXIS, and `senseFacts` already filters on
`play.AxisDomain` specifically because `readGloss` also reports REGISTER
(`informal`, `archaic`) — a different axis that `ParseDomain` would otherwise
flatten to `general`. So archaic and rare senses are excluded by the same test,
with no new judgement and no model call.

### Progress becomes per (word, sense)

The load-bearing change. `store.ReviewEvent` gains a sense identifier and
`schedule`'s progress derivation re-keys on `(word, sense)` instead of `word`.

**Cheaper than it sounds, by prior design.** `schedule/box.go` and
`schedule/progress.go` both record a deliberate decision NOT to cache progress on
`store.Word` — "a cache that drifts … it would go wrong invisibly" — so progress
is derived from the event log on every read. This is therefore a new event field
plus a re-keyed derivation, not a migration of stored state. Events already on
disk carry no sense and derive as they do today.

### Sense identity needs a stable key

An index into the parsed entry's senses is the obvious choice and the wrong one:
a NOAD update that inserts or reorders a sense silently re-points every stored
event at a different meaning, and nothing would detect it.

Recommendation: a digest of the sense's gloss text. It is stable while the gloss
is, and a gloss that changed arguably IS a different sense — which is the right
default, because it expires the progress rather than mis-attributing it. The
failure mode to state: a cosmetic re-wording of a gloss resets that sense's
progress. That is the direction to fail in.

### The three signals that pin a sense

Strongest first, and none of them requires asking the learner a question:

1. **A passage mark** (#67) — the sentence the word was marked in disambiguates
   the sense. This is the one signal that exists at capture time.
2. **A cloze mark** (#75) — a word marked inside a stem is disambiguated by that
   stem.
3. **A wrong answer** — the stem's sense is already known to the program, so
   every wrong answer is a free sense observation with no new UI at all.

### Downstream

#68's stem filter is exactly this mask: candidates are admitted only from senses
the deck holds. #68 is blocked on this issue.

## Done when

- A deck word carries the entry's sense set, and a word with one sense behaves
  exactly as today — asserted, since that is most of the deck.
- Archaic and register-marked senses are excluded, driven by `readGloss`'s axis
  rather than a new list — a table over the committed entry fixtures.
- `schedule` derives progress per `(word, sense)`: a learner correct on one sense
  and wrong on another does NOT have the correct sense rescheduled, asserted
  against the queue rather than by inspecting Progress.
- Events written before this change derive exactly as they do today — a fixture
  of sense-less events produces the current schedule, byte for byte.
- A sense's identity survives a re-parse of the same entry text, and a changed
  gloss expires that sense's progress rather than re-pointing it at another —
  both asserted.
- A wrong answer records which sense the stem tested.
- `atlas/define.md` records the sense dimension, the identity rule and its
  failure mode, and the three signals.

## Plan

- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] M1 — sense identity: the digest, its stability test over the fixtures, and
      the arcane-sense filter from `readGloss`'s axis
- [ ] M2 — `ReviewEvent` gains the sense; the write sites set it; old events
      derive unchanged
- [ ] M3 — `schedule` re-keys progress on `(word, sense)`; the queue test
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-17

Filed out of #68's brainstorm, which surfaced the gap: `entryUsages` flattens
examples across senses while `senseFacts` commits to one, so a dictionary-sourced
stem can test a different meaning than the word's stored band and domain describe.
Measured the exposure over the committed fixtures before filing — 242 examples
across 24 words, concentrated in mid-frequency polysemous words (`bank`,
`concrete`, `content`, `present`, `minute`, `desert`); rare words have one sense
and no examples, and high-polysemy words are common enough to be unlikely deck
entries.

The record-all-senses decision is the operator's, on the grounds that lookup
carries no sense signal at all. The per-sense progress requirement is the part the
brainstorm added: the comprehension cost of offering an unmet sense is benign, the
scheduling cost is not.
