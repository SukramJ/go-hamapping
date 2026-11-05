// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

// TestRowsNameGoFabricTypes holds every row against go-fabric's device
// kit: its type is a generated device type, and every cluster is one the
// type lists, or PowerSource, which BridgedNode lists for a bridged device.
func TestRowsNameGoFabricTypes(t *testing.T) {
	for _, r := range hamapping.Rows() {
		if r.Type == nil {
			if r.State != eligibility.StateUnmappable || len(r.Clusters) > 0 {
				t.Errorf("%s: no type but %v with %d clusters", r.ID, r.State, len(r.Clusters))
			}
			continue
		}
		if got, ok := device.Lookup(uint32(r.Type.ID)); !ok || got != r.Type {
			t.Errorf("%s: type %s is not go-fabric's", r.ID, r.Type.Name)
		}
		parts := make([]struct {
			typ *device.Type
			cs  []hamapping.Cluster
		}, 1, 1+len(r.Parts))
		parts[0].typ, parts[0].cs = r.Type, r.Clusters
		for _, p := range r.Parts {
			parts = append(parts, struct {
				typ *device.Type
				cs  []hamapping.Cluster
			}{p.Type, p.Clusters})
		}
		for _, p := range parts {
			for _, c := range p.cs {
				_, own := p.typ.Server(c.ID)
				br, bridged := device.BridgedNodeType().Server(c.ID)
				if !own && (!bridged || br.Conformance != device.ConditionBridgedPowerSourceInfo) {
					t.Errorf("%s: %s does not list %s", r.ID, p.typ.Name, c.Name)
				}
			}
		}
		if len(r.Cites) == 0 {
			t.Errorf("%s: no cites", r.ID)
		}
	}
}

// TestDeviceClassesHaveOneRow: every device class the catalog lists for
// sensor and binary_sensor (go-ha-catalog data/device_classes.json) has
// exactly one row, and every class a row names is the catalog's.
func TestDeviceClassesHaveOneRow(t *testing.T) {
	classes, err := hacatalog.LoadDeviceClasses()
	if err != nil {
		t.Fatal(err)
	}
	for _, platform := range []string{"sensor", "binary_sensor", "cover", "switch"} {
		count := map[string]int{}
		for _, r := range hamapping.Rows() {
			if r.Platform != platform {
				continue
			}
			for _, c := range r.When.DeviceClass {
				count[c]++
			}
		}
		for c := range count {
			if c != "" && !slices.Contains(classes[platform], c) {
				t.Errorf("%s: the rows name device_class %q, the catalog has none", platform, c)
			}
		}
		if platform != "sensor" && platform != "binary_sensor" {
			continue
		}
		for _, c := range append(slices.Clone(classes[platform]), "") {
			if count[c] != 1 {
				t.Errorf("%s: device_class %q has %d rows, want 1", platform, c, count[c])
			}
		}
	}
}

// TestSensorUnitsAreTheCatalogs: a mapped sensor row converts exactly the
// units the catalog allows its device classes (data/sensor.json
// device_class_units; its null, an entity without a unit, is the units
// key ""), so a unit the catalog adds fails here.
func TestSensorUnitsAreTheCatalogs(t *testing.T) {
	cat, err := hacatalog.LoadSensor()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range hamapping.Rows() {
		if r.Platform != "sensor" || r.Type == nil {
			continue
		}
		var tables []map[string]hamapping.Linear
		for _, c := range r.Clusters {
			for _, a := range c.Attributes {
				if a.Value.Units != nil {
					tables = append(tables, a.Value.Units)
				}
			}
		}
		if m := r.RideMeasurement; m != nil && m.Value.Units != nil {
			tables = append(tables, m.Value.Units)
		}
		for _, class := range r.When.DeviceClass {
			// The catalog's null (a sensor without a unit) is the key "".
			want := slices.Clone(cat.DeviceClassUnits[class])
			for i, u := range want {
				if u == "null" {
					want[i] = ""
				}
			}
			slices.Sort(want)
			if len(tables) == 0 {
				if len(want) > 1 {
					t.Errorf("%s: %s has units %v and the row converts none", r.ID, class, want)
				}
				continue
			}
			for _, units := range tables {
				got := slices.Sorted(maps.Keys(units))
				if !slices.Equal(got, want) {
					t.Errorf("%s: %s converts %v, the catalog's units are %v", r.ID, class, got, want)
				}
			}
		}
	}
}

// TestPredicatesNameTheCatalog: every feature flag and Extra key a
// predicate reads is one the catalog gives the platform.
func TestPredicatesNameTheCatalog(t *testing.T) {
	flags, err := hacatalog.LoadFeatures()
	if err != nil {
		t.Fatal(err)
	}
	mqtt, err := hacatalog.LoadMQTT()
	if err != nil {
		t.Fatal(err)
	}
	keys := func(platform string) map[string]bool {
		out := map[string]bool{"supported_features": true}
		schema := mqtt.Platforms[platform]
		for k := range schema.Keys {
			out[k] = true
		}
		for _, v := range schema.Variants {
			for k := range v.Keys {
				out[k] = true
			}
		}
		return out
	}
	check := func(id, platform string, w *hamapping.When) {
		if w == nil {
			return
		}
		if w.Features != nil {
			for _, f := range slices.Concat(w.Features.All, w.Features.Any, w.Features.None) {
				if _, ok := flags[platform][f]; !ok {
					t.Errorf("%s: feature %s is not the catalog's for %s", id, f, platform)
				}
			}
		}
		k := keys(platform)
		for key := range w.Extra {
			if !k[key] {
				t.Errorf("%s: extra key %s is not the catalog's for %s", id, key, platform)
			}
		}
	}
	for _, r := range hamapping.Rows() {
		check(r.ID, r.Platform, &r.When)
		for _, c := range r.Caveats {
			check(r.ID, r.Platform, &c.When)
		}
		for _, c := range r.Clusters {
			for _, f := range c.Features {
				check(r.ID, r.Platform, f.When)
			}
			for _, a := range slices.Concat(c.Attributes, c.Initial) {
				check(r.ID, r.Platform, a.When)
				if key, ok := strings.CutPrefix(a.Value.From, "extra."); ok && !keys(r.Platform)[key] {
					t.Errorf("%s: %s reads extra.%s, not the catalog's", r.ID, a.Name, key)
				}
			}
			for _, cmd := range c.Commands {
				check(r.ID, r.Platform, cmd.When)
			}
		}
	}
}

// TestPredicates holds which row decides for entities across the
// platforms.
func TestPredicates(t *testing.T) {
	modes := func(m ...string) model.Description {
		return model.Description{Extra: map[string]any{"supported_color_modes": m}}
	}
	for _, c := range []struct {
		name     string
		platform hacatalog.Platform
		desc     model.Description
		row      string
		state    eligibility.State
	}{
		{"light without modes", "light", model.Description{}, "light/onoff", eligibility.StateMappable},
		{"onoff", "light", modes("onoff"), "light/onoff", eligibility.StateMappable},
		{"brightness", "light", modes("brightness"), "light/dimmable", eligibility.StateMappable},
		{"white", "light", modes("white"), "light/dimmable", eligibility.StateMappable},
		{"color_temp", "light", modes("color_temp"), "light/color_temperature", eligibility.StateMappable},
		{"hs", "light", modes("hs"), "light/extended_color", eligibility.StateMappable},
		{"xy and ct", "light", modes("xy", "color_temp"), "light/extended_color", eligibility.StateMappable},
		{"rgbw", "light", modes("rgbw"), "light/rgb", eligibility.StatePartial},
		{"switch", "switch", model.Description{}, "switch/plug", eligibility.StateMappable},
		{"outlet", "switch", model.Description{DeviceClass: "outlet"}, "switch/plug", eligibility.StateMappable},
		{"sensor without class", "sensor", model.Description{}, "sensor/no_device_class", eligibility.StateUnmappable},
		{"co2", "sensor", model.Description{DeviceClass: "carbon_dioxide", Unit: "ppm"}, "sensor/carbon_dioxide", eligibility.StateMappable},
		{"aqi", "sensor", model.Description{DeviceClass: "aqi"}, "sensor/no_device_type", eligibility.StateUnmappable},
		{"temperature in a unit no row converts", "sensor", model.Description{DeviceClass: "temperature", Unit: "°R"}, "sensor/temperature", eligibility.StateUnmappable},
		{"window", "binary_sensor", model.Description{DeviceClass: "window"}, "binary_sensor/contact", eligibility.StateMappable},
		{"presence", "binary_sensor", model.Description{DeviceClass: "presence"}, "binary_sensor/occupancy", eligibility.StateMappable},
		{"tamper", "binary_sensor", model.Description{DeviceClass: "tamper"}, "binary_sensor/no_device_type", eligibility.StateUnmappable},
		{"garage door", "cover", model.Description{DeviceClass: "garage"}, "cover/window_covering", eligibility.StatePartial},
		{"blind", "cover", model.Description{DeviceClass: "blind"}, "cover/window_covering", eligibility.StateMappable},
		{"climate with the catalog's modes", "climate", model.Description{}, "climate/thermostat", eligibility.StatePartial},
		{"heat only", "climate", model.Description{Extra: map[string]any{"modes": []any{"off", "heat"}}}, "climate/thermostat", eligibility.StateMappable},
		{"presets", "climate", model.Description{Extra: map[string]any{"modes": []string{"heat"}, "preset_modes": []string{"eco"}}}, "climate/thermostat", eligibility.StatePartial},
		{"lock", "lock", model.Description{}, "lock/door_lock", eligibility.StateMappable},
		{"fan", "fan", model.Description{}, "fan/fan", eligibility.StateMappable},
		{"vacuum without start", "vacuum", model.Description{}, "vacuum/unmapped", eligibility.StateUnmappable},
		{"vacuum", "vacuum", model.Description{Extra: map[string]any{"supported_features": 8192}}, "vacuum/robotic_vacuum_cleaner", eligibility.StatePartial},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, v := hamapping.Map(testDevice, []model.Entity{entity("e", c.platform, c.desc)})
			if v[0].Row != c.row || v[0].State != c.state {
				t.Fatalf("row %q %v (%s), want %q %v", v[0].Row, v[0].State, v[0].Reason, c.row, c.state)
			}
			if c.state != eligibility.StateMappable && v[0].Reason == "" {
				t.Fatal("no reason")
			}
			if c.state == eligibility.StateUnmappable && (v[0].DeviceType != 0 || len(v[0].Clusters) != 0) {
				t.Fatalf("unmappable with %+v", v[0])
			}
		})
	}
}
