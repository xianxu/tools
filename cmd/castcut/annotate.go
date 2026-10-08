package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
)

//go:embed viewer.html
var viewerHTML []byte

// maxNotes bounds one notes upload; a captions file is a few kilobytes.
const maxNotes = 1 << 20

// newAnnotateHandler serves one take to the viewer on 127.0.0.1:port and
// keeps its notes in the sidecar file. It answers only requests addressed to
// that exact host (a DNS-rebinding page cannot reach it) and accepts writes
// only from its own origin.
func newAnnotateHandler(castPath string, port int) http.Handler {
	host := fmt.Sprintf("127.0.0.1:%d", port)
	origin := "http://" + host
	notes := sidecarPath(castPath)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(viewerHTML)
	})
	mux.HandleFunc("GET /session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"cast": filepath.Base(castPath), "notes": filepath.Base(notes)})
	})
	mux.HandleFunc("GET /cast", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(castPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	})
	mux.HandleFunc("GET /notes", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(notes)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	})
	mux.HandleFunc("PUT /notes", func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && o != origin {
			http.Error(w, "cross-origin write refused", http.StatusForbidden)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxNotes))
		if err != nil {
			http.Error(w, fmt.Sprintf("notes over %d bytes", maxNotes), http.StatusRequestEntityTooLarge)
			return
		}
		if err := writeAtomic(notes, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host {
			http.Error(w, "wrong host", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

// writeAtomic replaces path with data via a temp file in the same directory,
// so a reader (cut, an editor) never sees half a file.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after the rename
	// Keep the sidecar readable like any file the operator made; CreateTemp is 0600.
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func runAnnotate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("castcut annotate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: castcut annotate [--port N] [--no-open] <take.cast>\n\n"+
			"serves the take to a browser viewer; notes save to <take>.captions.txt.\n\n")
		fs.PrintDefaults()
	}
	port := fs.Int("port", 0, "port on 127.0.0.1 (0 picks a free one)")
	noOpen := fs.Bool("no-open", false, "print the URL instead of opening a browser")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return errUsage
	}
	castPath := pos[0]
	data, err := os.ReadFile(castPath)
	if err != nil {
		return err
	}
	if _, err := parseCast(castPath, data); err != nil {
		return err
	}
	// A temp file outlives its write only if an annotate died mid-write; the
	// sidecar itself was never touched, so old leftovers are garbage. A
	// write takes milliseconds, so a minute spares another annotate's.
	stale, _ := filepath.Glob(filepath.Join(filepath.Dir(castPath), "."+filepath.Base(sidecarPath(castPath))+".*.tmp"))
	for _, f := range stale {
		if fi, err := os.Stat(f); err == nil && time.Since(fi.ModTime()) > time.Minute {
			os.Remove(f)
		}
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	bound := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d/", bound)
	fmt.Fprintf(stderr, "castcut: serving %s at %s\n  notes save to %s\n  the page loads asciinema-player from cdn.jsdelivr.net\n  Ctrl-C to stop\n",
		castPath, url, sidecarPath(castPath))
	fmt.Fprintln(stdout, url)
	if !*noOpen && runtime.GOOS == "darwin" {
		if err := exec.Command("open", url).Run(); err != nil {
			fmt.Fprintf(stderr, "castcut: could not open a browser (%v); open %s yourself\n", err, url)
		}
	}
	srv := &http.Server{Handler: newAnnotateHandler(castPath, bound)}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() { <-stop; srv.Close() }()
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
