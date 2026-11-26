// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
	"thingfabric.dev/go-fabric/device"

	hamapping "sukramj.dev/go-hamapping"
)

// renamed is an entity a user renamed in a UI.
type renamed struct {
	model.Basic
	name string
}

func userName(e model.Entity) (string, bool) {
	if r, ok := e.(*renamed); ok {
		return r.name, true
	}
	return "", false
}

func renamedEntity(name string) *renamed {
	e := entity("switch.kitchen_plug", hacatalog.PlatformSwitch, model.Description{Name: model.L("Plug")}).(*model.Basic)
	return &renamed{Basic: *e, name: name}
}

// TestNameVerbatim: a name the source marks verbatim is the endpoint's
// label as given, without the device name; a long one is cut to 32 bytes.
func TestNameVerbatim(t *testing.T) {
	opts := hamapping.Options{Name: userName}
	p, _ := hamapping.Map(testDevice, []model.Entity{renamedEntity("Coffee machine")}, opts)
	if got := p.Endpoints[0].Options.Name; got != "Coffee machine" {
		t.Errorf("label %q; want the verbatim name %q", got, "Coffee machine")
	}
	for _, long := range []string{
		strings.Repeat("x", 40),
		// 31 ASCII bytes, then a two-byte rune across the 32-byte limit:
		// the cut snaps back to byte 31, as device.ComposeNodeLabel cuts.
		strings.Repeat("x", 31) + "ä" + "yyy",
	} {
		p, _ = hamapping.Map(testDevice, []model.Entity{renamedEntity(long)}, opts)
		got := p.Endpoints[0].Options.Name
		if want := device.ComposeNodeLabel(long, ""); got != want {
			t.Errorf("label %q; want %q, the name cut as ComposeNodeLabel cuts", got, want)
		}
		if len(got) > 32 || !utf8.ValidString(got) || !strings.HasPrefix(long, got) {
			t.Errorf("label %q (%d bytes); want a valid UTF-8 prefix of at most 32 bytes", got, len(got))
		}
	}
}

// TestNameDefaultUnchanged: without the option, with a nil Name, an empty
// name or verbatim false, the label is composed as before.
func TestNameDefaultUnchanged(t *testing.T) {
	ents := func() []model.Entity { return []model.Entity{renamedEntity("Coffee machine")} }
	plain, _ := hamapping.Map(testDevice, ents())
	want := plain.Endpoints[0].Options.Name
	if want == "Coffee machine" || want == "" {
		t.Fatalf("default label %q; want the composed one", want)
	}
	for name, fn := range map[string]func(model.Entity) (string, bool){
		"nil":          nil,
		"empty":        func(model.Entity) (string, bool) { return "", true },
		"not verbatim": func(model.Entity) (string, bool) { return "Coffee machine", false },
	} {
		p, _ := hamapping.Map(testDevice, ents(), hamapping.Options{Name: fn})
		if got := p.Endpoints[0].Options.Name; got != want {
			t.Errorf("%s: label %q; want the default %q", name, got, want)
		}
	}
}
