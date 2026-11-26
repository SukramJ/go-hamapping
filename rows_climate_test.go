// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"strings"
	"testing"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
	"thingfabric.dev/go-fabric/cluster/spec"
	fancontroldef "thingfabric.dev/go-fabric/cluster/spec/fancontrol"
	onoffdef "thingfabric.dev/go-fabric/cluster/spec/onoff"
	humiditydef "thingfabric.dev/go-fabric/cluster/spec/relativehumiditymeasurement"
	thermostatdef "thingfabric.dev/go-fabric/cluster/spec/thermostat"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/eligibility"

	"sukramj.dev/go-hamapping"
)

// TestClimatePresetsArePartial: go-fabric's Thermostat server refuses a
// preset without setpoints (cluster/thermo/presets.go:246-257 at
// v0.15.4), and a preset mode carries none, so the presets are a caveat.
func TestClimatePresetsArePartial(t *testing.T) {
	_, v := hamapping.Map(testDevice, []model.Entity{entity("hvac", hacatalog.PlatformClimate, model.Description{Extra: map[string]any{
		"modes": []string{"heat"}, "preset_modes": []string{"none", "eco"},
	}})})
	if v[0].Row != "climate/thermostat" || v[0].State != eligibility.StatePartial || !strings.Contains(v[0].Reason, "preset modes") {
		t.Fatalf("verdict %+v", v[0])
	}
}

func TestClimateAirConditioner(t *testing.T) {
	e := entity("ac", hacatalog.PlatformClimate, model.Description{Extra: map[string]any{
		"supported_features": []string{"TARGET_TEMPERATURE", "FAN_MODE", "TURN_ON", "TURN_OFF"},
		"modes":              []string{"off", "cool", "dry", "fan_only"}, "min_temp": 16, "max_temp": 30,
		"fan_modes": []string{"auto", "low", "medium", "high"},
	}})
	r := serve(t, e)
	wantType(t, r, "ac", device.RoomAirConditionerType(), eligibility.StatePartial)
	r.apply("ac", on(map[string]any{"state": "cool", "current_temperature": 26, "temperature": 24, "fan_mode": "medium", "current_humidity": 55}))
	if st, _ := onoffdef.StateFrom(r.values("ac", onoffdef.ClusterID)); !st.OnOff {
		t.Fatal("OnOff false while cooling")
	}
	fan, _ := fancontroldef.StateFrom(r.values("ac", fancontroldef.ClusterID))
	if fan.FanMode != fancontroldef.FanModeMedium || fan.FanModeSequence != fancontroldef.FanModeSequenceOffLowMedHighAuto {
		t.Fatalf("fan %+v", fan)
	}
	if h, _ := humiditydef.StateFrom(r.values("ac", humiditydef.ClusterID)); h.MeasuredValue == nil || *h.MeasuredValue != 5500 {
		t.Fatalf("humidity %v", h.MeasuredValue)
	}
	th, _ := thermostatdef.StateFrom(r.values("ac", thermostatdef.ClusterID))
	if th.SystemMode != thermostatdef.SystemModeCool || th.OccupiedCoolingSetpoint != 2400 {
		t.Fatalf("thermostat %v %d", th.SystemMode, th.OccupiedCoolingSetpoint)
	}
	// PercentSetting and PercentCurrent follow the fan mode; under auto
	// PercentSetting is null, so a write of the last speed reaches the veto.
	r.apply("ac", on(map[string]any{"state": "cool", "current_temperature": 26, "temperature": 24, "fan_mode": "low"}))
	fan, _ = fancontroldef.StateFrom(r.values("ac", fancontroldef.ClusterID))
	if fan.PercentSetting == nil || *fan.PercentSetting != 33 || fan.PercentCurrent != 33 {
		t.Fatalf("fan low %v %d", fan.PercentSetting, fan.PercentCurrent)
	}
	r.apply("ac", on(map[string]any{"state": "cool", "current_temperature": 26, "temperature": 24, "fan_mode": "auto"}))
	fan, _ = fancontroldef.StateFrom(r.values("ac", fancontroldef.ClusterID))
	if fan.FanMode != fancontroldef.FanModeAuto || fan.PercentSetting != nil || fan.PercentCurrent != 33 {
		t.Fatalf("fan auto %v %v %d", fan.FanMode, fan.PercentSetting, fan.PercentCurrent)
	}
	r.write("ac", fancontroldef.ClusterID, fancontroldef.AttrPercentSetting, spec.Uint8(33))
	expect(t, r.command(), hamapping.MethodFanMode, "low")
	r.invoke("ac", onoffdef.ClusterID, onoffdef.CmdOff, nil)
	expect(t, r.command(), hamapping.MethodTurnOff, "OFF")
	r.invoke("ac", onoffdef.ClusterID, onoffdef.CmdOn, nil)
	expect(t, r.command(), hamapping.MethodTurnOn, "ON")
	r.invoke("ac", onoffdef.ClusterID, onoffdef.CmdToggle, nil)
	expect(t, r.command(), hamapping.MethodToggle, "TOGGLE")
	r.write("ac", fancontroldef.ClusterID, fancontroldef.AttrFanMode, spec.Uint8(uint8(fancontroldef.FanModeHigh)))
	expect(t, r.command(), hamapping.MethodFanMode, "high")
	r.write("ac", fancontroldef.ClusterID, fancontroldef.AttrPercentSetting, spec.Uint8(20))
	expect(t, r.command(), hamapping.MethodFanMode, "low")
	r.write("ac", fancontroldef.ClusterID, fancontroldef.AttrPercentSetting, spec.Uint8(0))
	expect(t, r.command(), hamapping.MethodFanMode, "off")
	r.write("ac", thermostatdef.ClusterID, thermostatdef.AttrSystemMode, spec.Uint8(uint8(thermostatdef.SystemModeDry)))
	expect(t, r.command(), hamapping.MethodMode, "dry")
	r.apply("ac", on(map[string]any{"state": "off", "current_temperature": 26, "fan_mode": "off"}))
	if st, _ := onoffdef.StateFrom(r.values("ac", onoffdef.ClusterID)); st.OnOff {
		t.Fatal("OnOff true while off")
	}
	r.noMore()
}

func TestClimateFanModesWithoutPower(t *testing.T) {
	_, v := hamapping.Map(testDevice, []model.Entity{entity("hvac", hacatalog.PlatformClimate, model.Description{Extra: map[string]any{
		"supported_features": []string{"FAN_MODE"}, "modes": []string{"heat"}, "fan_modes": []string{"low", "high"},
	}})})
	if v[0].Row != "climate/thermostat" || v[0].State != eligibility.StatePartial {
		t.Fatalf("verdict %+v", v[0])
	}
}

// TestSetpointBoundsMappableBuilds is the standing guard of HR-P2-adpt-43:
// a climate or water_heater entity Map calls mappable builds for every
// pair of min_temp and max_temp, a missing bound read as Home Assistant's
// default for it; one whose min_temp is above its max_temp is unmappable
// with a reason, not swapped, and so is a heat_cool one whose span is
// below the thermostat's deadband.
func TestSetpointBoundsMappableBuilds(t *testing.T) {
	bounds := map[string][][2]any{
		"": {
			{nil, nil},
			{7, 30},
			{30, 7},
			{20, 20},
			{20, 21},
			{20, 22},
			{20, 23},
			{20.5, 20},
			{40, 65},
			{65, 40},
			{nil, 25},
			{15, nil},
			{-10, -20},
			{50, nil},
			{nil, 5},
			{34, nil},
			{nil, 8},
		},
		"°F": {{nil, nil}, {45, 86}, {86, 45}, {68, 68}, {60, 80}, {68.5, 68}, {110, 140}, {140, 110}, {50, nil}, {122, nil}, {nil, 40}},
	}
	shapes := []struct {
		p     hacatalog.Platform
		extra map[string]any
	}{
		{"water_heater", map[string]any{}},
		{hacatalog.PlatformClimate, map[string]any{"modes": []string{"heat"}}},
		{hacatalog.PlatformClimate, map[string]any{"modes": []string{"cool"}}},
		{hacatalog.PlatformClimate, map[string]any{"modes": []string{"off", "heat", "cool", "heat_cool", "auto"}}},
		{hacatalog.PlatformClimate, map[string]any{
			"modes": []string{"off", "cool"}, "supported_features": []string{"TARGET_TEMPERATURE", "TURN_ON", "TURN_OFF"},
		}},
	}
	for _, s := range shapes {
		for _, unit := range []string{"", "°F"} {
			for _, b := range bounds[unit] {
				x := map[string]any{}
				for k, v := range s.extra {
					x[k] = v
				}
				if b[0] != nil {
					x["min_temp"] = b[0]
				}
				if b[1] != nil {
					x["max_temp"] = b[1]
				}
				e := entity("e", s.p, model.Description{Unit: model.Unit(unit), Extra: x})
				p, v := hamapping.Map(testDevice, []model.Entity{e})
				_, err := p.Build(context.Background())
				p.Close()
				inverted := b[0] != nil && b[1] != nil && toF(b[0]) > toF(b[1])
				switch {
				case inverted && (v[0].State != eligibility.StateUnmappable || !strings.Contains(v[0].Reason, "min_temp")):
					t.Errorf("%s %v %s %v: %+v, want unmappable naming min_temp", s.p, s.extra, unit, b, v[0])
				case v[0].State != eligibility.StateUnmappable && err != nil:
					t.Errorf("%s %v %s %v: %s, refused: %v", s.p, s.extra, unit, b, v[0].State, err)
				}
			}
		}
	}
}

func toF(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

// TestSetpointBoundsRefused: the shapes go-fabric cannot build are
// unmappable with a reason naming the bounds — a missing bound read as
// Home Assistant's default (climate 7 / 35 °C, water_heater 110 / 140 °F),
// compared as the int16 go-fabric is given — and a heat_cool span of at
// least the 2.0 °C deadband is mappable.
func TestSetpointBoundsRefused(t *testing.T) {
	heatCool := map[string]any{"modes": []string{"off", "heat", "cool", "heat_cool"}}
	for _, c := range []struct {
		p        hacatalog.Platform
		unit     string
		extra    map[string]any
		min, max any
		reason   string // "" for mappable
	}{
		{hacatalog.PlatformClimate, "", map[string]any{"modes": []string{"heat"}}, 50, nil, "min_temp 50 is above max_temp 35 °C (Home Assistant's default)"},
		{hacatalog.PlatformClimate, "", map[string]any{"modes": []string{"cool"}}, nil, 5, "min_temp 7 °C (Home Assistant's default) is above max_temp 5"},
		{hacatalog.PlatformClimate, "", map[string]any{"modes": []string{"heat"}}, 15, nil, ""},
		{"water_heater", "", map[string]any{}, nil, 25, "min_temp 110 °F (Home Assistant's default) is above max_temp 25"},
		{"water_heater", "°F", map[string]any{}, nil, 120, ""},
		{hacatalog.PlatformClimate, "", heatCool, 20, 20, "min_temp 20 and max_temp 20 are closer than the thermostat's 2.0 °C heat/cool deadband"},
		{hacatalog.PlatformClimate, "", heatCool, 20, 21.99, "min_temp 20 and max_temp 21.99 are closer than the thermostat's 2.0 °C heat/cool deadband"},
		{hacatalog.PlatformClimate, "", heatCool, 20, 21.996, ""}, // 2200 as go-fabric is given it
		{hacatalog.PlatformClimate, "", heatCool, 34, nil, "min_temp 34 and max_temp 35 °C (Home Assistant's default) are closer than the thermostat's 2.0 °C heat/cool deadband"},
		{hacatalog.PlatformClimate, "", map[string]any{"modes": []string{"heat", "cool"}}, 20, 20, ""}, // no heat_cool, no AutoMode
	} {
		x := map[string]any{}
		for k, v := range c.extra {
			x[k] = v
		}
		if c.min != nil {
			x["min_temp"] = c.min
		}
		if c.max != nil {
			x["max_temp"] = c.max
		}
		p, v := hamapping.Map(testDevice, []model.Entity{entity("e", c.p, model.Description{Unit: model.Unit(c.unit), Extra: x})})
		if c.reason != "" {
			if v[0].State != eligibility.StateUnmappable || v[0].Reason != c.reason || len(p.Endpoints) != 0 {
				t.Errorf("%s %v %v..%v: %+v, want unmappable %q", c.p, c.extra, c.min, c.max, v[0], c.reason)
			}
			continue
		}
		if v[0].State == eligibility.StateUnmappable {
			t.Errorf("%s %v %v..%v: unmappable: %s", c.p, c.extra, c.min, c.max, v[0].Reason)
			continue
		}
		if _, err := p.Build(context.Background()); err != nil {
			t.Errorf("%s %v %v..%v: refused: %v", c.p, c.extra, c.min, c.max, err)
		}
		p.Close()
	}
}
