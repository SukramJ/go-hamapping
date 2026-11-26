// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

// Command cmd writes rows_gen.go from rows/*.yaml. Run it from the module
// root: go run ./internal/gen/cmd (or go generate ./...).
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sukramj.dev/go-hamapping/internal/gen"
	"sukramj.dev/go-hamapping/internal/rowfile"
)

func main() {
	root := flag.String("root", ".", "the module root")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	files, err := rowfile.Load(filepath.Join(root, "rows"))
	if err != nil {
		return err
	}
	src, err := gen.GenerateFiles(files)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, gen.Output), src, 0o644); err != nil { //nolint:gosec // a source file, world-readable
		return err
	}
	docs := filepath.Join(root, gen.Docs)
	page, err := os.ReadFile(docs) //nolint:gosec // the repository's page
	if err != nil {
		return err
	}
	// A checkout that converted line endings (git autocrlf) writes LF again.
	page = bytes.ReplaceAll(page, []byte("\r\n"), []byte("\n"))
	page, err = gen.Page(page, files)
	if err != nil {
		return err
	}
	return os.WriteFile(docs, page, 0o644) //nolint:gosec // a documentation page, world-readable
}
