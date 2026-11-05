// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"testing"

	"github.com/SukramJ/go-fabric/cluster/spec"
	rvcopdef "github.com/SukramJ/go-fabric/cluster/spec/rvcoperationalstate"
	rvcrunmodedef "github.com/SukramJ/go-fabric/cluster/spec/rvcrunmode"
	thermostatdef "github.com/SukramJ/go-fabric/cluster/spec/thermostat"
	valvedef "github.com/SukramJ/go-fabric/cluster/spec/valveconfigurationandcontrol"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-fabric/fabrictest"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

func TestVacuum(t *testing.T) {
	e := entity("vac", hacatalog.PlatformVacuum, model.Description{Extra: map[string]any{
		"supported_features": []string{"START", "PAUSE", "RETURN_HOME", "STATE"},
	}})
	r := serve(t, e)
	wantType(t, r, "vac", device.RoboticVacuumCleanerType(), eligibility.StatePartial)
	r.apply("vac", value("cleaning"))
	run, _ := rvcrunmodedef.StateFrom(r.values("vac", rvcrunmodedef.ClusterID))
	if run.CurrentMode != 1 || len(run.SupportedModes) != 2 || run.SupportedModes[0].ModeTags[0].Value != rvcrunmodedef.ModeTagIdle {
		t.Fatalf("run mode %+v", run)
	}
	op, _ := rvcopdef.StateFrom(r.values("vac", rvcopdef.ClusterID))
	if op.OperationalState != rvcopdef.OperationalStateRunning || len(op.OperationalStateList) != 7 ||
		op.OperationalError.ErrorStateId != rvcopdef.ErrorStateNoError {
		t.Fatalf("operational state %+v", op)
	}
	resp, err := fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep("vac"), rvcrunmodedef.ClusterID, rvcrunmodedef.CmdChangeToMode),
		rvcrunmodedef.ChangeToModeRequest{NewMode: 0})
	if err != nil {
		t.Fatal(err)
	}
	expect(t, r.command(), hamapping.MethodRunMode, "0")
	if got := changeToModeStatus(t, resp); got != rvcrunmodedef.ModeChangeStatusSuccess {
		t.Fatalf("ChangeToMode 0: status %v", got)
	}
	resp, err = fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep("vac"), rvcrunmodedef.ClusterID, rvcrunmodedef.CmdChangeToMode),
		rvcrunmodedef.ChangeToModeRequest{NewMode: 7})
	if err != nil {
		t.Fatal(err)
	}
	if got := changeToModeStatus(t, resp); got != rvcrunmodedef.ModeChangeStatusUnsupportedMode {
		t.Fatalf("ChangeToMode 7: status %v, want UnsupportedMode", got)
	}
	r.invoke("vac", rvcopdef.ClusterID, rvcopdef.CmdPause, nil)
	expect(t, r.command(), hamapping.MethodPause, "PAUSE")
	r.invoke("vac", rvcopdef.ClusterID, rvcopdef.CmdResume, nil)
	expect(t, r.command(), hamapping.MethodResume, "START")
	r.invoke("vac", rvcopdef.ClusterID, rvcopdef.CmdGoHome, nil)
	expect(t, r.command(), hamapping.MethodGoHome, "RETURN_TO_BASE")
	r.apply("vac", value("docked"))
	if op, _ := rvcopdef.StateFrom(r.values("vac", rvcopdef.ClusterID)); op.OperationalState != rvcopdef.OperationalStateDocked {
		t.Fatalf("docked: %v", op.OperationalState)
	}
	r.noMore()
}

func TestVacuumWithoutStart(t *testing.T) {
	_, v := hamapping.Map(testDevice, []model.Entity{entity("vac", hacatalog.PlatformVacuum, model.Description{})})
	if v[0].Row != "vacuum/unmapped" || v[0].State != eligibility.StateUnmappable {
		t.Fatalf("verdict %+v", v[0])
	}
}

func changeToModeStatus(t *testing.T, res fabrictest.Result) rvcrunmodedef.ModeChangeStatus {
	t.Helper()
	var resp rvcrunmodedef.ChangeToModeResponse
	if err := resp.DecodeTLV(res.Fields); err != nil {
		t.Fatal(err)
	}
	return resp.Status
}

func TestValve(t *testing.T) {
	r := serve(t, entity("valve", "valve", model.Description{DeviceClass: "water", Extra: map[string]any{"supported_features": 3}}))
	wantType(t, r, "valve", device.WaterValveType(), eligibility.StateMappable)
	r.apply("valve", value("opening"))
	st, _ := valvedef.StateFrom(r.values("valve", valvedef.ClusterID))
	if st.CurrentState == nil || *st.CurrentState != valvedef.ValveStateTransitioning ||
		st.TargetState == nil || *st.TargetState != valvedef.ValveStateOpen || st.OpenDuration != nil {
		t.Fatalf("valve %+v", st)
	}
	r.invoke("valve", valvedef.ClusterID, valvedef.CmdClose, nil)
	expect(t, r.command(), hamapping.MethodCloseValve, "CLOSE")
	r.invoke("valve", valvedef.ClusterID, valvedef.CmdOpen, valvedef.OpenRequest{})
	expect(t, r.command(), hamapping.MethodOpenValve, "OPEN")
	r.apply("valve", value("closed"))
	if st, _ := valvedef.StateFrom(r.values("valve", valvedef.ClusterID)); *st.CurrentState != valvedef.ValveStateClosed {
		t.Fatalf("closed: %v", *st.CurrentState)
	}
	r.noMore()
	_, v := hamapping.Map(testDevice, []model.Entity{entity("pos", "valve", model.Description{Extra: map[string]any{"supported_features": 7}})})
	if v[0].State != eligibility.StatePartial {
		t.Fatalf("a valve with a position: %+v", v[0])
	}
}

func TestWaterHeater(t *testing.T) {
	e := entity("boiler", "water_heater", model.Description{Extra: map[string]any{"min_temp": 40, "max_temp": 65}})
	r := serve(t, e)
	wantType(t, r, "boiler", device.ThermostatType(), eligibility.StatePartial)
	r.apply("boiler", on(map[string]any{"state": "eco", "current_temperature": 48.5, "temperature": 55}))
	st, _ := thermostatdef.StateFrom(r.values("boiler", thermostatdef.ClusterID))
	if st.SystemMode != thermostatdef.SystemModeHeat || st.LocalTemperature == nil || *st.LocalTemperature != 4850 ||
		st.OccupiedHeatingSetpoint != 5500 || st.AbsMaxHeatSetpointLimit != 6500 ||
		st.ControlSequenceOfOperation != thermostatdef.ControlSequenceOfOperationHeatingOnly {
		t.Fatalf("thermostat %+v", st)
	}
	r.write("boiler", thermostatdef.ClusterID, thermostatdef.AttrOccupiedHeatingSetpoint, spec.Int16(6000))
	expect(t, r.command(), hamapping.MethodTemperature, "60")
	r.write("boiler", thermostatdef.ClusterID, thermostatdef.AttrSystemMode, spec.Uint8(uint8(thermostatdef.SystemModeOff)))
	expect(t, r.command(), hamapping.MethodOperationMode, "off")
	r.apply("boiler", on(map[string]any{"state": "off", "current_temperature": 48.5, "temperature": 55}))
	if st, _ := thermostatdef.StateFrom(r.values("boiler", thermostatdef.ClusterID)); st.SystemMode != thermostatdef.SystemModeOff {
		t.Fatalf("off: %v", st.SystemMode)
	}
	r.write("boiler", thermostatdef.ClusterID, thermostatdef.AttrSystemMode, spec.Uint8(uint8(thermostatdef.SystemModeHeat)))
	expect(t, r.command(), hamapping.MethodOperationMode, "on")
	r.noMore()
}
