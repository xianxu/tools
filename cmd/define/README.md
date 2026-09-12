# define

`define` is a terminal app for dictionary definition with pronunciation
integration. It is also a learner's tool: every word you look up joins a deck
you can later play games to help you remember. Definition works without Internet; 
pronunciation requires Internet; free form chat requires LLM subscription. 

## The Basics

### Examples

```
# Start the program at the directory you want to keep your deck

./define

# At the prompt, just type a word and return, definition is returned, and the 
# word is pronounced three times. A bare return without any words repeats the 
# the pronunciation of previous word.

› sycophantic
sycophantic  syc·o·phan·tic
/ˌsikəˈfan(t)ik/
...

# Free form questions can be asked about nuances between words

› what's the difference between sycophantic and obsequious<CR>
Both mean "excessively flattering," but the motive and the method differ.
...

# Each day, play with words you need to remember with /play

› /play

epithelial

1  not covered with varnish.
2  adjective based on the first impression; accepted as correct until proved otherwise
3  a person who refuses to strike or to join a labor union or who takes over the job responsibilities of a striking worker.
4  relating to or denoting the thin tissue forming the outer layer of a body's surface and lining the alimentary canal and other hollow structures

0 right, 0 wrong
~18 reviews/day · 0.2 new words/day at 20 a sitting

# There's extensive typeahead system, press / to see what command are availble.
# Tyep first several chars of words you are learning to bring up auto completion.
# E.g. /sound 1 to pronounce word once instead the default three times.

› /sound 1

```

### Install

```sh
brew trust xianxu/tools          # third-party taps are untrusted by default
brew tap xianxu/tools
brew install xianxu/tools/define
```
macOS only, the definitions and the IPA come from Dictionary.app.

### The directory is the deck, so it asks first

The directory you run `define` in **is** the deck. That makes running it in the
wrong shell a quiet accident, so the first time it would write somewhere new it
asks:

```
$ cd /tmp && define sycophantic
define: /tmp is not a deck yet. Create one here? [y/N]
```

**A bare Enter declines**, because the cost of a wrong *yes* is a stray deck in
your home directory and the cost of a wrong *no* is re-running one command.

Declining does not stop the lookup — you still get the definition, the IPA and
the audio. Nothing is written, and everything that reads history reports **empty**
rather than refusing: `--stats` says so plainly, `--play` finds nothing due.

**When it cannot ask** — piped input, a script, CI — it does not create anything
and does not hang waiting for an answer nobody can give. The lookup still works.

```sh
define --here sycophantic     # yes, make THIS directory a deck; never asks
```

`--here` is the path for scripts, and it is the only one: since a non-terminal
never creates a deck, automation that wants one has to say so.

A directory that is already a deck is never asked about.

### Keyboard Shortcuts

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

### Clickable Words

**Underlined words are clickable.** Click the headword to hear it again; click
the language after `ORIGIN` to hear the word in *that* language — `concrete` in
French, `jalapeño` in Spanish — without typing a command. 

It works on words you have scrolled back to, not just the last one. A click on
ordinary text does nothing.

### Coloring of Words

**Words you have looked up show in green** in many different context: in the line
you type, in definitions, and in answers, so to constantly remind you of them.

Definition headwords and labels stay their own colour; the highlight marks
vocabulary in prose, which is where noticing a word you know actually tells you
something.

### Auto Completion

**The grey suggestion follows the word you are typing, anywhere in the line.** It
completes from what you have looked up and asked before, so a long word you know
you want but not how to spell finishes itself in the middle of a question:

```
you type:   what's the difference to obseq
you see:    what's the difference to obseq|uious      (the tail in grey)
```
This completion shows up as you type a sentence as well, though will only trigger
with first three letters typed.

### `/` command

Inside the session, everything besides looking a word up is a `/` command:

<!-- command-list -->
| command | does |
|---|---|
| `/help` | list the commands, or explain one |
| `/history` | words looked up recently |
| `/stats` | deck, streak and accuracy figures |
| `/play` | review the words due today |
| `/sound` | how many times to play a pronunciation |
| `/lang` | the language this deck is in |
| `/pron` | replay this word in its source language, once |
<!-- /command-list -->

`/play` runs today's review without leaving the prompt. Answer the questions, or
press Ctrl-C when you have had enough — either way you land back where you were,
with the session's summary in the scrollback above you. 

To check additional help for commands, type `/help [command]`.

<!-- command-usage -->
- `/help [command]` — With nothing, list the commands. With a command's name, say how to use it, which --help after any command also does.
- `/history [N | --days N | --days=N]` — The words looked up in the last N days. With nothing, the last 2; N is at most 3650.
- `/stats` — The deck, streak and accuracy figures for this directory. Takes no arguments.
- `/play` — Review the words due today; Ctrl-C stops and keeps every answer. Takes no arguments.
- `/sound [N]` — With nothing, how many times each pronunciation plays. With N, play it N times for the rest of this session; 0 turns playback off, and 20 is the most.
- `/lang [language]` — With nothing, the language in effect and the dictionary answering it. With a two-letter tag like es, switch to that language: saved when this directory is a deck, for this session otherwise.
- `/pron [language]` — Replay this word once in another language. With nothing, it reads the source language off the entry's ORIGIN and says which it chose. It declines when ORIGIN names only historical stages (Old French, Latin) or cognates ("related to Dutch …"), because neither is a language anyone speaks the word in today.
<!-- /command-usage -->

> NOTE: while other languages are available, only English dictionary is well tested.

### Asking Free-Form Questions

**Type a question and it is answered instead of looked up.** There is no mode and
no prefix to remember:

```
› sycophantic                            # a word: the dictionary entry
› what's the difference to obsequious?   # a question: answered by the model
› hot dog                                # still a word — two of them
```
The word lookup vs free form chat can be deterministically triggered by `?` and `\\` prefixes. The question-answer is driven by LLM; the definition is driven by local
dictionary.

| prefix | means |
|---|---|
| `?` | ask, even if it is a word — `?why` asks about *why* instead of defining it |
| `\` | define, even if it reads as a question — `\how so` answers `no dictionary entry` |

### Languages

The goal is to support multiple different languages, but only English is well tested.
One interesting cross language feature is the ability to hear pronunciation in original
language of a borrowed word. For example, try `arrondissement`, which is from French. 
Click on the `French` link in the ORIGIN section to hear French pronunciation of it.

The dictionary follows the language, and `/lang` says which one is answering. The curated ones:

<!-- curated-languages -->
- **English** — the New Oxford American Dictionary (hence the Google-matching
  notation), plus Apple Dictionary, which is where `iPhone` comes from.
- **Spanish** — the Larousse *Diccionario General*.
- **Italian** — the *Devoto-Oli*. Note that Italian has **no recordings** in the
  pronunciation CDN, so an Italian session gives you definitions and silence;
  and the Devoto-Oli writes syllabification with stress, `(cià·o)`, rather than
  a phonetic transcription.
<!-- /curated-languages -->


## Periodical Reviewing

**`/play` reviews what is due today**. There are four kinds of question.

### The sentence, once a word has one

The word's own sentence with the word blanked out, and four words to choose
from. 

```
Judge Mehta noted that securities fraud claims lay outside his ___ and
transferred that portion of the case to the Southern District of New York.

1  defenestrate
2  bailiwick
3  ephemeral
4  obsequious

1-4 = pick the word, ? = bad question, d = remove from deck, Ctrl-C to stop
```

**If a question is bad, press `?`.** That records it — with the four options you
were shown, which is what makes it diagnosable later — and moves on WITHOUT
marking you wrong. 

It stays offered after you have answered, which is usually when you notice:

```
any key = next word, ? = bad question, d = remove from deck, Ctrl-C to stop
```

### Multiple choice, once your deck can supply distractors

The word appears with up to four definitions, one of them right. 

```
sycophantic

1  an isolated flat-topped hill with steep sides
2  behaving or done in an obsequious way in order to gain advantage
3  a small short-tailed wallaby with a short face
4  an official report of the proceedings of a court

1-4 = pick the definition, d = remove from deck, Ctrl-C to stop
```

Once you have answered, the prompt changes:

```
any key = next word, d = remove from deck, Ctrl-C to stop
```

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

A word marked yes counts as a correct answer, never as a confident one — the
grid is self-report, so it can never earn the double promotion a real retrieval
test can. As a board closes it leaves one line naming the words you marked no.

Ctrl-C stops whenever you like and keeps everything you answered — each answer
is written as it happens, not at the end.

### Keys in a review

<!-- review-keys -->
| key | does |
|---|---|
| `1`–`4` | multiple choice: pick the definition, or on a cloze pick the word |
| `?` | on a cloze: bad question — records it with the options you were shown, and moves on without marking you wrong |
| `0`–`9`, `a`–`f` | board: mark the word printed beside that key |
| click | board: mark that word. Anywhere else, a click plays the word — the headword, a language named in the ORIGIN, or any word already in your deck, wherever it appears |
| Tab | board: cycle what a mark means — yes, no, then drop |
| Enter | board: finish, taking everything unmarked as "no" — held while the window is too short to show the whole board. Elsewhere: see the answer, like space |
| space | see the answer first — on a multiple choice this shows which option is right, so it is on you not to then press it |
| `d` | remove this word from the deck — its history is kept. On a board it is a cell's key instead, and `Tab` to drop mode is how you remove a word there: a grid has no single current word |
| PageUp / PageDown, wheel | scroll back through the sitting — a long entry no longer pushes the word off the top |
| Ctrl-C | stop; everything you answered is already saved |
<!-- /review-keys -->

### On a Schedule

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

## Advanced

### The learner model

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

Questions need a model configured (see [Checking the LLM model connection](#checking-the-llm-model-connection)); without one, `define`
says so and exits `1` rather than looking up a sentence.

**`-raw` never asks**, on either route: it is the scripting form, so an unforced
miss stays a miss, and an explicit `?` alongside it is a usage error (exit `2`)
rather than a guess at which of the two contradicting flags you meant.

### Practice material, written ahead of time

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

`--harvest` needs a model configured (see [Checking the LLM model connection](#checking-the-llm-model-connection)). Without one it says so
and exits `1`. If the model becomes unavailable mid-run, everything already
banded is saved and only the harvesting stops.

**One mode at a time.** A mode is a flag that makes `define` do one whole job and
exit — reviewing, harvesting, reflecting, reporting figures, forgetting a word,
checking the model configuration. Asking for two on one line and asking for two on one line is a usage error (exit
`2`) rather than a guess at which you meant. Previously whichever dispatched
first silently won.

### What it writes, where you run it

**`define` reads and writes the current directory — once you have said it may.**
In a directory that is already a deck, *every* successful lookup — one-shot, piped,
or in the editor — records the word where you started `define`, so your deck and
history build themselves. In a directory that is not one yet, it asks first, and
records nothing until you answer yes (see [The directory is the deck, so it asks first](#the-directory-is-the-deck-so-it-asks-first)). **Every question that reaches the model
is recorded too, by its text** — whatever became of the answer, since what you
asked is the signal, not whether it arrived. (What decides it is whether a request
was actually sent: with no model configured nothing is, so nothing is recorded.
A model that is configured but does not answer says so, and the question is kept.)

```
words/en/sycophantic.yaml  one file per word, under its language
words/es/madrugar.yaml     a different language, a different deck
events/2026-08-21.yaml     append-only, one file per day (named in UTC)
                           kinds: looked-up, asked, reviewed, flagged. A
                           reviewed record carries correct:, and a MISS from a
                           multiple-choice form also carries missed: — which
                           kind of wrong answer it was (domain, register,
                           general). A flagged record is a question you called
                           broken: it carries options: — every word you were
                           shown — and no correct:, because it is not evidence
                           about you and moves nothing
usage/sycophantic.yaml     the news cache: recent headlines using the word,
                           and WHEN they were fetched. Per word and FLAT, so it
                           is shared across languages — forgetting a word in one
                           clears it for the other, which costs a refetch rather
                           than lost work
facts/en/sycophantic.yaml  a word's CEFR band and domain, written by
                           --harvest and never re-asked; per language, because
                           `red` is a different word in English and Spanish
audio/sycophantic/51f6….mp3 the recorded pronunciation, kept so a replay
                           costs no network — and a `.yaml` beside it naming
                           the URL that answered. A DIRECTORY per word, one
                           file per voice inside it. Flat, not per language:
                           the filename's digest already says which voice a
                           recording is for, so a language shelf would be a
                           second answer to the same question. A word the CDN
                           has no recording for gets a record saying so,
                           believed for thirty days
items/en/sycophantic.yaml  practice items authored ahead of time by
                           --harvest, per language: a sentence with its answer
                           and the words vetoed to sit beside it. This is what
                           a cloze question is built from
lang.txt                   which language this directory is in
user-model.en.md           written by --reflect, read to pitch answers; one per
                           language, because it is read off that language's
                           deck. Its ## Corrections section is yours and is
                           never rewritten
```

**Look at any of it, and edit it if you like.** Everything above is plain YAML
beside one plain-text file and the cached recordings — no database, no index to
keep in step. Deleting a word's file forgets it; correcting a band in `facts/`
is read back on the next sitting; the `## Corrections` section of the learner
model is yours and is never rewritten.

The program treats what it reads back as UNTRUSTED, which is what makes that
safe rather than merely possible: a hand-edited file cannot forge a row in a
grid, an escape sequence in a stem cannot reach the terminal, and a recording
truncated to nothing is re-fetched instead of played as silence.

### From the command line

Everything the session does is also reachable as a one-shot command or from a
pipe, which is what scripts want. A word on the command line is looked up once,
and flags set things for that one run.

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

Two of those flags, in the words `define -h` uses:

- `-pron`: <!-- pron-help -->hear THIS lookup in another language without switching the session: -pron fr arrondissement. The entry's ORIGIN says which; at the prompt /pron alone reads it for you. Falls back to the session's recording, and says so, when the source has none<!-- /pron-help -->
- `-locale`: <!-- locale-help -->regional variant of the pronunciation, per language: en us|gb; es es (Castilian, cazar /θ/) or us (seseo, /s/). Others exist — the CDN decides, not a list here<!-- /locale-help -->

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

### Checking the LLM model connection

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
