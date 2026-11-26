// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"testing"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
	"thingfabric.dev/go-fabric/cluster/spec"
	fancontroldef "thingfabric.dev/go-fabric/cluster/spec/fancontrol"
	playbackdef "thingfabric.dev/go-fabric/cluster/spec/mediaplayback"
	onoffdef "thingfabric.dev/go-fabric/cluster/spec/onoff"
	ozonedef "thingfabric.dev/go-fabric/cluster/spec/ozoneconcentrationmeasurement"
	rvcopdef "thingfabric.dev/go-fabric/cluster/spec/rvcoperationalstate"
	thermostatdef "thingfabric.dev/go-fabric/cluster/spec/thermostat"
	"thingfabric.dev/go-fabric/eligibility"

	"sukramj.dev/go-hamapping"
)

// TestWaveRowsServe serves one entity of every row of this wave the
// round trips above leave out, applies the states its row maps and runs
// the commands not run elsewhere: every built device passes go-fabric's
// device-type validator (serve), and every Set is accepted (apply).
func TestWaveRowsServe(t *testing.T) {
	r := serve(t,
		sensor("pm1", "pm1", "μg/m³"), sensor("pm25", "pm25", "μg/m³"), sensor("pm10", "pm10", "μg/m³"),
		sensor("o3", "ozone", "μg/m³"), sensor("vocp", "volatile_organic_compounds_parts", "ppm"),
		player("vol", "receiver", "VOLUME_SET"),
		player("mute", "", "VOLUME_MUTE"),
		player("tv", "tv", "TURN_ON", "TURN_OFF", "PAUSE"),
		entity("ac", hacatalog.PlatformClimate, model.Description{Extra: map[string]any{
			"supported_features": []string{"TARGET_TEMPERATURE_RANGE", "FAN_MODE", "TURN_ON", "TURN_OFF"},
			"modes":              []string{"off", "heat", "cool", "heat_cool"}, "fan_modes": []string{"low", "medium", "high"},
		}}),
		entity("vac", hacatalog.PlatformVacuum, model.Description{Extra: map[string]any{"supported_features": 8192}}),
	)
	for _, k := range []string{"pm1", "pm25", "pm10", "vocp"} {
		r.apply(k, value(12.0))
	}
	r.apply("o3", value(80.0))
	if st, _ := ozonedef.StateFrom(r.values("o3", ozonedef.ClusterID)); st.MeasurementUnit != ozonedef.MeasurementUnitUgm3 {
		t.Fatalf("ozone in μg/m³: unit %v", st.MeasurementUnit)
	}
	r.apply("vol", on(map[string]any{"state": "idle", "volume_level": 1.0}))
	r.apply("mute", on(map[string]any{"state": "playing", "is_volume_muted": false, "volume_level": 0.0}))
	if st, _ := onoffdef.StateFrom(r.values("mute", onoffdef.ClusterID)); !st.OnOff {
		t.Fatal("OnOff false while not muted")
	}
	r.apply("tv", on(map[string]any{"state": "buffering"}))
	r.invoke("tv", playbackdef.ClusterID, playbackdef.CmdPause, nil)
	expect(t, r.command(), hamapping.MethodMediaPause, "PAUSE")
	r.invoke("tv", onoffdef.ClusterID, onoffdef.CmdOff, nil)
	expect(t, r.command(), hamapping.MethodTurnOff, "OFF")

	r.apply("ac", on(map[string]any{
		"state": "heat_cool", "current_temperature": 22, "target_temp_low": 20, "target_temp_high": 25, "fan_mode": "high",
	}))
	th, _ := thermostatdef.StateFrom(r.values("ac", thermostatdef.ClusterID))
	if th.SystemMode != thermostatdef.SystemModeAuto || th.OccupiedHeatingSetpoint != 2000 || th.OccupiedCoolingSetpoint != 2500 {
		t.Fatalf("thermostat %v %d %d", th.SystemMode, th.OccupiedHeatingSetpoint, th.OccupiedCoolingSetpoint)
	}
	if fan, _ := fancontroldef.StateFrom(r.values("ac", fancontroldef.ClusterID)); fan.FanModeSequence != fancontroldef.FanModeSequenceOffLowMedHigh {
		t.Fatalf("FanModeSequence %v without auto", fan.FanModeSequence)
	}
	r.write("ac", thermostatdef.ClusterID, thermostatdef.AttrOccupiedCoolingSetpoint, spec.Int16(2600))
	expect(t, r.command(), hamapping.MethodTemperatureHigh, "26")
	for pct, mode := range map[uint8]string{50: "medium", 90: "high"} {
		r.write("ac", fancontroldef.ClusterID, fancontroldef.AttrPercentSetting, spec.Uint8(pct))
		expect(t, r.command(), hamapping.MethodFanMode, mode)
	}

	for st, want := range map[string]rvcopdef.OperationalStateEnum{
		"returning": rvcopdef.OperationalStateSeekingCharger, "paused": rvcopdef.OperationalStatePaused,
		"idle": rvcopdef.OperationalStateStopped, "error": rvcopdef.OperationalStateError,
	} {
		r.apply("vac", value(st))
		if op, _ := rvcopdef.StateFrom(r.values("vac", rvcopdef.ClusterID)); op.OperationalState != want {
			t.Errorf("%s: %v, want %v", st, op.OperationalState, want)
		}
	}
	if v := r.verdict("vol"); v.State != eligibility.StatePartial {
		t.Errorf("a volume-only player: %+v", v)
	}
	r.noMore()
}
