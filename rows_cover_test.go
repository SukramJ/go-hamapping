// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"fmt"
	"testing"

	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"
	windowcoveringdef "thingfabric.dev/go-fabric/cluster/spec/windowcovering"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/eligibility"

	"github.com/SukramJ/go-hamapping"
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
// with tilt, take the LF&TL members (fact note 2026-11-24 §3).
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
		"garage":  unknown, "gate": unknown, "door": unknown, "damper": unknown, "window": unknown,
	}
	tilt := map[string]te{
		"":        {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeInteriorBlind},
		"blind":   {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeInteriorBlind},
		"shade":   {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeSheerShade},
		"shutter": {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeExteriorVenetianBlind},
		"curtain": {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeVerticalBlindStripCurtain},
		"awning":  {windowcoveringdef.TypeTiltBlindLift, windowcoveringdef.EndProductTypeUnknown},
		"garage":  unknown, "gate": unknown, "door": unknown, "damper": unknown, "window": unknown,
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
// (OPEN_TILT|CLOSE_TILT) opens and closes its tilt: fully open opens,
// anything else closes, as the lift without SET_POSITION.
func TestCoverTiltOnlyBinary(t *testing.T) {
	r := serve(t, cover("blind", 48))
	r.cover(windowcoveringdef.CmdUpOrOpen)
	expect(t, r.command(), hamapping.MethodOpenTilt, "OPEN")
	r.cover(windowcoveringdef.CmdDownOrClose)
	expect(t, r.command(), hamapping.MethodCloseTilt, "CLOSE")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(0))
	expect(t, r.command(), hamapping.MethodOpenTilt, "OPEN")
	r.cover(windowcoveringdef.CmdGoToLiftPercentage, uint16(5000))
	expect(t, r.command(), hamapping.MethodCloseTilt, "CLOSE")
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
