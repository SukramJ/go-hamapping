// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"slices"
	"testing"

	onoffdef "github.com/SukramJ/go-fabric/cluster/spec/onoff"
	switchdef "github.com/SukramJ/go-fabric/cluster/spec/switchcluster"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-fabric/fabrictest"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

func TestButton(t *testing.T) {
	r := serve(t, entity("btn", hacatalog.PlatformButton, model.Description{Name: model.L("Restart")}))
	wantType(t, r, "btn", device.OnOffPlugInUnitType(), eligibility.StateMappable)
	r.apply("btn", value("2026-10-31T10:00:00+00:00"))
	r.invoke("btn", onoffdef.ClusterID, onoffdef.CmdOn, nil)
	expect(t, r.command(), hamapping.MethodPress, "PRESS")
	r.apply("btn", value("2026-10-31T10:00:05+00:00"))
	if st, _ := onoffdef.StateFrom(r.values("btn", onoffdef.ClusterID)); st.OnOff {
		t.Fatal("OnOff true after a press, want false")
	}
	r.noMore()
}

func TestScript(t *testing.T) {
	actionPlug(t, "script", "on", hamapping.MethodTurnOn, "RUN")
}

func TestScene(t *testing.T) {
	actionPlug(t, hacatalog.PlatformScene, "2026-11-08T10:00:00+00:00", hamapping.MethodTurnOn, "ACTIVATE")
}

func TestAutomation(t *testing.T) {
	actionPlug(t, "automation", "on", hamapping.MethodTrigger, "TRIGGER")
}

// actionPlug checks an action target served as an OnOffPlugInUnit: OnOff
// stays false whatever the entity's state, and On delivers method with
// payload.
func actionPlug(t *testing.T, p hacatalog.Platform, state any, method, payload string) {
	t.Helper()
	r := serve(t, entity("act", p, model.Description{Name: model.L("Good night")}))
	wantType(t, r, "act", device.OnOffPlugInUnitType(), eligibility.StateMappable)
	r.apply("act", value(state))
	if st, _ := onoffdef.StateFrom(r.values("act", onoffdef.ClusterID)); st.OnOff {
		t.Fatalf("OnOff true after state %v, want false", state)
	}
	r.invoke("act", onoffdef.ClusterID, onoffdef.CmdOn, nil)
	expect(t, r.command(), method, payload)
	if st, _ := onoffdef.StateFrom(r.values("act", onoffdef.ClusterID)); st.OnOff {
		t.Fatal("OnOff true after On, want false")
	}
	r.noMore()
}

// switchEvents are the Switch events of the entity's endpoint, by id, in
// report order.
func (r *rig) switchEvents(key string) []uint32 {
	r.t.Helper()
	ep := r.ep(key)
	got, err := fabrictest.Events(r.ctx, r.pair,
		client.EventAt(ep, switchdef.ClusterID, switchdef.EventInitialPress),
		client.EventAt(ep, switchdef.ClusterID, switchdef.EventShortRelease))
	if err != nil {
		r.t.Fatal(err)
	}
	slices.SortFunc(got, func(a, b fabrictest.EventData) int { return int(a.Number) - int(b.Number) })
	ids := make([]uint32, len(got))
	for i := range got {
		ids[i] = got[i].Path.Event
	}
	return ids
}

func TestEvent(t *testing.T) {
	r := serve(t, entity("ev", hacatalog.PlatformEvent, model.Description{Name: model.L("Doorbell")}))
	wantType(t, r, "ev", device.GenericSwitchType(), eligibility.StatePartial)
	st, _ := switchdef.StateFrom(r.values("ev", switchdef.ClusterID))
	if st.NumberOfPositions != 2 || st.CurrentPosition != 0 {
		t.Fatalf("switch %+v", st)
	}
	// The first value is the baseline: an event before the bridge started
	// is no press.
	for _, p := range r.plan.Apply(r.ctx, "ev", value("2026-10-31T10:00:00.000+00:00")) {
		if p.Events != nil || p.Err != nil {
			t.Fatalf("baseline emitted %+v", p)
		}
	}
	if got := r.switchEvents("ev"); len(got) != 0 {
		t.Fatalf("events after the baseline %v", got)
	}
	var fired []string
	for _, p := range r.plan.Apply(r.ctx, "ev", value("2026-10-31T10:00:01.000+00:00")) {
		if p.Err != nil {
			t.Fatal(p.Err)
		}
		fired = append(fired, p.Events...)
	}
	if !slices.Equal(fired, []string{"InitialPress", "ShortRelease"}) {
		t.Fatalf("Patch.Events %v", fired)
	}
	want := []uint32{switchdef.EventInitialPress, switchdef.EventShortRelease}
	if got := r.switchEvents("ev"); !slices.Equal(got, want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	// The same value again, and an unavailable period, emit nothing; the
	// value after it is a new baseline.
	r.apply("ev", value("2026-10-31T10:00:01.000+00:00"))
	r.apply("ev", model.State{Available: false})
	r.apply("ev", value("2026-10-31T10:00:02.000+00:00"))
	r.apply("ev", value("unknown"))
	r.apply("ev", value("2026-10-31T10:00:03.000+00:00"))
	if got := r.switchEvents("ev"); len(got) != 2 {
		t.Fatalf("events %v after repeats and gaps, want the first two only", got)
	}
	r.apply("ev", value("2026-10-31T10:00:04.000+00:00"))
	if got := r.switchEvents("ev"); len(got) != 4 {
		t.Fatalf("events %v, want a second press", got)
	}
	r.noMore()
}
