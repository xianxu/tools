castcut — record, annotate and cut captioned terminal demos

castcut turns a raw terminal recording into a short, captioned demo: the take
plays fast between captions and drops to real time around each one, for long
enough to read it. It is written to be followed step by step, by a person or
by an agent driving one.

USAGE

  castcut record [-o take.cast] [--capture-input] [--idle-time-limit S] -- <command> [args...]
  castcut annotate [--port N] [--no-open] <take.cast>
  castcut cut <take.cast> [captions.txt] [-o cut.cast] [timing flags]
  castcut --version
  castcut <command> -h        flags of one command

REQUIREMENTS

  asciinema 3 (`brew install asciinema`) for record. A browser with network
  access for annotate: the page loads asciinema-player 3.17.0 from
  cdn.jsdelivr.net (pinned, with an integrity hash). cut needs nothing.

THE FLOW

1. Record a take.

     castcut record -- ./parley_app --demo

   Runs the command under asciinema at the terminal's own size: size the
   window before recording (smaller reads better when a blog column scales
   it down; around 95x36 worked for the parley post). Exit the command to
   stop. The take goes to recordings/take-NN.cast under the current directory
   (the next free number) unless -o names a file; an existing file is never
   overwritten. castcut exits with the recorded command's status.

   Launch the app in an isolated demo profile (its own config, state, and
   seed content) so takes are repeatable; that launcher belongs to each app,
   not to castcut. Keep a shot list beside it. Record several takes; cut the
   best.

   --idle-time-limit S caps idle gaps in the take itself (asciinema's own
   limit; the viewer and cut honour it). Usually unnecessary: cut squeezes
   idle anyway, and an uncapped take keeps the real pauses for annotating.

   --capture-input also records keystrokes as `i` events. They pass through
   cut untouched, but no embed shows them yet, and they include anything
   typed, passwords too.

2. Annotate it.

     castcut annotate recordings/take-03.cast

   Serves the take to a viewer at http://127.0.0.1:PORT/ and opens it (macOS;
   --no-open just prints the URL). Play the take; at each moment worth a
   caption press Alt+T: the player pauses and a `~m:ss.s  ` line is inserted
   at the cursor. Type the caption after it. Notes save as you type to
   recordings/take-03.captions.txt, beside the take; "Download notes" saves a
   copy. Ctrl-C stops the server. annotate prints the next command when it
   starts and again when it stops (the cut, or embedding if this is a cut).

   The captions file is plain text, one caption per line, blank lines
   ignored, any order:

     ~0:05.0  Parley installs its plugins on first launch.
     ~0:42.3  Branch off to ask about radiators.

   The stamp is view time: the take's clock with its idle_time_limit applied,
   exactly what the viewer shows. The leading ~ is optional. Edit the file by
   hand freely. Two tabs on one take: the last save wins, whole file.

   Annotating a cut cast (castcut annotate cut.cast) previews it with its
   captions overlaid, as the blog will show them.

3. Cut it.

     castcut cut recordings/take-03.cast

   Reads take-03.captions.txt unless a captions file is named, and writes
   recordings/take-03-cut.cast unless -o names another file (re-cutting
   overwrites it; the take and captions are never written). Prints the cut
   length and each caption's place in it. The cut is the file you publish.

   Timing (defaults in brackets):
     --lead S      real time starts S before each stamp [1]
     --min-hold S  each caption plays at least S real seconds [4]
     --wps N       reading speed, words per second [3.5]
     --beat S      extra seconds after the reading time [1]
     --idle S      between captions, idle per gap is cut to at most S [1]
     --speed X     ...and then played X times faster [5]

   A caption's hold is max(min-hold, words/wps + beat). A caption that would
   overlap the previous one starts when that one ends. A caption that outlasts
   the take holds the final frame until it is read. Too fast to read? Raise
   --min-hold or lower --wps. Too slow between captions? Raise --speed.
   Iterate: cut is instant and never touches the take or its captions.

4. Embed it. See EMBEDDING.

OUTPUT CONTRACT

  The cut is an asciicast v3 file (https://docs.asciinema.org/manual/asciicast/v3/):

  - header `captions`: [{"start": s, "end": s, "text": "..."}], in output
    seconds (3 decimals), sorted, non-overlapping. Show a caption while
    start <= t < end, where t is the player's current time.
  - one `m` (marker) event per caption at its start, whose data is the text;
    at the same instant a marker comes before any output.
  - `idle_time_limit` is removed: the cut already paced idle time.
  - every other header field and every event of the take pass through.

  Consumers should read `captions` from the header; markers are for players
  that only understand markers (they carry no end time).

EMBEDDING

  xianxu.dev (Astro): copy the cut to public/casts/<name>.cast and put

      <div class="cast-embed" data-cast="/casts/<name>.cast"></div>

  in the post (optional data-poster="npt:0:30" picks the frame shown before
  playback). src/components/blog/CastEmbed.astro mounts the player and draws
  the header captions over the terminal's bottom-left.

  Plain asciinema-player: AsciinemaPlayer.create({url}, el) plays the cut;
  with markers it can pause at each caption (pauseOnMarkers: true). To show
  captions, fetch the cast text yourself, JSON.parse its first line, and
  every 100 ms set an overlay's text to the caption whose start <= t < end,
  t = player.getCurrentTime(); pass {data: text} to create().

  Another destination: implement that overlay. The box castcut's viewer and
  CastEmbed.astro use, in fractions of the terminal element: left 0.02, max
  width 0.5, bottom edge at 0.935 of its height, font size 3.2% of its width.

FILES AND THEIR LIFETIMES

  recordings/take-NN.cast        record writes; you delete takes you are done with
  recordings/take-NN.captions.txt annotate writes; dies with its take
  recordings/take-NN-cut.cast    cut writes (or the -o path); you publish it

  castcut deletes nothing.

LIMITS

  Takes, stamps and timing flags are bounded to one week of seconds, so the
  arithmetic stays finite; nothing real comes close. A cut of 200,000 events
  takes under a second.

EXIT STATUS

  0 success · 1 error (message on stderr) · 2 usage · record: the recorded
  command's status.
