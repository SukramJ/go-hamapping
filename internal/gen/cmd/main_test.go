// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"sukramj.dev/go-hamapping/internal/gen"
)

// TestRun writes the repository's output into a copy and finds it equal
// to the committed files.
func TestRun(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(filepath.Join(root, "rows"), os.DirFS("../../../rows")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join("..", "..", "..", gen.Docs))
	if err != nil {
		t.Fatal(err)
	}
	// As a Windows checkout with autocrlf has it.
	page = bytes.ReplaceAll(bytes.ReplaceAll(page, []byte("\r\n"), []byte("\n")), []byte("\n"), []byte("\r\n"))
	if err := os.WriteFile(filepath.Join(root, gen.Docs), page, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{gen.Output, gen.Docs} {
		got, _ := os.ReadFile(filepath.Join(root, name))
		want, _ := os.ReadFile(filepath.Join("..", "..", "..", name))
		lf := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
		if !bytes.Equal(lf(got), lf(want)) {
			t.Errorf("%s differs from the committed one", name)
		}
	}
	if err := run(t.TempDir()); err == nil {
		t.Fatal("ran without rows")
	}
}
