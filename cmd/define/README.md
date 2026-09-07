# define

Print a word's dictionary definition with Google-style IPA, and play its
pronunciation. A learner's tool: it builds a deck out of what you look up, asks
you about it later, and answers questions a dictionary cannot.

It reads the dictionaries already installed on the machine — no account, no
index, no network for a lookup. The directory you run it in *is* the deck.

```sh
go build -o ~/bin/define ./cmd/define    # from the repo root
```

macOS only, and deliberately: the dictionaries come from Dictionary.app through
`CoreServices`, which is where the definitions and the IPA actually live.

## Using it

```sh
define                      # interactive: type a word, / for commands, ^C to quit
echo sycophantic | define   # or feed it words on stdin
define sycophantic          # definition + /ˌsikəˈfan(t)ik/, played 3x
define /history             # a command works as an argument too
define --sound 1 record     # play once instead of three times
define -no-audio bank       # no fetch, no sound
define -locale gb schedule  # British pronunciation
define -lang es -locale us jalapeño   # Latin American, not Castilian
define -lang es madrugar    # one lookup in Spanish, without switching
define -lang it pizza       # and in Italian: the Devoto-Oli, not NOAD
define -pron fr arrondissement  # the French recording, English everything else
define -raw record          # the unparsed dictionary entry
define -no-color bank       # never emit ANSI (also automatic when piped)
```

## Reviewing what is due

**`define --play` reviews what is due today.** There are four kinds of question,
and you never choose which you get: which one you meet depends on how well you
already know the word, and on what material the tool has for it.

### The sentence, once a word has one

The word's own sentence with the word blanked out, and four words to choose
from. `define --harvest` writes these ahead of time, so they cost nothing to
show:

```
Judge Mehta noted that securities fraud claims lay outside his ___ and
transferred that portion of the case to the Southern District of New York.

1  defenestrate
2  bailiwick
3  ephemeral
4  obsequious
```

**This is the form the tool prefers when it can**, because picking the word that
fits a sentence is a harder and more useful test than picking a definition that
matches a word. Reveal it and you see the sentence whole, which is the point:
the word doing its work in the context it was written for.

The line under the question tells you what works:

```
1-4 = pick the word, ? = bad question, d = remove from deck, Ctrl-C to stop
```

**If a question is bad, press `?`.** That records it — with the four options you
were shown, which is what makes it diagnosable later — and moves on WITHOUT
marking you wrong. A broken question is not evidence about you, so it does not
move the word's schedule in either direction.

It stays offered after you have answered, which is usually when you notice:

```
any key = next word, ? = bad question, d = remove from deck, Ctrl-C to stop
```

### Multiple choice, once your deck can supply distractors

The word appears with up to four definitions, one of them right. This is the
main form, because recognising a meaning among plausible alternatives is a
harder and more useful test than deciding for yourself whether you knew it. A
young deck gives two or three options rather than four — there is nothing to pad
them with — and the prompt always names the digits that actually work.

```
$ define --play
sycophantic

1  an isolated flat-topped hill with steep sides
2  behaving or done in an obsequious way in order to gain advantage
3  a small short-tailed wallaby with a short face
4  an official report of the proceedings of a court

1-4 = pick the definition, d = remove from deck, Ctrl-C to stop
```

**The keys line and the figures line are chrome, and they look like it.** Both
are dimmed, and a blank row separates them from whatever you are reading — so the
sitting is what you see and those two rows are the frame around it. On a board
the keys line sits above the grid rather than below it, as the picture further
down shows.

**The wrong answers are your own words**, taken from your deck — never invented
by a model, so this works offline and costs nothing. They are also chosen to
differ from one another: where your deck allows, one is a specialist sense the
dictionary labels (`Law`, `Grammar`, `Nautical`), one is marked for register
(`informal`, `archaic`, `dated`), and one is ordinary vocabulary. Which one you
pick is recorded, not just whether you were right — so "kept picking the archaic
ones" is a thing your history can eventually tell you.

A word is not offered as a distractor against a word whose dictionary definition
mentions it. NOAD defines close synonyms through each other — `sycophantic` is
glossed *"behaving or done in an obsequious way"* — and that cross-reference is
exactly the case where two options could both be defensible. It is a filter, not
a proof: it only matches headwords of six characters or more (shorter ones like
"thing" appear in too many definitions by coincidence), and two words can be
close in meaning without the dictionary ever linking them.

Answer and the full entry appears, with the right answer and what you picked
named above it.

### The board: settled words, and words no test can be built for

**The form set is one sentence: a multiple choice tests you, the board triages
you.** Two kinds of word reach the grid, by two different doors.

**A word that has climbed three rungs of the ladder**, because it is settled. It
is the BOX that decides, not a tally of correct answers — a miss knocks a word
back down, so one recalled five times and missed twice is still tested properly.

**A word no multiple choice can be built for**, and there are exactly three
reasons:

- **the deck has no other word to draw on** — the very first reviews;
- **the entry is nothing but cross-references** ("another term for …"), so there
  is no definition to be the right answer;
- **the entry defines a different word** — the dictionary sends derived forms to
  their base, so looking up *bargainer* returns *bargain*, and offering that
  definition as *bargainer*'s meaning would be wrong.

That used to be its own form — the word alone, and you rated yourself —
and it was the same instrument as the grid at sixteen times the cost: nothing was
being checked either way. It was also the worst place for self-report, since a
word you have just met is the one you are most likely to think you know.

Sixteen at once, one keystroke or one click each: you are saying whether you still
have it, not proving it.

```
$ define --play
marking [yes] no drop, Tab cycles, click or key, Enter ends, Ctrl-C to stop
[0] arrondissement  [1] bailiwick       [2] keel            [3] mesa
[4] ephemeral       [5] quokka          [6] potassium       [7] ligament
[8] sycophantic     [9] concrete        [a] parrot          [b] run
[c] light           [d] bank            [e] set             [f] obsequious

sycophantic  behaving in an obsequious way to gain advantage
12 of 41 · ~12 reviews/day · 3.2 new words/day at 20 a sitting
```

**This is why a large deck stays affordable.** Most of a grown deck is words you
mostly know, each costing a few reviews a year — and one at a time that is most
of the day's work. On the grid it is a glance. A multiple choice puts a whole
dictionary entry on screen for every word; a board puts one line up for the
entire sweep.

**Click a word, or press the key printed beside it** — `0`–`9` then `a`–`f`, in
order and with no gaps. `Tab` cycles what a mark MEANS — yes, then no, then
drop; the bracketed one on the prompt line is live, and that line is the last
thing a short window gives up.

**Drop mode removes a word from the deck** rather than answering it, which is how
you throw away a typo or a word you never meant to keep without leaving the
sitting. Its history is kept, so looking the word up again brings it back exactly
where it was. It is two Tabs from the default deliberately: the destructive mode
is never one press away from the one you start in.

**A marked word turns green for yes, red for no, and struck-out for a drop, and
keeps its key** — so you
can still see what you answered and the grid still reads the same way. A word can
only be marked once: the answer is written the moment it lands, so there is
nothing to take back.

**`Enter` takes everything still unmarked as "no"** — "I am out of time, ask me
all of these again". `Ctrl-C` does the opposite and costs nothing: what you
marked is saved, and what you did not is simply not reviewed today.

**`Enter` is held if the window is too short to show the whole board**, and the
prompt line says so. It would otherwise demote words that were never on screen —
a short window drops the bottom rows of the grid. Marking what you can see still
works, and `Ctrl-C` is still free; make the window taller, or stop.

A word marked yes counts as a correct answer, never as a confident one — the
grid is self-report, so it can never earn the double promotion a real retrieval
test can. As a board closes it leaves one line naming the words you marked no.

| key | does |
|---|---|
| `1`–`4` | multiple choice: pick the definition, or on a cloze pick the word |
| `?` | on a cloze: bad question — records it with the options you were shown, and moves on without marking you wrong |
| `0`–`9`, `a`–`f` | board: mark the word printed beside that key |
| click | board: mark that word. Anywhere else in a sitting, a click plays the word rather than answering |
| Tab | board: cycle what a mark means — yes, no, then drop |
| Enter | board: finish, taking everything unmarked as "no" — held while the window is too short to show the whole board. Elsewhere: see the answer, like space |
| space | see the answer first — on a multiple choice this shows which option is right, so it is on you not to then press it |
| `d` | remove this word from the deck — its history is kept. On a board it is a cell's key instead, and `Tab` to drop mode is how you remove a word there: a grid has no single current word |
| PageUp / PageDown, wheel | scroll back through the sitting — a long entry no longer pushes the word off the top |
| Ctrl-C | stop; everything you answered is already saved |

Ctrl-C stops whenever you like and keeps everything you answered — each answer
is written as it happens, not at the end.

After an `n` the definition is on screen and the prompt changes:

```
any key = next word, d = remove from deck, Ctrl-C to stop
```

**Words come back on a widening schedule, and each correct recall multiplies the
wait by 1.6** — so 1, 1, 2, 4, 6, 10, 16, 26, 42, 68, 109 days and onward. You see
a new word tomorrow and again the day after, which is when forgetting is
steepest; a word you have recalled ten times you will not see again for months.

There is no top rung. A word you know well drifts to a year, then two, and keeps
drifting — it never leaves, it just gets cheap. That is what makes a large deck
affordable: the cost of a word falls about as fast as the deck grows.

**A miss halves the box** rather than dropping one step. A word at a 281-day
interval falls back to 16 days, which is a real chance to relearn it; a word at
4 days barely moves. And it climbs back faster than it went up — once you have
known a word, relearning it is quicker than learning it was, and the schedule
knows that.

`-count` bounds a sitting (default 20). A bar pinned to the bottom of the screen
shows how far in you are and what the deck costs, and it updates as you answer:

```
7 of 18 · ~14 reviews/day · 0.9 new words/day at 20 a sitting
```

The same figures close the sitting, under the score:

```
7 right, 3 wrong
~14 reviews/day · 0.9 new words/day at 20 a sitting
```

A brand-new deck looks expensive — every unreviewed word is due tomorrow — and
gets cheaper fast as words climb.

A sitting takes the screen the same way the interactive session does: the bar
stays at the bottom, a definition longer than the window is scrolled rather than
lost, resizing the window redraws — and wraps what comes after it to the new
width — and everything you reviewed is printed back into your terminal when you
quit.

**The words in a sitting are clickable too.** Click the word you are being asked
about to hear it; after a reveal, click anything in the definition — the headword
or the language after `ORIGIN` — exactly as in the interactive session. A click
never answers: hearing the word is what `y`/`n` are answering *about*, so it plays
and nothing else. Narrow the window and the links follow the text as it
re-wraps; a link inside a line that had to be broken drops out rather than
guessing, because one that played the word beside the one you pointed at would be
worse than no link.

Because it draws a whole screen, `--play` needs one. `define --play > file` and
`define --play -no-color` both say so and stop rather than filling a file with
escape sequences or painting control codes at a terminal that was asked not to
receive any. No API key: the deck and the dictionary are
enough, and the review loop never reaches for the model. Pronunciation audio is fetched over the network
only when a word is REVEALED, so a sitting you answer entirely with `y` makes no
network call at all; `--no-audio` makes one fully offline either way.

## The interactive session

On a terminal, `define` with no word opens a line editor and draws the session
itself:

| key | does |
|---|---|
| Up / Down | walk history — narrowed to what you have typed |
| Right / End / Tab | accept the grey suggestion |
| Enter | define what you typed (never the suggestion) |
| Enter on an empty line | replay the pronunciation, without moving the screen |
| Cmd+Delete (Ctrl-U) | clear the line |
| PageUp / PageDown, wheel | scroll back through the session |
| Ctrl-C | quit, including mid-playback |

Definitions wrap to your terminal width at word boundaries, and follow it when
you resize the window.

Because `define` owns the screen while it runs, the mouse belongs to it too —
**hold Option to select text** (Shift in some terminals). Everything the session
showed is printed back into your terminal when you quit, so the words you looked
up are in your scrollback to return to. The frame itself is not: the prompt you
were typing at and the `♫ playing` indicator were ephemeral, and stay that way.

### Clickable words

**Underlined words are clickable.** Click the headword to hear it again; click
the language after `ORIGIN` to hear the word in *that* language — `concrete` in
French, `jalapeño` in Spanish — without typing a command. Every language an
etymology names as a source is its own target, so `piano`'s "either from French,
or … Italian" gives you both: you point at the one you meant. Cognates and dead
stages are not offered, because they are not something a speaker says today.

It works on words you have scrolled back to, not just the last one. A click on
ordinary text does nothing.

### The words you already know

**Words you have looked up show in green** — in the line you type, in definitions,
and in answers — so the vocabulary you are building is visible rather than
something you have to remember having met. Looking up `sycophantic` when you
already know `obsequious` shows you the connection in the gloss itself. A word
looked up in this session turns green the moment you next see it.

Definition headwords and labels stay their own colour; the highlight marks
vocabulary in prose, which is where noticing a word you know actually tells you
something.

### Completion

**The grey suggestion follows the word you are typing, anywhere in the line.** It
completes from what you have looked up and asked before, so a long word you know
you want but not how to spell finishes itself in the middle of a question:

```
you type:   what's the difference to obseq
you see:    what's the difference to obseq|uious      (the tail in grey)
```

Whole lines still win over single words — if you start retyping a question you
have asked, the rest of it appears — and a short word mid-sentence is left alone,
so `to` does not offer to become `torpid`.

## Asking questions

**Type a question and it is answered instead of looked up.** There is no mode and
no prefix to remember:

```
› sycophantic                            # a word: the dictionary entry
› what's the difference to obsequious?   # a question: answered by the model
› hot dog                                # still a word — two of them
```

The dictionary decides which is which, and that is why multi-word headwords keep
working: `define` asks it first, and only classifies what it does not have. So
`hot dog` and `a priori` are definitions, while a line it has no entry for that
reads as a question — a wh-word, a question mark, or a request like `use it in a
sentence` — goes to the model. Anything else is still a miss, so a typo says
`no dictionary entry` rather than starting a conversation.

Both directions have a one-key escape, and neither is the only way to reach its
outcome:

| prefix | means |
|---|---|
| `?` | ask, even if it is a word — `?why` asks about *why* instead of defining it |
| `\` | define, even if it reads as a question — `\how so` answers `no dictionary entry` |

The answer is streamed, and **Ctrl-C stops the answer rather than the session** —
you land back at the prompt with the word you were reading still current. (In a
one-shot, `define "…?"`, there is no session to return to, so it ends the run.)

What the model is told is the directory you are in: the word on screen and its
dictionary entry, what you have looked up this session, your recent deck,
the learner model if you keep one, and the earlier questions in this session — so a
follow-up like `give me two more examples` resolves against the answer before it.
Nothing is remembered between runs except the files, which means a fresh process
answers as well as a long-running one and you can read the context with `cat`.

## The learner model

**`define --reflect` writes down who it thinks you are.** It reads your deck and
your lookup history and produces the learner model: a working level, the domains
you read in, and — the part that matters — what practice material should DO about
each. Every claim names the words it was read off, and a claim citing a word your
deck does not hold is dropped before you see it.

It is batch and on demand: nothing calls a model while you are looking a word up.
Below a dozen words it declines and says so, because a learner model built from
four lookups is a confident guess.

**`## Corrections` is yours.** Disagree with it in your own words and re-run
`--reflect`: everything from that heading down comes back byte-for-byte, and a
correction outranks anything inferred above it.

Questions need a model configured (see `--llm-check` below); without one, `define`
says so and exits `1` rather than looking up a sentence.

**`-raw` never asks**, on either route: it is the scripting form, so an unforced
miss stays a miss, and an explicit `?` alongside it is a usage error (exit `2`)
rather than a guess at which of the two contradicting flags you meant.

## Practice material, written ahead of time

**`define --harvest` prepares the material a review sitting will use.** It reads
your deck and, for every word it has not seen before, records two facts that
never change: a CEFR band (`A1`–`C2`) and a subject domain. Those are what make
a good wrong answer possible — a distractor is *selected* from real words at your
level, never invented.

It is batch, on demand, and the only thing in `define` that may take a while.
Nothing a sitting does ever waits on it, and a review works perfectly well
against a deck that has never been harvested; it simply has less to draw on.

**It asks about each word once, ever.** Run it again and it re-reads what it
already knows and makes no calls at all, so the cost does not grow with time —
only with new words. `-limit N` caps the **model calls** one run may make
(default 200) — counted across both halves, since banding a word and writing its
item are both calls. A capped run is a partial run and says so, and running again
picks up where it stopped, so a large deck is harvested over several runs rather
than in one long one.

**The domain usually costs nothing.** When your dictionary already prints a
subject field on a word — `Law`, `Medicine`, `Nautical` — that label is used
directly and the model is never asked. Most words carry no field at all and are
simply `general`, which is the common and correct answer.

**It then writes the practice items.** For every banded word without material,
the model writes one sentence that USES the word — not a definition — and the
wrong answers are **selected from your own deck**, never invented: same subject
field where it can, at your level or one step below, and never above it. A word
you do not know is not a wrong answer you can reject; it is one you eliminate by
ignorance, which teaches nothing.

Four things are checked before an item is kept, and each says so when it fires:

- **the sentence must actually contain the word**, unblanked. Checked without
  asking a model, because it costs nothing to check.
- **the word's meaning must do the work.** *"His ___ behaviour was noted by
  all"* is thrown out: the word is decorative there, and the sentence would read
  the same with almost any adjective. You will see four options, so the word does
  not have to be the only one in the language that fits — the sentence has to be
  *about* what it means.
- **the sentence must not define the word.** *"the alewife, the small silver
  herring"* is a reading test, not a vocabulary test.
- **it must name someone or somewhere real** — not "a manager", not "the
  company".

Then each wrong answer is checked on its own: a near-synonym like `obsequious`
beside `sycophantic` would also fit the blank, so it is vetoed and dropped.

If nothing survives, the word stays unauthored and the next run tries again. A
word keeps at most four items; the oldest are dropped.

**Where the wrong answers come from is reported**, because on a small deck it
matters. It looks for words in the same subject field first, then in a field
*you* read in, then ordinary vocabulary, then anything at or below your level —
and only reaches above your level as a last resort. Whatever it settled for, it
says so: `3 item(s) drew options from any domain, at or below band` means your
deck could not supply better ones yet, not that the material is wrong.

**`define --harvest -agreement N` measures how stable the banding is.** It
re-asks a sample of already-banded words N times each and reports how often the
answers agree. It writes nothing, so measuring cannot disturb what it measures.

Read the number for exactly what it says: **agreement is stability, not
correctness.** A model that gives the same wrong band every time scores a perfect
1.00. It measures the one property the cache actually depends on — that a band
assigned once is the band this model usually gives — and it cannot tell you the
scale is right.

`--harvest` needs a model configured (see `--llm-check`). Without one it says so
and exits `1`. If the model becomes unavailable mid-run, everything already
banded is saved and only the harvesting stops.

**One mode at a time.** `-harvest`, `--play`, `--reflect`, `-forget` and
`--llm-check` are modes, and asking for two on one line is a usage error (exit
`2`) rather than a guess at which you meant. Previously whichever dispatched
first silently won.

## What it writes, where you run it

**`define` reads and writes the current directory.** *Every* successful lookup —
one-shot, piped, or in the editor — records the word where you started `define`,
so your deck and history build themselves. **Every question that reaches the model
is recorded too, by its text** — whatever became of the answer, since what you
asked is the signal, not whether it arrived. (What decides it is whether a request
was actually sent: with no model configured nothing is, so nothing is recorded.
A model that is configured but does not answer says so, and the question is kept.)

```
words/en/sycophantic.yaml  one file per word, under its language
words/es/madrugar.yaml     a different language, a different deck
events/2026-08-21.yaml     append-only, one file per day (named in UTC)
                           kinds: looked-up, asked, reviewed. A reviewed record
                           carries correct:, and a MISS from the multiple-choice
                           form also carries missed: — which kind of wrong answer
                           it was (domain, register, general)
facts/en/sycophantic.yaml  a word's CEFR band and domain, written by
                           --harvest and never re-asked; per language, because
                           `red` is a different word in English and Spanish
items/en/sycophantic.yaml  practice items authored ahead of time, per
                           language. Nothing writes this yet — authoring is the
                           next milestone; the directory is here because the
                           facts above are what it will be authored from
lang.txt                   which language this directory is in
user-model.en.md           written by --reflect, read to pitch answers; one per
                           language, because it is read off that language's
                           deck. Its ## Corrections section is yours and is
                           never rewritten
```

## Languages

**One language at a time.** `/lang` says which one, `/lang es` switches, and the
setting stays with the directory — unlike `/sound`, which lasts one session.
It has to persist: a one-shot `define madrugar` has no session to inherit from,
and re-declaring the language at every lookup is the friction the mode removes.
Everything follows it — the deck a word files into, the words `--play` offers,
and the recording that is fetched, unless `-pron` asked otherwise for one
lookup. `-lang es` is the one-run form, for scripts that should not have to
change state to ask a question.

### Hearing a word in its source language

**Hearing a borrowed word in its source language.** `define -pron fr
arrondissement` plays the French recording and changes nothing else: the entry
is still the English one, the word still files into the English deck, and the
next lookup is English again.

<!-- pron-help -->hear THIS lookup in another language without switching the session: -pron fr arrondissement. The entry's ORIGIN says which; at the prompt /pron alone reads it for you. Falls back to the session's recording, and says so, when the source has none<!-- /pron-help -->

You name the language; the tool never guesses it. That is a decision with
measurements behind it — the dictionary writes `ORIGIN French` for
*arrondissement* and for *police* alike, and the CDN serves `police_fr_fr`,
`restaurant_fr_fr` and `machine_fr_fr` perfectly happily. Anything automatic
would replace the English recording for a large class of ordinary words that
merely came from French centuries ago. The entry prints its `ORIGIN` right
above, so the answer is on screen when you need it.

Coverage is partial and the tool says so rather than going quiet: `-pron fr
hotel` prints `no fr recording for hotel; played the en one`. Italian and
Japanese have no recordings in this CDN generation at all, so they always report.

`-locale` picks the regional variant, and it works for **every** language:

<!-- locale-help -->regional variant of the pronunciation, per language: en us|gb; es es (Castilian, cazar /θ/) or us (seseo, /s/). Others exist — the CDN decides, not a list here<!-- /locale-help -->

For Spanish the choice is **phonemic, not an accent flavour**: `es_es` is
Castilian, where *cazar* /θ/ and *casar* /s/ are different words; `es_us` is
Latin American *seseo*, where both are /s/. Picking one picks which sound system
you learn. Spanish entries carry no written pronunciation at all — the spelling
already determines it — so the recording is the *only* place that information
exists, which makes this choice matter more for Spanish than for English.

Nothing here enumerates which combinations exist: a pair the CDN does not serve
simply gets the same "no recording" warning as any other miss.

The event log is deliberately *not* split by language: a review event names a
word, and which deck it came from is the deck's business. "How much did I study
today" stays one question rather than a join.

A deck from before this existed is moved under `words/en/` the next time
`define` runs — along with any `user-model.md`, which becomes
`user-model.en.md` — and it says so. That move cannot tell languages apart — a Spanish
word filed earlier lands in `words/en/` too — so it prints what it moved and
leaves a `mv` to you. It never overwrites and never deletes.

A failed lookup is recorded as history but never enters the deck, so typos are
recallable with Up-arrow without becoming vocabulary. `-raw` records nothing —
scripting a dictionary should not mutate a deck — and neither does it ask.

```sh
define --forget sycophantic   # drop a word and its material (history is kept)
DEFINE_NO_CAPTURE=1 define …  # write nothing in this directory
```

`DEFINE_NO_CAPTURE=1` means *nothing at all*, and that includes the event log —
which is what persists your history, so with it set, history is session-only. It
also means the directory is not **read**: answers come back un-adapted, with no
deck and no learner model behind them.

The directory *is* the deck: run `define` somewhere else and you get a different
one. If that directory happens to be synced, so is your vocabulary; `define`
neither knows nor cares.

With no word and no terminal, `define` reads stdin: a word defines and speaks it, a bare return
replays the *pronunciation* of the current one — nothing is re-fetched, and the
screen is left as it was provided you let the sound finish — and Ctrl-C quits
silently. `-raw` prints the unparsed entry and never plays. The prompt appears
only on a terminal, so piping stays clean. Flags are session settings — `define
--sound 1` opens the loop with single playback.

## Checking the model connection

`define` can use a language model for the parts a dictionary cannot do. Every one
of those features **degrades silently by design** — no key or no network means
they are skipped, not failed, so a review session is never blocked on a third
party. That makes a misconfiguration invisible, which is what this flag is for:

```sh
define --llm-check
```

```
  base url  http://127.0.0.1:8317
  model     claude-opus-5 (effort high)
  key       (set, short)
  latency   1.379s
  tokens    22 in, 5 out (0 thinking)
  preamble  1902 tokens injected upstream (not ours)
  answer    "PONG"
  ok
```

It is the one surface where an unusable configuration is **loud**: it exits
non-zero and names the reason. Configure it with `DEFINE_LLM_API_KEY` (or `ANTHROPIC_API_KEY`),
`DEFINE_LLM_BASE_URL`, `DEFINE_LLM_MODEL`, `DEFINE_LLM_EFFORT` and
`DEFINE_LLM_TIMEOUT` (a duration, e.g. `90s`) — all five the tool reads.

**You usually need none of them.** The default is a local proxy on
`127.0.0.1:8317` — the parley-managed cliproxyapi — and `define` supplies that
proxy's loopback handshake token itself, so questions work with nothing set at
all. Point `DEFINE_LLM_BASE_URL` anywhere else and a key becomes required, since
a token invented for a local proxy has no business being sent to a real
provider.

Exit codes: `0` success; `1` the request failed; `2` usage error. What produces
each is enumerated rather than sampled, because a list of examples goes stale the
moment a new one is added and nothing says so:

| code | produced by |
|---|---|
| `1` | no dictionary entry; a question with no model configured; a question whose model **was** configured and did not deliver (the message carries the cause); a model answer that could not be used; `--forget` found nothing to remove; `--llm-check` found no usable configuration; `--reflect` with too small a deck, with no deck at all, with no model, or with nothing in the answer left standing after the deck check |
| `2` | an unknown `/command`; a bare `?` or `\` with nothing after it; `-raw` combined with an explicit `?`; `--reflect` combined with a word |

A piped run exits `1` if any word failed and `2` if a command was malformed, so
`echo "$w" | define || …` works in a script; an interactive typo does not fail
the session.

A line beginning with `/` is a command rather than a word — `/` is safe as a
marker because no English headword starts with one, and `define` needs whole
lines for multi-word headwords like `hot dog`. The same reasoning picks `?` and
`\` for the two question hatches above: no headword begins with either. Type `/` to see what there is,
Tab to complete, `/help` to list them. It works the same from every entry mode:
`define /help`, `echo /help | define`, and `/help` typed at the prompt are one
thing.

`/history [N]` lists what you looked up in the last N days — two by default,
counted as local calendar days rather than N×24 hours. `N` can be written three
ways, so it reads the same whichever you reach for: `/history 7`,
`/history --days 7`, `/history --days=7`. It works from every entry mode, so
`define /history 7` and `echo '/history 7' | define` mean the same thing.

```
  defenestrate  today
  sycophantic   yesterday   2×
  perennial     Aug 1       2×
```

Deduped, and ordered by when each word was **first** seen, so one you keep
returning to holds its place instead of jumping to the top; the count is how
often you have looked it up. Words the dictionary could not find are kept for
up-arrow recall but never listed here — a typo is not vocabulary.

`/sound N` changes how many times a pronunciation plays for the rest of the
session; `/sound` on its own reports it, and `0` turns playback off. It is the
in-session form of `--sound`, which sets it for one run. (`-times` is the older
name for `--sound` and still works; passing both is a usage error rather than a
guess at which you meant.)

`/pron` replays the word you just looked up in its source language, once.
With no argument it reads the language off the entry's `ORIGIN` and tells you
which it chose — `ORIGIN says French` — and declines when `ORIGIN` names only a
historical stage (`Old French`, `Latin`) or a cognate (*"related to Dutch…"*),
because neither is a language anyone says the word in today. `/pron fr` names it
explicitly. Either way it
leaves nothing switched on — the next word is back to the session's own voice.
It is an action, not a setting, which is the difference from both `/sound` and
`/lang`: there is no `/pron` to undo. `-pron fr <word>` is the same thing for a
one-shot lookup.

`/lang` reports the language this directory is in; `/lang es` switches it and
keeps it. That is the deliberate difference from `/sound`: a language has to
survive the session, because a one-shot lookup has no session to inherit one
from. `-lang es` is the same choice for a single run, without writing it down.

Lookup goes through macOS's CoreServices, and **the dictionary follows the
language**. `/lang` says which books are answering.

<!-- curated-languages -->
- **English** — the New Oxford American Dictionary (hence the Google-matching
  notation), plus Apple Dictionary, which is where `iPhone` comes from.
- **Spanish** — the Larousse *Diccionario General*.
- **Italian** — the *Devoto-Oli*. Note that Italian has **no recordings** in the
  pronunciation CDN, so an Italian session gives you definitions and silence;
  and the Devoto-Oli writes syllabification with stress, `(cià·o)`, rather than
  a phonetic transcription.
<!-- /curated-languages -->

Each must be **monolingual** — indexed in its own language *and* defined in it.
That rules out books like the bilingual Oxford Spanish and Oxford Italian, which
are installed on many machines and would put English glosses in front of a
learner who asked for the other language.

So `mesa` is an isolated flat-topped hill in English and *"un tablero
horizontal, sostenido por uno o varios pies"* in Spanish, and `sycophantic` in a
Spanish session reports **no entry** — which is correct, and which this tool
could not say about anything before.

Two honest limits. The dictionaries are chosen from a short **curated list**,
because nothing in the system's metadata distinguishes a general dictionary from
a thesaurus; on a machine with a different set installed, nothing curated matches
and `define` falls back to searching every active dictionary and says so. And the
calls that select a dictionary are **private** — undocumented, and free to
disappear on an OS update — so they are resolved at run time and the tool
degrades to that same whole-set search rather than breaking. Only on that
fallback path does the host's Dictionary.app configuration decide what you get —
on the curated path it does not, which is the point.
