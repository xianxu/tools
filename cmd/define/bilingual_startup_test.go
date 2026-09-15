package main

import (
	"bytes"
	"github.com/xianxu/tools/cmd/define/store"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestBilingualStartup(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := store.WriteBilingual(dir, false); err != nil {
		t.Fatal(err)
	}
	d := (deps{newStore: openStore}).withStore(options{}, io.Discard)
	if d.bilingualEnabled() {
		t.Fatal("saved off replaced by default on")
	}
	if d.persistBilingual == nil {
		t.Fatal("missing persistence callback")
	}
	if err := d.persistBilingual(true); err != nil {
		t.Fatal(err)
	}
	next := (deps{newStore: openStore}).withStore(options{}, io.Discard)
	if !next.bilingualEnabled() {
		t.Fatal("saved on did not survive restart")
	}
	noCapture := (deps{newStore: openStore}).withStore(options{noCapture: true}, io.Discard)
	if !noCapture.bilingualEnabled() || noCapture.persistBilingual != nil {
		t.Fatal("no-capture setting contract")
	}
}

func TestBilingualDeclinedDeckWritesNothing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	perm := newDeckPermission(func() bool { return false })
	d := (deps{newStore: openStore, deckPermission: perm}).withStore(options{}, io.Discard)
	saved, err := sessionSetBilingual(&d, d.persistBilingual)(false)
	if err != nil || saved || d.bilingualEnabled() {
		t.Fatalf("saved %v enabled %v err %v", saved, d.bilingualEnabled(), err)
	}
	if _, err := os.Stat(filepath.Join(dir, store.BilingualFileName())); !os.IsNotExist(err) {
		t.Fatalf("setting written after decline: %v", err)
	}
}

func TestBilingualCommandRegistry(t *testing.T) {
	c, ok := findCommand("bilingual", commands)
	if !ok {
		t.Fatal("missing command")
	}
	var out, errout bytes.Buffer
	d := deps{}
	cc := newCommandCtx(d, options{}, &out, &errout)
	cc.setBilingual = sessionSetBilingual(&d, nil)
	if c.run(cc, nil) != 0 || d.bilingualEnabled() {
		t.Fatalf("command did not toggle: %s", &errout)
	}
}
