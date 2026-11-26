// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"slices"
	"strings"
	"testing"

	"sukramj.dev/go-hamapping/internal/gen"
	"sukramj.dev/go-hamapping/internal/rowfile"
)

// TestEveryRowMethodHasAConstant: the method constants are exactly the
// methods the rows' commands and options deliver.
func TestEveryRowMethodHasAConstant(t *testing.T) {
	var want []string
	for _, r := range Rows() {
		cs := slices.Clone(r.Clusters)
		for _, p := range r.Parts {
			cs = append(cs, p.Clusters...)
		}
		for _, c := range cs {
			for _, cmd := range c.Commands {
				if cmd.Method != "" && !slices.Contains(want, cmd.Method) {
					want = append(want, cmd.Method)
				}
			}
			if c.Options != nil && !slices.Contains(want, c.Options.Method) {
				want = append(want, c.Options.Method)
			}
		}
	}
	slices.Sort(want)
	if !slices.Equal(methods, want) {
		t.Errorf("method constants %v; the rows deliver %v", methods, want)
	}
	files, err := rowfile.Load("rows")
	if err != nil {
		t.Fatal(err)
	}
	if got := gen.Methods(files); !slices.Equal(got, want) {
		t.Errorf("gen.Methods %v; want %v", got, want)
	}
}

// TestKeyTableDefaultIsTheReadersDefault: the key table's catalog default
// column says what extra() reads for a key the source leaves out.
func TestKeyTableDefaultIsTheReadersDefault(t *testing.T) {
	files, err := rowfile.Load("rows")
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range gen.Keys(files) {
		key, ok := strings.CutPrefix(k.Name, prefixEx)
		if !ok {
			continue
		}
		if _, def := catalogDefault(k.Platform, key); def != k.Default {
			t.Errorf("%s %s: table default %v, catalogDefault %v", k.Platform, k.Name, k.Default, def)
		}
	}
}
