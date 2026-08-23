package llmtest

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"
)

// SkipIfUnreachable ends the test as SKIPPED when the service is not RUNNING.
//
// "Not running" is not "changed", and every live suite must draw that line or it
// reports drift on a stopped proxy — telling the operator to re-record captures
// that are perfectly good.
//
// It probes the TCP endpoint, not the API. An earlier version issued a real
// Complete and skipped on ErrUnavailable, which was worse than no guard at all:
// a renamed or withdrawn model answers 502 through this proxy, that classifies as
// ErrUnavailable, and the suite would have SKIPPED on precisely the drift it
// exists to catch. A connection that cannot be established is the only signal
// that means "nothing is there"; anything that answers, even with an error, is
// the service talking and belongs to the suite.
//
// One helper rather than a check per suite: the set of live suites grows, and a
// per-site check is a site the next one forgets.
func SkipIfUnreachable(t *testing.T, baseURL string) {
	t.Helper()

	u, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("llmtest: unparseable base URL %q: %v", baseURL, err)
	}
	host := u.Host
	if u.Port() == "" {
		if u.Scheme == "https" {
			host = net.JoinHostPort(u.Hostname(), "443")
		} else {
			host = net.JoinHostPort(u.Hostname(), "80")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host)
	if err != nil {
		t.Skipf("service not running at %s, so nothing has drifted: %v", baseURL, err)
	}
	conn.Close()
}
