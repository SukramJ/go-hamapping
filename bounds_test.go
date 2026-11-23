// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

// TestSelectLabelsCutByBytes: go-fabric bounds a string by its UTF-8
// bytes since v0.24.0, so a select's option labels and its description
// are cut to 64 bytes between two characters; a non-ASCII label within 64
// UTF-16 units but over 64 bytes no longer fails the build.
func TestSelectLabelsCutByBytes(t *testing.T) {
	long := strings.Repeat("ä", 40) // 40 UTF-16 units, 80 bytes
	e := entity("sel", hacatalog.PlatformSelect, model.Description{
		Name: model.L(long), Options: &model.Enum{Codes: []string{long, "x" + long}},
	})
	p, _ := hamapping.Map(testDevice, []model.Entity{e})
	if len(p.Endpoints) != 1 {
		t.Fatalf("endpoints %+v", p.Endpoints)
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatalf("Build: %v", err)
	}
}

// TestVendorCutTo32Bytes: go-fabric refuses an Options.Vendor over 32
// bytes since v0.23.0 instead of cutting it, so the plan cuts the
// manufacturer between two characters; a long one no longer fails the
// build.
func TestVendorCutTo32Bytes(t *testing.T) {
	for _, m := range []string{
		strings.Repeat("v", 40),
		strings.Repeat("v", 31) + "ä" + "yyy", // a two-byte rune across the limit
	} {
		dev := *testDevice
		dev.Manufacturer = m
		e := entity("switch.plug", hacatalog.PlatformSwitch, model.Description{Name: model.L("Plug")})
		p, _ := hamapping.Map(&dev, []model.Entity{e})
		got := p.Endpoints[0].Options.Vendor
		if len(got) > 32 || !utf8.ValidString(got) || !strings.HasPrefix(m, got) || len(got) < 31 {
			t.Errorf("vendor %q (%d bytes); want the manufacturer cut to a valid prefix of at most 32 bytes", got, len(got))
		}
		if _, err := p.Build(context.Background()); err != nil {
			t.Fatalf("Build with manufacturer %q: %v", m, err)
		}
	}
}
