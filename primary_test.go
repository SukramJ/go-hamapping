// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"fmt"
	"slices"
	"testing"

	"sukramj.dev/go-hamqtt/model"

	"sukramj.dev/go-hamapping"
)

// shape is what of a plan its input order must not change: per endpoint
// its key, stable key and riders.
func shape(p *hamapping.Plan) string {
	var out string
	for i := range p.Endpoints {
		ep := &p.Endpoints[i]
		out += fmt.Sprintf("%s=%v%v;", ep.Key, ep.Options.StableKey, ep.Riders)
	}
	return out
}

// permutations calls fn with every order of l.
func permutations(l []model.Entity, fn func([]model.Entity)) {
	var rec func(int)
	rec = func(k int) {
		if k == len(l) {
			fn(slices.Clone(l))
			return
		}
		for i := k; i < len(l); i++ {
			l[k], l[i] = l[i], l[k]
			rec(k + 1)
			l[k], l[i] = l[i], l[k]
		}
	}
	rec(0)
}

// TestPlanIsInputOrderIndependent: every order of the same entities
// yields the same endpoints, stable keys and riders; the primary is the
// non-riding entity with the smallest stable key, and the verdicts follow
// the input order.
func TestPlanIsInputOrderIndependent(t *testing.T) {
	in := []model.Entity{
		sensor("sensor.t", "temperature", "°C"), sensor("sensor.bat2", "battery", "%"),
		sensor("sensor.h", "humidity", "%"), sensor("sensor.bat1", "battery", "%"),
	}
	want := ""
	permutations(in, func(order []model.Entity) {
		p, v := hamapping.Map(testDevice, order)
		got := shape(p)
		if want == "" {
			want = got
		}
		if got != want {
			t.Errorf("order %v: plan %s; want %s", keys(order), got, want)
		}
		if p.Endpoints[0].Key != "sensor.h" || !slices.Equal(p.Endpoints[0].Riders, []string{"sensor.bat1", "sensor.bat2"}) {
			t.Errorf("order %v: primary %s riders %v; want sensor.h with [sensor.bat1 sensor.bat2]", keys(order), p.Endpoints[0].Key, p.Endpoints[0].Riders)
		}
		for i := range v {
			if v[i].Key != order[i].Key() {
				t.Errorf("order %v: verdict %d is %s; want the input order", keys(order), i, v[i].Key)
			}
		}
	})
	// Without a primary the riders are endpoints of their own, also in
	// stable-key order.
	permutations([]model.Entity{sensor("sensor.bat2", "battery", "%"), sensor("sensor.bat1", "battery", "%")}, func(order []model.Entity) {
		p, _ := hamapping.Map(testDevice, order)
		if len(p.Endpoints) != 2 || p.Endpoints[0].Key != "sensor.bat1" || p.Endpoints[1].Key != "sensor.bat2" {
			t.Errorf("order %v: plan %s; want sensor.bat1, sensor.bat2", keys(order), shape(p))
		}
	})
}

// TestPrimarySurvivesARename: the primary follows the stable key, not the
// Key, so renaming the primary's entity to a key that sorts last keeps it
// primary while Options.StableKey keeps its identity.
func TestPrimarySurvivesARename(t *testing.T) {
	ids := map[string]string{"sensor.a": "2", "sensor.b": "1", "sensor.zzz": "1"}
	opts := hamapping.Options{StableKey: func(e model.Entity) string { return ids[e.Key()] }}
	bat := sensor("sensor.bat", "battery", "%")
	before, _ := hamapping.Map(testDevice, []model.Entity{sensor("sensor.a", "temperature", "°C"), sensor("sensor.b", "humidity", "%"), bat}, opts)
	after, _ := hamapping.Map(testDevice, []model.Entity{sensor("sensor.a", "temperature", "°C"), sensor("sensor.zzz", "humidity", "%"), bat}, opts)
	b, a := before.Endpoints[0], after.Endpoints[0]
	if b.Key != "sensor.b" || a.Key != "sensor.zzz" || a.Options.StableKey != b.Options.StableKey {
		t.Errorf("primary %s (%v) before, %s (%v) after the rename; want the renamed entity both times", b.Key, b.Options.StableKey, a.Key, a.Options.StableKey)
	}
	if !slices.Equal(a.Riders, []string{"sensor.bat"}) {
		t.Errorf("riders after the rename %v; want [sensor.bat]", a.Riders)
	}
}

func keys(l []model.Entity) []string {
	out := make([]string, len(l))
	for i, e := range l {
		out[i] = e.Key()
	}
	return out
}
