// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"bytes"
	"os"
	"testing"

	"sukramj.dev/go-hamapping/internal/gen"
	"sukramj.dev/go-hamapping/internal/rowfile"
)

// TestGeneratedRowsAreCurrent fails when rows_gen.go (the rows and the
// method constants) or the row and key tables of docs/rows.md differ from what internal/gen writes from rows/*.yaml —
// a row changed without `make generate`, a hand edit of the Go form, or a
// go-fabric bump that moves a generated name.
func TestGeneratedRowsAreCurrent(t *testing.T) {
	files, err := rowfile.Load("rows")
	if err != nil {
		t.Fatal(err)
	}
	want, err := gen.GenerateFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(gen.Output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
		t.Errorf("%s is stale: run make generate and commit the output", gen.Output)
	}
	page, err := os.ReadFile(gen.Docs)
	if err != nil {
		t.Fatal(err)
	}
	page = bytes.ReplaceAll(page, []byte("\r\n"), []byte("\n"))
	spliced, err := gen.Page(page, files)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(page, spliced) {
		t.Errorf("%s: the row table or the key table is stale: run make generate and commit the output", gen.Docs)
	}
}
