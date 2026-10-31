// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

// Package arch holds the architecture guard of the module (brief H1 hard
// rule 1): go-fabric is used through device, contract, schema,
// eligibility and the generated <cluster>def packages (cluster/spec/<name>)
// only — no node, no cluster/spec itself, nothing below. Tests may use
// go-fabric's in-process test kit besides. The counterpart of go-fabric's
// script/archcheck and of go-fabric-server's internal/arch, as a test.
package arch

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const goFabric = "github.com/SukramJ/go-fabric/"

// facade are the go-fabric packages non-test code may import.
var facade = map[string]bool{
	"device":      true,
	"contract":    true,
	"schema":      true,
	"eligibility": true,
}

// testOnly are the go-fabric packages a test may import besides: the
// in-process site and the node it serves, the controller's path builders,
// the scalar encoders of a write (docs/guides/testing-in-process.md) and
// the camera kit's server types a test asserts a device's servers are.
var testOnly = map[string]bool{
	"fabrictest":        true,
	"node":              true,
	"controller/client": true,
	"cluster/spec":      true,
	"cluster/camera":    true,
}

// isDef reports whether dep is a generated <cluster>def package.
func isDef(dep string) bool {
	name, ok := strings.CutPrefix(dep, "cluster/spec/")
	return ok && name != "" && !strings.Contains(name, "/")
}

func allowed(dep string, isTest bool) bool {
	return facade[dep] || isDef(dep) || isTest && testOnly[dep]
}

func TestImports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	files := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "testdata" || strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		isTest := strings.HasSuffix(path, "_test.go")
		rel, _ := filepath.Rel(root, path)
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if dep, ok := strings.CutPrefix(p, goFabric); ok && !allowed(dep, isTest) {
				t.Errorf("%s imports %s: below go-fabric's facade", filepath.ToSlash(rel), p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 15 {
		t.Fatalf("walked %d files: the root is wrong", files)
	}
}

// TestGuardRefuses holds the guard itself.
func TestGuardRefuses(t *testing.T) {
	for _, c := range []struct {
		dep  string
		test bool
	}{
		{"node", false},
		{"cluster/spec", false},
		{"endpoint", false},
		{"bridge", false},
		{"im", false},
		{"fabrictest", false},
		{"cluster/spec/onoff/internal", false},
		{"cluster/onoff", false},
		{"cluster/camera", false},
		{"controller/client", false},
		{"tlv", true},
		{"endpoint", true},
	} {
		if allowed(c.dep, c.test) {
			t.Errorf("%s (test %v) is allowed", c.dep, c.test)
		}
	}
	for _, c := range []struct {
		dep  string
		test bool
	}{
		{"device", false},
		{"contract", false},
		{"eligibility", false},
		{"schema", false},
		{"cluster/spec/onoff", false},
		{"fabrictest", true},
		{"node", true},
		{"cluster/spec", true},
		{"cluster/camera", true},
	} {
		if !allowed(c.dep, c.test) {
			t.Errorf("%s (test %v) is refused", c.dep, c.test)
		}
	}
}
