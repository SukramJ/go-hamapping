// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"testing"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
	"thingfabric.dev/go-fabric/device"

	hamapping "sukramj.dev/go-hamapping"
)

// registryEntity is an entity whose source keeps an identity across a
// rename of its key, as a Home Assistant registry entry id does.
type registryEntity struct {
	model.Basic
	id string
}

func registry(key, id string) *registryEntity {
	e := entity(key, hacatalog.PlatformSwitch, model.Description{Name: model.L("Plug")}).(*model.Basic)
	return &registryEntity{Basic: *e, id: id}
}

func stableKey(e model.Entity) string {
	if r, ok := e.(*registryEntity); ok {
		return r.id
	}
	return ""
}

// TestStableKeySurvivesARename: with Options.StableKey set, a renamed
// entity (a new Key) keeps its endpoint's stable key; the Key still
// addresses the endpoint and the verdict.
func TestStableKeySurvivesARename(t *testing.T) {
	opts := hamapping.Options{StableKey: stableKey}
	before, _ := hamapping.Map(testDevice, []model.Entity{registry("switch.plug", "0f3c")}, opts)
	after, v := hamapping.Map(testDevice, []model.Entity{registry("switch.kitchen_plug", "0f3c")}, opts)
	if len(before.Endpoints) != 1 || len(after.Endpoints) != 1 {
		t.Fatalf("endpoints %d, %d; want 1, 1", len(before.Endpoints), len(after.Endpoints))
	}
	b, a := before.Endpoints[0], after.Endpoints[0]
	if a.Options.StableKey != b.Options.StableKey {
		t.Errorf("stable key %v after the rename, %v before", a.Options.StableKey, b.Options.StableKey)
	}
	if want := testDevice.UID() + "/0f3c"; a.Options.StableKey != device.StringKey(want) {
		t.Errorf("stable key %v; want %s (the device UID, \"/\" and the identity)", a.Options.StableKey, want)
	}
	if a.Key != "switch.kitchen_plug" || a.Options.ChannelAddress != "switch.kitchen_plug" || v[0].Key != "switch.kitchen_plug" {
		t.Errorf("key %q, channel %q, verdict %q; want the new entity id", a.Key, a.Options.ChannelAddress, v[0].Key)
	}
}

// TestStableKeyDefaultsToKey: without the option, and for an entity the
// option has no identity for, the stable key follows the Key.
func TestStableKeyDefaultsToKey(t *testing.T) {
	plain, _ := hamapping.Map(testDevice, []model.Entity{registry("switch.a", "1")})
	empty, _ := hamapping.Map(testDevice, []model.Entity{entity("switch.a", hacatalog.PlatformSwitch, model.Description{})},
		hamapping.Options{StableKey: stableKey})
	renamed, _ := hamapping.Map(testDevice, []model.Entity{registry("switch.b", "1")})
	if plain.Endpoints[0].Options.StableKey != empty.Endpoints[0].Options.StableKey {
		t.Errorf("default %v, empty identity %v; want equal", plain.Endpoints[0].Options.StableKey, empty.Endpoints[0].Options.StableKey)
	}
	if plain.Endpoints[0].Options.StableKey == renamed.Endpoints[0].Options.StableKey {
		t.Errorf("without the option a rename kept the stable key %v", plain.Endpoints[0].Options.StableKey)
	}
}

// TestExportedStableKeyIsMapsKey: StableKey returns the stable key Map
// assigns the endpoint, with and without Options.StableKey (an identity,
// and an empty one), and with and without the device's UID.
func TestExportedStableKeyIsMapsKey(t *testing.T) {
	noUID := &model.Device{Name: model.L("No UID")}
	for _, dev := range []*model.Device{testDevice, noUID, nil} {
		for _, opts := range [][]hamapping.Options{nil, {{StableKey: stableKey}}, {{StableKey: stableKey}, {}}} {
			for _, e := range []model.Entity{
				registry("switch.plug", "0f3c"),
				entity("switch.plain", hacatalog.PlatformSwitch, model.Description{}),
			} {
				p, _ := hamapping.Map(dev, []model.Entity{e}, opts...)
				if len(p.Endpoints) != 1 {
					t.Fatalf("%s: %d endpoints; want 1", e.Key(), len(p.Endpoints))
				}
				got := hamapping.StableKey(dev, e, opts...)
				if p.Endpoints[0].Options.StableKey != got {
					t.Errorf("device %v, %d options, %s: StableKey %q, Map %v", dev.UID(), len(opts), e.Key(), got, p.Endpoints[0].Options.StableKey)
				}
			}
		}
	}
	if got, want := hamapping.StableKey(testDevice, registry("switch.plug", "0f3c"), hamapping.Options{StableKey: stableKey}),
		device.StringKey(testDevice.UID()+"/0f3c"); got != want {
		t.Errorf("StableKey %q; want %q", got, want)
	}
	if got := hamapping.StableKey(nil, registry("switch.plug", "0f3c")); got != "switch.plug" {
		t.Errorf("StableKey without a device %q; want the Key", got)
	}
}
