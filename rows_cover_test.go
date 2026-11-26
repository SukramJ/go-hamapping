// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"fmt"
	"testing"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
	windowcoveringdef "thingfabric.dev/go-fabric/cluster/spec/windowcovering"
	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/eligibility"
	"thingfabric.dev/go-fabric/fabrictest"

	"sukramj.dev/go-hamapping"
)

// coverClasses are Home Assistant's cover device classes and none.
var coverClasses = []string{"", "awning", "blind", "curtain", "damper", "door", "garage", "gate", "shade", "shutter", "window"}

func cover(class string, features int) model.Entity {
	return entity("cover", hacatalog.PlatformCover, model.Description{
		DeviceClass: model.DeviceClass(class), Extra: map[string]any{"supported_features": features},
	})
}

// TestCoverMappableBuilds is the standing guard of HR-P2-adpt-33: every
// cover Map calls mappable (or partial) builds against go-fabric, for
// every device class and every set of the eight feature flags the row
// reads.
func TestCoverMappableBuilds(t *testing.T) {
	bad := 0
	for _, class := range coverClasses {
		for f := range 256 {
			p, v := hamapping.Map(testDevice, []model.Entity{cover(class, f)})
			if v[0].State == eligibility.StateUnmappable {
				t.Errorf("class %q features %d: unmappable: %s", class, f, v[0].Reason)
				continue
			}
			if _, err := p.Build(context.Background()); err != nil {
				bad++
				if bad <= 5 {
					t.Errorf("class %q features %d: mappable, refused: %v", class, f, err)
				}
			}
			p.Close()
		}
	}
	if bad > 0 {
		t.Errorf("%d cover shapes are mappable and do not build", bad)
	}
}

// TestCoverTypeByTilt: Type and EndProductType follow the device class and,
// with tilt, take the LF&TL members (fact note 2026-11-24 §3); a window is
// Rollershade / RollerShade with Lift alone, TiltBlindLift / Unknown with
// Tilt (fact note 2026-11-26 Q4a).
func TestCoverTypeByTilt(t *testing.T) {
	type te struct {
		typ windowcoveringdef.TypeEnum
		end windowcoveringdef.EndProductTypeEnum
	}
	unknown := te{windowcoveringdef.TypeUnknown, windowcoveringdef.EndProductTypeUnknown}
	lift := map[string]te{
		"":        {windowcoveringdef.TypeRollershade, windowcoveringdef.EndProductTypeRollerShade},
		"blind":   {windowcoveringdef.TypeRollershade, windowcoveringdef.EndProductTypeRollerShade},
		"shade":   {windowcoveringdef.TypeRollershade, windowcoveringdef.EndProductTypeRollerShade},
		"shutter": {windowcoveringdef.TypeShutter, windowcoveringdef.EndProductTypeRollerShutter},
		"curtain": {windowcoveringdef.TypeDrapery, windowcoveringdef.EndProductTypeCentralCurtain},
		"awning":  {windowcoveringdef.TypeAwning, windowcoveringdef.EndProductTypeAwningTerracePatio},
		"window":  {windowcoveringdef.TypeRollershade, windowcoveringdef.EndProductTypeRollerShade},
		"garage":  unknown, "gate": unknown, "door": unknown, "damper": unknown,
	}
	tilt := map[string]te{
		"":        {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeInteriorBlind},
		"blind":   {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeInteriorBlind},
		"shade":   {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeSheerShade},
		"shutter": {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeExteriorVenetianBlind},
		"curtain": {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeVerticalBlindStripCurtain},
		"awning":  {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeUnknown},
		"window":  {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeUnknown},
		"garage":  unknown, "gate": unknown, "door": unknown, "damper": unknown,
	}
	// 15: lift only; 0: neither; 255: lift+tilt; 240: tilt only (SwitchBot
	// Blind Tilt); 48: binary tilt only.
	for _, f := range []int{15, 0, 255, 240, 48} {
		want := lift
		if f&(16|32|128) != 0 {
			want = tilt
		}
		for _, class := range coverClasses {
			t.Run(fmt.Sprintf("%s/%d", class, f), func(t *testing.T) {
				r := serve(t, cover(class, f))
				wantType(t, r, "cover", device.WindowCoveringType(), r.verdict("cover").State)
				st, _ := windowcoveringdef.StateFrom(r.values("cover", windowcoveringdef.ClusterID))
				if w := want[class]; st.Type != w.typ || st.EndProductType != w.end {
					t.Fatalf("Type %v EndProductType %v, want %v %v", st.Type, st.EndProductType, w.typ, w.end)
				}
			})
		}
	}
}

func (r *rig) cover(cmd uint32, fields ...any) {
	r.t.Helper()
	switch cmd {
	case windowcoveringdef.CmdGoToLiftPercentage:
		r.invoke("cover", windowcoveringdef.ClusterID, cmd, windowcoveringdef.GoToLiftPercentageRequest{LiftPercent100thsValue: fields[0].(uint16)})
	case windowcoveringdef.CmdGoToTiltPercentage:
		r.invoke("cover", windowcoveringdef.ClusterID, cmd, windowcoveringdef.GoToTiltPercentageRequest{TiltPercent100thsValue: fields[0].(uint16)})
	default:
		r.invoke("cover", windowcoveringdef.ClusterID, cmd, nil)
	}
}

// TestCoverTiltOnlyPositional: a tilt-only cover with SET_TILT_POSITION
// (SwitchBot Blind Tilt, features 240) has no open_cover, close_cover or
// set_cover_position; its lift commands go to the tilt axis, as
// matter-hub does (decision 47).
func TestCoverTiltOnlyPositional(t *testing.T) {
	r := serve(t, cover("blind", 240))
	r.cover(windowcoveringdef.CmdUpOrOpen)
	expect(t, r.command(), hamapping.MethodTilt, "100")
	r.cover(windowcoveringdef.CmdDownOrClose)
	expect(t, r.command(), hamapping.MethodTilt, "0")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(2500))
	expect(t, r.command(), hamapping.MethodTilt, "75")
	r.cover(windowcoveringdef.CmdGoToTiltPercentage, uint16(2500))
	expect(t, r.command(), hamapping.MethodTilt, "75")
	r.noMore()
}

// TestCoverTiltOnlyBinary: a tilt-only cover without SET_TILT_POSITION
// (OPEN_TILT|CLOSE_TILT) opens and closes its tilt: a lift fully open
// opens, fully closed closes, a position between is refused; its own tilt
// percentage opens at 0 and closes otherwise.
func TestCoverTiltOnlyBinary(t *testing.T) {
	r := serve(t, cover("blind", 48))
	r.cover(windowcoveringdef.CmdUpOrOpen)
	expect(t, r.command(), hamapping.MethodOpenTilt, "OPEN")
	r.cover(windowcoveringdef.CmdDownOrClose)
	expect(t, r.command(), hamapping.MethodCloseTilt, "CLOSE")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(0))
	expect(t, r.command(), hamapping.MethodOpenTilt, "OPEN")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(10000))
	expect(t, r.command(), hamapping.MethodCloseTilt, "CLOSE")
	r.refusedLift(5000)
	r.cover(windowcoveringdef.CmdGoToTiltPercentage, uint16(0))
	expect(t, r.command(), hamapping.MethodOpenTilt, "OPEN")
	r.cover(windowcoveringdef.CmdGoToTiltPercentage, uint16(10000))
	expect(t, r.command(), hamapping.MethodCloseTilt, "CLOSE")
	r.noMore()
}

// TestCoverLiftBinaryTilt: lift with OPEN_TILT|CLOSE_TILT but no
// SET_TILT_POSITION keeps its lift and opens or closes the tilt;
// set_cover_tilt_position would be refused.
func TestCoverLiftBinaryTilt(t *testing.T) {
	r := serve(t, cover("shutter", 51)) // OPEN|CLOSE|OPEN_TILT|CLOSE_TILT
	r.cover(windowcoveringdef.CmdUpOrOpen)
	expect(t, r.command(), "", "OPEN")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(10000))
	expect(t, r.command(), "", "CLOSE")
	r.cover(windowcoveringdef.CmdGoToTiltPercentage, uint16(0))
	expect(t, r.command(), hamapping.MethodOpenTilt, "OPEN")
	r.cover(windowcoveringdef.CmdGoToTiltPercentage, uint16(2500))
	expect(t, r.command(), hamapping.MethodCloseTilt, "CLOSE")
	r.noMore()
}

// TestCoverLiftTilt: lift and tilt with positions keep both axes as they
// were.
func TestCoverLiftTilt(t *testing.T) {
	r := serve(t, cover("blind", 255))
	r.cover(windowcoveringdef.CmdUpOrOpen)
	expect(t, r.command(), "", "OPEN")
	r.cover(windowcoveringdef.CmdDownOrClose)
	expect(t, r.command(), "", "CLOSE")
	r.cover(windowcoveringdef.CmdStopMotion)
	expect(t, r.command(), "", "STOP")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(2500))
	expect(t, r.command(), hamapping.MethodSetPosition, "75")
	r.cover(windowcoveringdef.CmdGoToTiltPercentage, uint16(2500))
	expect(t, r.command(), hamapping.MethodTilt, "75")
	r.noMore()
}

// TestCoverStopTilt: StopMotion of a cover with STOP_TILT and no STOP is
// stop_cover_tilt (MethodStopTilt); with STOP and STOP_TILT it stays
// stop_cover (the command binding's payload_stop).
func TestCoverStopTilt(t *testing.T) {
	// 112: tilt open, close and stop; 67: lift open and close, tilt stop.
	for _, f := range []int{112, 67} {
		t.Run(fmt.Sprint(f), func(t *testing.T) {
			r := serve(t, cover("blind", f))
			r.cover(windowcoveringdef.CmdStopMotion)
			expect(t, r.command(), hamapping.MethodStopTilt, "STOP")
			r.noMore()
		})
	}
	r := serve(t, cover("blind", 1|2|8|64)) // STOP and STOP_TILT
	r.cover(windowcoveringdef.CmdStopMotion)
	expect(t, r.command(), "", "STOP")
	r.noMore()
}

// refusedLift invokes GoToLiftPercentage and wants it refused with FAILURE.
func (r *rig) refusedLift(pct uint16) {
	r.t.Helper()
	_, err := fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep("cover"), windowcoveringdef.ClusterID, windowcoveringdef.CmdGoToLiftPercentage),
		windowcoveringdef.GoToLiftPercentageRequest{LiftPercent100thsValue: pct})
	if s := statusOf(err); s != 0x01 {
		r.t.Fatalf("GoToLiftPercentage %d: status 0x%02X (%v), want FAILURE", pct, s, err)
	}
}

// TestCoverPositionAwareLiftAlways: every cover serves PositionAwareLift,
// with or without SET_POSITION (fact note 2026-11-26 Q4b).
func TestCoverPositionAwareLiftAlways(t *testing.T) {
	for _, f := range []int{0, 3, 4, 15, 48, 240, 255} {
		p, _ := hamapping.Map(testDevice, []model.Entity{cover("shutter", f)})
		found := false
		for _, d := range p.Endpoints[0].Declarations {
			if wd, ok := d.(windowcoveringdef.Declare); ok {
				found = true
				if wd.Features&windowcoveringdef.FeaturePositionAwareLift == 0 {
					t.Errorf("features %d: Features 0x%X without PositionAwareLift", f, uint32(wd.Features))
				}
			}
		}
		if !found {
			t.Fatalf("features %d: no WindowCovering declaration", f)
		}
	}
}

// TestCoverLiftPositionFromState: without SET_POSITION the lift position
// is the state's — closed 10000, open 0, moving null — unless the state
// carries current_position, which wins.
func TestCoverLiftPositionFromState(t *testing.T) {
	r := serve(t, cover("shutter", 3)) // OPEN|CLOSE
	lift := func() (*uint16, *uint16) {
		st, _ := windowcoveringdef.StateFrom(r.values("cover", windowcoveringdef.ClusterID))
		return st.CurrentPositionLiftPercent100ths, st.TargetPositionLiftPercent100ths
	}
	for _, c := range []struct {
		state map[string]any
		want  *uint16
	}{
		{map[string]any{"state": "closed"}, new(uint16(10000))},
		{map[string]any{"state": "opening"}, nil},
		{map[string]any{"state": "open"}, new(uint16(0))},
		{map[string]any{"state": "closing"}, nil},
		{map[string]any{"state": "open", "current_position": 30}, new(uint16(7000))},
	} {
		r.apply("cover", on(c.state))
		cur, tgt := lift()
		if !equalPtr(cur, c.want) || !equalPtr(tgt, c.want) {
			t.Errorf("state %v: current %v target %v, want %v", c.state, deref(cur), deref(tgt), deref(c.want))
		}
	}
}

func equalPtr(a, b *uint16) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

func deref(p *uint16) any {
	if p == nil {
		return nil
	}
	return *p
}

// TestCoverGoToLiftWithoutSetPosition: without SET_POSITION a
// GoToLiftPercentage of 0 opens, of 10000 closes, and any position between
// is refused with FAILURE and hands out nothing — no threshold turns 40 %
// into "close fully" (docs/by_design.md). With SET_TILT_POSITION a
// tilt-only cover's lift percentage stays exact.
func TestCoverGoToLiftWithoutSetPosition(t *testing.T) {
	for _, f := range []int{3, 0} { // OPEN|CLOSE; neither axis
		t.Run(fmt.Sprint(f), func(t *testing.T) {
			r := serve(t, cover("shutter", f))
			r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(0))
			expect(t, r.command(), "", "OPEN")
			r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(10000))
			expect(t, r.command(), "", "CLOSE")
			for _, pct := range []uint16{1, 4000, 5000, 9999} {
				r.refusedLift(pct)
			}
			r.noMore()
		})
	}
	r := serve(t, cover("blind", 240))
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(4000))
	expect(t, r.command(), hamapping.MethodTilt, "60")
	r.noMore()
}
