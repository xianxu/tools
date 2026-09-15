package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// helpPrepareBudget bounds one sitting's preparation. It is a deadline the
// client observes, not a promise about the proxy (#61 PQ-6): at 2–5 s per
// structured call a cold 20-word queue takes five or six calls.
const helpPrepareBudget = 30 * time.Second

// errHelpRefused is the cause when the model answered but some translations
// failed checkHelp or could not be matched to the text they were for.
var errHelpRefused = errors.New("some translations failed their checks")

// practiceHelpCache is the session's view of the deck's translation cache.
//
// A POINTER in deps, so every copy of deps, a nested sitting's included,
// shares what an earlier sitting learned. The disk half is optional: with no
// deck, or where the directory never agreed to be one, translations live for
// the session only. Entries read back are UNTRUSTED and are re-checked by
// prepareHelp exactly as a fresh reply is.
//
// In memory the list grows by at most one sitting's texts per sitting; the
// file it is written to is bounded by the store.
type practiceHelpCache struct {
	mu      sync.Mutex
	loaded  bool
	entries []store.HelpEntry
	read    func() []store.HelpEntry
	write   func([]store.HelpEntry) error
}

func newPracticeHelpCache(read func() []store.HelpEntry, write func([]store.HelpEntry) error) *practiceHelpCache {
	return &practiceHelpCache{read: read, write: write}
}

// loadLocked reads the disk half once per session. Callers hold mu.
func (c *practiceHelpCache) loadLocked() {
	if c.loaded {
		return
	}
	c.loaded = true
	if c.read != nil {
		c.entries = c.read()
	}
}

// lookup returns what the cache holds for keys. A nil cache holds nothing.
func (c *practiceHelpCache) lookup(keys []helpKey) map[helpKey]string {
	got := map[helpKey]string{}
	if c == nil || len(keys) == 0 {
		return got
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	want := make(map[helpKey]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	for _, e := range c.entries {
		k := helpKey{lang: e.Lang, mode: helpMode(e.Mode), text: e.Source}
		if want[k] {
			got[k] = e.English
		}
	}
	return got
}

// remember records accepted translations and writes the cache through. Only
// ACCEPTED translations reach it: a failure is never cached, so the next
// sitting asks again instead of inheriting the refusal.
func (c *practiceHelpCache) remember(got map[helpKey]string, at time.Time) error {
	if c == nil || len(got) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadLocked()
	kept := make([]store.HelpEntry, 0, len(c.entries)+len(got))
	for _, e := range c.entries {
		if _, replaced := got[helpKey{lang: e.Lang, mode: helpMode(e.Mode), text: e.Source}]; !replaced {
			kept = append(kept, e)
		}
	}
	for _, k := range sortedHelpKeys(got) {
		kept = append(kept, store.HelpEntry{Lang: k.lang, Mode: string(k.mode), Source: k.text, English: got[k], At: at})
	}
	c.entries = kept
	if c.write == nil {
		return nil
	}
	return c.write(kept)
}

// prepareHelp gives today's questions their English before the first one is
// asked (#61).
//
// THE ONE PLACE A SITTING MAY REACH FOR THE MODEL, and only when all of these
// hold: bilingual display is on, the deck is not English, some shown text has no
// usable cached translation, and a model seam exists. So an English sitting and
// an off sitting never resolve a configuration, and a warm cache never builds a
// client. Whatever cannot be translated stays in the deck's language, with one
// line saying so; nothing is retried here beyond the client's own bounded policy.
func prepareHelp(ctx context.Context, d deps, opt options, qs []play.Question, out, warn io.Writer) {
	if !helpWanted(d) || len(qs) == 0 {
		return
	}
	needs := helpNeedsOf(qs, d.lang)
	keys := helpKeysOf(needs)
	if len(keys) == 0 {
		return
	}
	accepted := acceptHelp(needs, d.practiceHelp.lookup(keys))
	var missing []helpKey
	for _, k := range keys {
		if _, ok := accepted[k]; !ok {
			missing = append(missing, k)
		}
	}
	var cause error
	if len(missing) > 0 && hasModelSeam(d) {
		fresh, err := translateHelp(ctx, d, opt, missing, out)
		cause = err
		good := acceptHelp(needs, fresh)
		for k, v := range good {
			accepted[k] = v
		}
		if cause == nil && len(good) < len(missing) {
			cause = errHelpRefused
		}
		if err := d.practiceHelp.remember(good, d.clock.Now()); err != nil && !errors.Is(err, errDeckDeclined) {
			fmt.Fprintf(warn, "define: could not save the English help (%v); it is used for this sitting only\n", err)
		}
	}
	missed := applyHelp(qs, needs, accepted)
	if missed > 0 && cause != nil && ctx.Err() == nil {
		fmt.Fprintf(warn, "define: no English help for %d question(s) (%s); they stay untranslated\n", missed, helpCause(cause))
	}
}

// translateHelp asks the model for the missing texts, batch by batch, inside
// one deadline. It stops at the first failed call: a proxy that refused one
// batch will refuse the next, and the learner is waiting.
func translateHelp(ctx context.Context, d deps, opt options, keys []helpKey, out io.Writer) (map[helpKey]string, error) {
	got := map[helpKey]string{}
	batches, excluded := planHelpBatches(keys)
	var tooLong error
	if len(excluded) > 0 {
		tooLong = fmt.Errorf("%d text(s) over %d bytes", len(excluded), maxHelpSource)
	}
	if len(batches) == 0 {
		return got, tooLong
	}
	cfg, err := llm.Resolve(d.getenv)
	if err != nil {
		return got, err
	}
	n := 0
	for _, b := range batches {
		n += len(b)
	}
	fmt.Fprintf(out, "define: translating %d text(s) into English for this sitting; later sittings reuse them\n", n)
	client := foregroundClient(d.newLLM(cfg), out, opt)
	ctx, cancel := context.WithTimeout(ctx, helpPrepareBudget)
	defer cancel()
	for _, batch := range batches {
		reply, err := llm.Run(ctx, client, helpTask(d.lang, batch))
		if err != nil {
			return got, err
		}
		for k, v := range readHelpReply(batch, reply) {
			got[k] = v
		}
	}
	return got, tooLong
}

// helpCause is the warning's reason, in words for a learner rather than a
// wrapped error chain where one is known.
func helpCause(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("the model took longer than %s", helpPrepareBudget)
	case errors.Is(err, llm.ErrUnavailable):
		return "the model is unavailable"
	default:
		return err.Error()
	}
}
