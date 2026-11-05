// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"strings"
	"testing"

	"github.com/SukramJ/go-fabric/cluster/spec"
	fancontroldef "github.com/SukramJ/go-fabric/cluster/spec/fancontrol"
	onoffdef "github.com/SukramJ/go-fabric/cluster/spec/onoff"
	humiditydef "github.com/SukramJ/go-fabric/cluster/spec/relativehumiditymeasurement"
	thermostatdef "github.com/SukramJ/go-fabric/cluster/spec/thermostat"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
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
	r.invoke("ac", onoffdef.ClusterID, onoffdef.CmdOff, nil)
	expect(t, r.command(), hamapping.MethodTurnOff, "OFF")
	r.invoke("ac", onoffdef.ClusterID, onoffdef.CmdOn, nil)
	expect(t, r.command(), hamapping.MethodTurnOn, "ON")
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
