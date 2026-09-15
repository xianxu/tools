package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// An independent wrapper proves quieting is dispatched through the interface,
// including inherited Store methods, without teaching quietStore another shape.
type quietTestWrapper struct{ store.Store }

func (s quietTestWrapper) Quiet() store.Store { return quietTestWrapper{Store: quietStore(s.Store)} }

// Every production concrete Store, including embedded method sets, must provide
// a Quiet view. Source checking keeps unexported implementations in the scan;
// compiled file lists honor build tags and supply cgo's translated declarations.
func TestEveryStoreHasAQuietView(t *testing.T) {
	pkgs, _, _, st, qt := quietGuardPackages(t)
	var found []string
	for _, p := range pkgs {
		stores, missing := storesWithoutQuiet(p, st, qt)
		found = append(found, stores...)
		for _, name := range missing {
			t.Errorf("%s implements store.Store without store.Quieter; background reads could write into the frame", name)
		}
	}
	if len(found) < 3 {
		t.Fatalf("found only %v; expected YAML, Mem, and the deck gate", found)
	}
}

func TestQuietStoreGuardSeesEmbeddedMethods(t *testing.T) {
	_, fset, imports, st, qt := quietGuardPackages(t)
	src := `package fixture
 import "github.com/xianxu/tools/cmd/define/store"
 type missing struct { store.Store }
 type forwarded struct { store.Store }
 func (forwarded) Quiet() store.Store { return nil }
 type inherited struct { *store.YAML }
 type pointerQuiet struct { store.Store }
 func (*pointerQuiet) Quiet() store.Store { return nil }
 type shadowed struct { *store.YAML }
 func (shadowed) Quiet() {}
 type unrelated struct {}
 func (unrelated) Deck() {}
 func (unrelated) AppendEvent() {}
 `
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	cfg := types.Config{Importer: imports, IgnoreFuncBodies: true}
	p, err := cfg.Check("fixture", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, missing := storesWithoutQuiet(p, st, qt)
	if !slices.Equal(got, []string{"fixture.forwarded", "fixture.inherited", "fixture.missing", "fixture.pointerQuiet", "fixture.shadowed"}) || !slices.Equal(missing, []string{"fixture.missing", "fixture.pointerQuiet", "fixture.shadowed"}) {
		t.Fatalf("stores=%v, missing Quiet=%v; embedded Store must fail, forwarded/promoted Quiet must pass, unrelated signatures must not count", got, missing)
	}
}

// Use Go's method sets, not method-name guesses: signatures, pointer receivers,
// promotion, shadowing and interface embedding are all part of implementation.
func storesWithoutQuiet(p *types.Package, st, qt *types.Interface) (found, missing []string) {
	for _, name := range p.Scope().Names() {
		obj, ok := p.Scope().Lookup(name).(*types.TypeName)
		if !ok || obj.IsAlias() {
			continue
		}
		typ := obj.Type()
		if _, ok := typ.Underlying().(*types.Interface); ok {
			continue
		}
		ptr := types.NewPointer(typ)
		if !types.Implements(typ, st) && !types.Implements(ptr, st) {
			continue
		}
		key := p.Path() + "." + name
		found = append(found, key)
		// Each usable Store form must quiet itself; a pointer-only Quiet cannot
		// protect a value that independently implements Store.
		if types.Implements(typ, st) && !types.Implements(typ, qt) || types.Implements(ptr, st) && !types.Implements(ptr, qt) {
			missing = append(missing, key)
		}
	}
	return found, missing
}

type quietGuardImporter struct {
	types.Importer
	checked map[string]*types.Package
}

func (i *quietGuardImporter) Import(path string) (*types.Package, error) {
	if p := i.checked[path]; p != nil {
		return p, nil
	}
	return i.Importer.Import(path)
}

func quietGuardPackages(t *testing.T) ([]*types.Package, *token.FileSet, *quietGuardImporter, *types.Interface, *types.Interface) {
	t.Helper()
	cmd := exec.Command("go", "list", "-compiled", "-export", "-deps", "-json", ".", "./store")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, stderr.String())
	}
	type metadata struct {
		ImportPath, Dir, Export string
		CompiledGoFiles         []string
	}
	var targets []metadata
	exports := map[string]string{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m metadata
		err := dec.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		exports[m.ImportPath] = m.Export
		if m.ImportPath == "github.com/xianxu/tools/cmd/define" || m.ImportPath == "github.com/xianxu/tools/cmd/define/store" {
			targets = append(targets, m)
		}
	}
	if len(targets) != 2 {
		t.Fatalf("go list returned %d target packages", len(targets))
	}
	// Check Store first so the main package and the interfaces share type identity.
	slices.SortFunc(targets, func(a, b metadata) int { return strings.Compare(b.ImportPath, a.ImportPath) })
	fset := token.NewFileSet()
	imp := &quietGuardImporter{checked: map[string]*types.Package{}}
	imp.Importer = importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) { return os.Open(exports[path]) })
	var pkgs []*types.Package
	for _, m := range targets {
		var files []*ast.File
		for _, name := range m.CompiledGoFiles {
			if !filepath.IsAbs(name) {
				name = filepath.Join(m.Dir, name)
			}
			f, err := parser.ParseFile(fset, name, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, f)
		}
		cfg := types.Config{Importer: imp, IgnoreFuncBodies: true}
		p, err := cfg.Check(m.ImportPath, fset, files, nil)
		if err != nil {
			t.Fatal(err)
		}
		imp.checked[m.ImportPath] = p
		pkgs = append(pkgs, p)
	}
	stpkg := imp.checked["github.com/xianxu/tools/cmd/define/store"]
	return pkgs, fset, imp, stpkg.Scope().Lookup("Store").Type().Underlying().(*types.Interface), stpkg.Scope().Lookup("Quieter").Type().Underlying().(*types.Interface)
}
