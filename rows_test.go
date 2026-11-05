// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/SukramJ/go-fabric/cluster/spec"
	booleanstatedef "github.com/SukramJ/go-fabric/cluster/spec/booleanstate"
	colorcontroldef "github.com/SukramJ/go-fabric/cluster/spec/colorcontrol"
	doorlockdef "github.com/SukramJ/go-fabric/cluster/spec/doorlock"
	eemdef "github.com/SukramJ/go-fabric/cluster/spec/electricalenergymeasurement"
	epmdef "github.com/SukramJ/go-fabric/cluster/spec/electricalpowermeasurement"
	fancontroldef "github.com/SukramJ/go-fabric/cluster/spec/fancontrol"
	illuminancedef "github.com/SukramJ/go-fabric/cluster/spec/illuminancemeasurement"
	levelcontroldef "github.com/SukramJ/go-fabric/cluster/spec/levelcontrol"
	occupancydef "github.com/SukramJ/go-fabric/cluster/spec/occupancysensing"
	onoffdef "github.com/SukramJ/go-fabric/cluster/spec/onoff"
	powersourcedef "github.com/SukramJ/go-fabric/cluster/spec/powersource"
	pressuredef "github.com/SukramJ/go-fabric/cluster/spec/pressuremeasurement"
	humiditydef "github.com/SukramJ/go-fabric/cluster/spec/relativehumiditymeasurement"
	smokecoalarmdef "github.com/SukramJ/go-fabric/cluster/spec/smokecoalarm"
	temperaturedef "github.com/SukramJ/go-fabric/cluster/spec/temperaturemeasurement"
	thermostatdef "github.com/SukramJ/go-fabric/cluster/spec/thermostat"
	windowcoveringdef "github.com/SukramJ/go-fabric/cluster/spec/windowcovering"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-fabric/fabrictest"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

func on(m map[string]any) model.State { return model.State{Value: m, Available: true} }

func light(modes ...string) model.Entity {
	return entity("light", hacatalog.PlatformLight, model.Description{
		Name: model.L("Lamp"), Extra: map[string]any{"supported_color_modes": modes},
	})
}

func (r *rig) invoke(key string, cluster, cmd uint32, fields spec.Encodable) {
	r.t.Helper()
	if _, err := fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep(key), cluster, cmd), fields); err != nil {
		r.t.Fatalf("invoke 0x%04X/0x%02X: %v", cluster, cmd, err)
	}
}

func (r *rig) invokeTimed(key string, cluster, cmd uint32, fields spec.Encodable) {
	r.t.Helper()
	if _, err := fabrictest.InvokeTimed(r.ctx, r.pair, client.CommandAt(r.ep(key), cluster, cmd), fields); err != nil {
		r.t.Fatalf("invoke 0x%04X/0x%02X: %v", cluster, cmd, err)
	}
}

func (r *rig) write(key string, cluster, attr uint32, v spec.Encodable) {
	r.t.Helper()
	if err := fabrictest.Write(r.ctx, r.pair, client.AttributeAt(r.ep(key), cluster, attr), v); err != nil {
		r.t.Fatalf("write 0x%04X/0x%04X: %v", cluster, attr, err)
	}
}

func (r *rig) verdict(key string) hamapping.Verdict {
	r.t.Helper()
	for _, v := range r.verdct {
		if v.Key == key {
			return v
		}
	}
	r.t.Fatalf("no verdict for %s", key)
	return hamapping.Verdict{}
}

func wantType(t *testing.T, r *rig, key string, typ *device.Type, state eligibility.State) {
	t.Helper()
	v := r.verdict(key)
	if v.State != state || v.DeviceType != uint16(typ.ID) {
		t.Fatalf("verdict %+v, want %v %s", v, state, typ.Name)
	}
	if got := r.plan.DeviceFor(key).Type(); got.ID != typ.ID {
		t.Fatalf("built %s, want %s", got.Name, typ.Name)
	}
}

func TestLightOnOff(t *testing.T) {
	r := serve(t, light("onoff"))
	wantType(t, r, "light", device.OnOffLightType(), eligibility.StateMappable)
	r.apply("light", on(map[string]any{"state": "ON"}))
	if st, _ := onoffdef.StateFrom(r.values("light", onoffdef.ClusterID)); !st.OnOff {
		t.Fatalf("OnOff %+v", st)
	}
	r.invoke("light", onoffdef.ClusterID, onoffdef.CmdOff, nil)
	expect(t, r.command(), "", `{"state":"OFF"}`)
	r.invoke("light", onoffdef.ClusterID, onoffdef.CmdToggle, nil)
	expect(t, r.command(), "", `{"state":"ON"}`)
}

func TestLightDimmable(t *testing.T) {
	r := serve(t, light("brightness"))
	wantType(t, r, "light", device.DimmableLightType(), eligibility.StateMappable)
	r.apply("light", on(map[string]any{"state": "ON", "brightness": 128}))
	lc, _ := levelcontroldef.StateFrom(r.values("light", levelcontroldef.ClusterID))
	if lc.CurrentLevel == nil || *lc.CurrentLevel != 127 {
		t.Fatalf("CurrentLevel %v, want 127 (128 of 255 on 1..254)", lc.CurrentLevel)
	}
	r.invoke("light", levelcontroldef.ClusterID, levelcontroldef.CmdMoveToLevel, levelcontroldef.MoveToLevelRequest{Level: 254})
	expect(t, r.command(), "", `{"state":"ON","brightness":255}`)
}

func TestLightBrightnessScale(t *testing.T) {
	e := entity("light", hacatalog.PlatformLight, model.Description{Extra: map[string]any{
		"supported_color_modes": []any{"brightness"}, "brightness_scale": 100,
	}})
	r := serve(t, e)
	r.apply("light", on(map[string]any{"state": "ON", "brightness": 50}))
	lc, _ := levelcontroldef.StateFrom(r.values("light", levelcontroldef.ClusterID))
	if lc.CurrentLevel == nil || *lc.CurrentLevel != 127 {
		t.Fatalf("CurrentLevel %v, want 127 (50 of 100)", lc.CurrentLevel)
	}
	r.invoke("light", levelcontroldef.ClusterID, levelcontroldef.CmdMoveToLevel, levelcontroldef.MoveToLevelRequest{Level: 254})
	expect(t, r.command(), "", `{"state":"ON","brightness":100}`)
}

func TestLightColorTemperature(t *testing.T) {
	for _, kelvin := range []bool{false, true} {
		t.Run(fmt.Sprint("kelvin=", kelvin), func(t *testing.T) {
			e := entity("light", hacatalog.PlatformLight, model.Description{Extra: map[string]any{
				"supported_color_modes": []string{"color_temp"}, "color_temp_kelvin": kelvin,
				"min_kelvin": 2000, "max_kelvin": 6500,
			}})
			r := serve(t, e)
			wantType(t, r, "light", device.ColorTemperatureLightType(), eligibility.StateMappable)
			temp, back := 250, `{"state":"ON","color_temp":370}`
			if kelvin {
				temp, back = 4000, `{"state":"ON","color_temp":2703}`
			}
			r.apply("light", on(map[string]any{"state": "ON", "color_temp": temp}))
			cc, _ := colorcontroldef.StateFrom(r.values("light", colorcontroldef.ClusterID))
			if cc.ColorTemperatureMireds != 250 {
				t.Fatalf("ColorTemperatureMireds %d, want 250", cc.ColorTemperatureMireds)
			}
			if cc.ColorTempPhysicalMinMireds != 154 || cc.ColorTempPhysicalMaxMireds != 500 {
				t.Fatalf("physical range %d..%d, want 154..500 (6500 K..2000 K)", cc.ColorTempPhysicalMinMireds, cc.ColorTempPhysicalMaxMireds)
			}
			r.invoke("light", colorcontroldef.ClusterID, colorcontroldef.CmdMoveToColorTemperature,
				colorcontroldef.MoveToColorTemperatureRequest{ColorTemperatureMireds: 370})
			expect(t, r.command(), "", back)
		})
	}
}

func TestLightExtendedColor(t *testing.T) {
	r := serve(t, light("color_temp", "hs", "xy"))
	wantType(t, r, "light", device.ExtendedColorLightType(), eligibility.StateMappable)
	r.apply("light", on(map[string]any{
		"state": "ON", "color_mode": "hs",
		"color": map[string]any{"h": 180.0, "s": 50.0, "x": 0.3, "y": 0.3},
	}))
	cc, _ := colorcontroldef.StateFrom(r.values("light", colorcontroldef.ClusterID))
	if cc.CurrentHue != 127 || cc.CurrentSaturation != 127 || cc.CurrentX != 19661 || cc.CurrentY != 19661 ||
		cc.ColorMode != colorcontroldef.ColorModeCurrentHueAndCurrentSaturation {
		t.Fatalf("colour %+v", cc)
	}
	r.invoke("light", colorcontroldef.ClusterID, colorcontroldef.CmdMoveToHueAndSaturation,
		colorcontroldef.MoveToHueAndSaturationRequest{Hue: 127, Saturation: 254})
	expect(t, r.command(), "", `{"state":"ON","color":{"h":180,"s":100}}`)
	r.noMore()
	r.invoke("light", colorcontroldef.ClusterID, colorcontroldef.CmdMoveToColor,
		colorcontroldef.MoveToColorRequest{ColorX: 32768, ColorY: 21845})
	expect(t, r.command(), "", `{"state":"ON","color":{"x":0.5,"y":0.3333}}`)
	r.noMore()
}

func TestLightHSWithoutXY(t *testing.T) {
	r := serve(t, light("hs"))
	r.apply("light", on(map[string]any{"state": "ON"})) // a light that is off ignores colour commands (ExecuteIfOff unset)
	r.invoke("light", colorcontroldef.ClusterID, colorcontroldef.CmdMoveToColor,
		colorcontroldef.MoveToColorRequest{ColorX: 19661, ColorY: 19661})
	h, s := colorcontroldef.XYToHSV(19661.0/65536, 19661.0/65536)
	expect(t, r.command(), "", fmt.Sprintf(`{"state":"ON","color":{"h":%v,"s":%v}}`,
		math.Round(h*10)/10, math.Round(s*1000)/10))
}

func TestLightRGB(t *testing.T) {
	r := serve(t, light("rgb"))
	wantType(t, r, "light", device.ExtendedColorLightType(), eligibility.StatePartial)
	if v := r.verdict("light"); v.Reason == "" {
		t.Fatal("partial without a reason")
	}
	r.apply("light", on(map[string]any{"state": "ON"}))
	r.invoke("light", colorcontroldef.ClusterID, colorcontroldef.CmdMoveToColor,
		colorcontroldef.MoveToColorRequest{ColorX: 45875, ColorY: 19661})
	red, green, blue := colorcontroldef.XYToRGB(45875.0/65536, 19661.0/65536)
	expect(t, r.command(), "", fmt.Sprintf(`{"state":"ON","color":{"r":%v,"g":%v,"b":%v}}`,
		math.Round(red*255), math.Round(green*255), math.Round(blue*255)))
}

func sensor(key, class, unit string) model.Entity {
	return entity(key, hacatalog.PlatformSensor, model.Description{
		Name: model.L(key), DeviceClass: model.DeviceClass(class), Unit: model.Unit(unit),
		StateClass: hacatalog.StateClass("measurement"),
	})
}

func value(v any) model.State { return model.State{Value: v, Available: true} }

func TestSensorTemperature(t *testing.T) {
	r := serve(t, sensor("t", "temperature", "°F"))
	wantType(t, r, "t", device.TemperatureSensorType(), eligibility.StateMappable)
	r.apply("t", value(68.0))
	st, _ := temperaturedef.StateFrom(r.values("t", temperaturedef.ClusterID))
	if st.MeasuredValue == nil || *st.MeasuredValue != 2000 {
		t.Fatalf("MeasuredValue %v, want 2000 (68 °F)", st.MeasuredValue)
	}
	r.apply("t", value("unknown"))
	if st, _ := temperaturedef.StateFrom(r.values("t", temperaturedef.ClusterID)); st.MeasuredValue != nil {
		t.Fatalf("MeasuredValue %v after unknown, want null", *st.MeasuredValue)
	}
}

func TestSensorHumidity(t *testing.T) {
	r := serve(t, sensor("h", "humidity", "%"))
	wantType(t, r, "h", device.HumiditySensorType(), eligibility.StateMappable)
	r.apply("h", value("45.5"))
	st, _ := humiditydef.StateFrom(r.values("h", humiditydef.ClusterID))
	if st.MeasuredValue == nil || *st.MeasuredValue != 4550 {
		t.Fatalf("MeasuredValue %v, want 4550", st.MeasuredValue)
	}
}

func TestSensorPressure(t *testing.T) {
	r := serve(t, sensor("p", "atmospheric_pressure", "kPa"))
	wantType(t, r, "p", device.PressureSensorType(), eligibility.StateMappable)
	r.apply("p", value(101.3))
	st, _ := pressuredef.StateFrom(r.values("p", pressuredef.ClusterID))
	if st.MeasuredValue == nil || *st.MeasuredValue != 1013 {
		t.Fatalf("MeasuredValue %v, want 1013", st.MeasuredValue)
	}
}

func TestSensorIlluminance(t *testing.T) {
	r := serve(t, sensor("l", "illuminance", "lx"))
	wantType(t, r, "l", device.LightSensorType(), eligibility.StateMappable)
	for lux, want := range map[float64]uint16{1000: 30001, 1: 1, 0: 0} {
		r.apply("l", value(lux))
		st, _ := illuminancedef.StateFrom(r.values("l", illuminancedef.ClusterID))
		if st.MeasuredValue == nil || *st.MeasuredValue != want {
			t.Fatalf("%v lx: MeasuredValue %v, want %d", lux, st.MeasuredValue, want)
		}
	}
}

func TestSensorElectrical(t *testing.T) {
	r := serve(t, sensor("w", "power", "kW"), sensor("v", "voltage", "V"), sensor("a", "current", "mA"),
		sensor("e", "energy", "kWh"))
	for _, k := range []string{"w", "v", "a", "e"} {
		wantType(t, r, k, device.ElectricalSensorType(), eligibility.StateMappable)
	}
	r.apply("w", value(-0.6))
	r.apply("v", value(230.1))
	r.apply("a", value(1500))
	r.apply("e", value(1.25))
	if st, _ := epmdef.StateFrom(r.values("w", epmdef.ClusterID)); st.ActivePower == nil || *st.ActivePower != -600000 {
		t.Fatalf("ActivePower %v, want -600000", st.ActivePower)
	}
	if st, _ := epmdef.StateFrom(r.values("v", epmdef.ClusterID)); st.Voltage == nil || *st.Voltage != 230100 {
		t.Fatalf("Voltage %v, want 230100", st.Voltage)
	}
	if st, _ := epmdef.StateFrom(r.values("a", epmdef.ClusterID)); st.ActiveCurrent == nil || *st.ActiveCurrent != 1500 {
		t.Fatalf("ActiveCurrent %v, want 1500", st.ActiveCurrent)
	}
	st, _ := eemdef.StateFrom(r.values("e", eemdef.ClusterID))
	if st.CumulativeEnergyImported == nil || st.CumulativeEnergyImported.Energy != 1250000 {
		t.Fatalf("CumulativeEnergyImported %+v, want 1250000 mWh", st.CumulativeEnergyImported)
	}
}

func TestSensorBatteryRidesOnPrimary(t *testing.T) {
	r := serve(t, sensor("bat", "battery", "%"), sensor("t", "temperature", "°C"))
	if len(r.devs) != 1 {
		t.Fatalf("%d devices, want one: the battery rides on the temperature sensor", len(r.devs))
	}
	if r.plan.DeviceFor("bat") != r.plan.DeviceFor("t") {
		t.Fatal("the battery is not on the primary's device")
	}
	if v := r.verdict("bat"); v.State != eligibility.StatePartial || v.DeviceType != uint16(device.TemperatureSensorID) {
		t.Fatalf("battery verdict %+v", v)
	}
	r.apply("t", value(21.0))
	r.apply("bat", value(80))
	ps, _ := powersourcedef.StateFrom(r.values("t", powersourcedef.ClusterID))
	if ps.BatPercentRemaining == nil || *ps.BatPercentRemaining != 160 {
		t.Fatalf("BatPercentRemaining %v, want 160", ps.BatPercentRemaining)
	}
	// The rider's availability leaves the primary's reachability alone.
	r.apply("bat", model.State{})
	if !r.reachable("t") {
		t.Fatal("unreachable after the battery's state")
	}
}

func TestSensorBatteryAlone(t *testing.T) {
	r := serve(t, sensor("bat", "battery", "%"))
	wantType(t, r, "bat", device.PowerSourceType(), eligibility.StatePartial)
	r.apply("bat", value(50))
	ps, _ := powersourcedef.StateFrom(r.values("bat", powersourcedef.ClusterID))
	if ps.BatPercentRemaining == nil || *ps.BatPercentRemaining != 100 {
		t.Fatalf("BatPercentRemaining %v, want 100", ps.BatPercentRemaining)
	}
}

func binary(key, class string) model.Entity {
	return entity(key, hacatalog.PlatformBinarySensor, model.Description{DeviceClass: model.DeviceClass(class)})
}

func TestBinarySensors(t *testing.T) {
	r := serve(t, binary("door", "door"), binary("motion", "motion"), binary("leak", "moisture"),
		binary("smoke", "smoke"), binary("co", "carbon_monoxide"), binary("plain", ""))
	wantType(t, r, "door", device.ContactSensorType(), eligibility.StateMappable)
	wantType(t, r, "motion", device.OccupancySensorType(), eligibility.StateMappable)
	wantType(t, r, "leak", device.WaterLeakDetectorType(), eligibility.StateMappable)
	wantType(t, r, "smoke", device.SmokeCoAlarmType(), eligibility.StateMappable)
	wantType(t, r, "co", device.SmokeCoAlarmType(), eligibility.StateMappable)
	wantType(t, r, "plain", device.ContactSensorType(), eligibility.StateMappable)
	for _, k := range []string{"door", "motion", "leak", "smoke", "co", "plain"} {
		r.apply(k, value(true))
	}
	if st, _ := booleanstatedef.StateFrom(r.values("door", booleanstatedef.ClusterID)); st.StateValue {
		t.Fatal("door open: StateValue true, want false (no contact)")
	}
	r.apply("door", value("OFF"))
	if st, _ := booleanstatedef.StateFrom(r.values("door", booleanstatedef.ClusterID)); !st.StateValue {
		t.Fatal("door closed: StateValue false, want true (contact)")
	}
	if st, _ := occupancydef.StateFrom(r.values("motion", occupancydef.ClusterID)); st.Occupancy != occupancydef.OccupancyOccupied {
		t.Fatalf("Occupancy %v", st.Occupancy)
	}
	if st, _ := booleanstatedef.StateFrom(r.values("leak", booleanstatedef.ClusterID)); !st.StateValue {
		t.Fatal("leak: StateValue false")
	}
	if st, _ := smokecoalarmdef.StateFrom(r.values("smoke", smokecoalarmdef.ClusterID)); st.SmokeState != smokecoalarmdef.AlarmStateCritical ||
		st.ExpressedState != smokecoalarmdef.ExpressedStateSmokeAlarm {
		t.Fatalf("smoke %+v", st)
	}
	if st, _ := smokecoalarmdef.StateFrom(r.values("co", smokecoalarmdef.ClusterID)); st.CoState != smokecoalarmdef.AlarmStateCritical ||
		st.ExpressedState != smokecoalarmdef.ExpressedStateCoAlarm {
		t.Fatalf("co %+v", st)
	}
	if st, _ := booleanstatedef.StateFrom(r.values("plain", booleanstatedef.ClusterID)); st.StateValue {
		t.Fatal("generic on: StateValue true, want false (as opening)")
	}
}

func TestCover(t *testing.T) {
	e := entity("cover", hacatalog.PlatformCover, model.Description{
		DeviceClass: "shutter",
		Extra:       map[string]any{"supported_features": 15}, // OPEN|CLOSE|SET_POSITION|STOP
	})
	r := serve(t, e)
	wantType(t, r, "cover", device.WindowCoveringType(), eligibility.StateMappable)
	r.apply("cover", on(map[string]any{"state": "open", "current_position": 30}))
	st, _ := windowcoveringdef.StateFrom(r.values("cover", windowcoveringdef.ClusterID))
	if st.CurrentPositionLiftPercent100ths == nil || *st.CurrentPositionLiftPercent100ths != 7000 {
		t.Fatalf("CurrentPositionLiftPercent100ths %v, want 7000 (30 %% open)", st.CurrentPositionLiftPercent100ths)
	}
	if st.Type != windowcoveringdef.TypeShutter {
		t.Fatalf("Type %v, want Shutter", st.Type)
	}
	r.invoke("cover", windowcoveringdef.ClusterID, windowcoveringdef.CmdUpOrOpen, nil)
	expect(t, r.command(), "", "OPEN")
	r.invoke("cover", windowcoveringdef.ClusterID, windowcoveringdef.CmdStopMotion, nil)
	expect(t, r.command(), "", "STOP")
	r.invoke("cover", windowcoveringdef.ClusterID, windowcoveringdef.CmdGoToLiftPercentage,
		windowcoveringdef.GoToLiftPercentageRequest{LiftPercent100thsValue: 2500})
	expect(t, r.command(), "set_position", "75")
	r.noMore()
}

func TestClimate(t *testing.T) {
	e := entity("hvac", hacatalog.PlatformClimate, model.Description{Extra: map[string]any{
		"modes": []string{"off", "heat", "cool", "heat_cool"}, "min_temp": 7, "max_temp": 30,
	}})
	r := serve(t, e)
	wantType(t, r, "hvac", device.ThermostatType(), eligibility.StateMappable)
	r.apply("hvac", on(map[string]any{"state": "heat", "current_temperature": 21.5, "temperature": 22}))
	st, _ := thermostatdef.StateFrom(r.values("hvac", thermostatdef.ClusterID))
	if st.LocalTemperature == nil || *st.LocalTemperature != 2150 || st.OccupiedHeatingSetpoint != 2200 ||
		st.SystemMode != thermostatdef.SystemModeHeat {
		t.Fatalf("thermostat %v %d %v", st.LocalTemperature, st.OccupiedHeatingSetpoint, st.SystemMode)
	}
	if st.AbsMinHeatSetpointLimit != 700 || st.AbsMaxCoolSetpointLimit != 3000 {
		t.Fatalf("limits %d %d", st.AbsMinHeatSetpointLimit, st.AbsMaxCoolSetpointLimit)
	}
	r.write("hvac", thermostatdef.ClusterID, thermostatdef.AttrOccupiedHeatingSetpoint, spec.Int16(2350))
	expect(t, r.command(), "temperature", "23.5")
	r.write("hvac", thermostatdef.ClusterID, thermostatdef.AttrSystemMode, spec.Uint8(uint8(thermostatdef.SystemModeCool)))
	expect(t, r.command(), "mode", "cool")
	r.noMore()
}

func TestClimateWithoutHeatingOrCooling(t *testing.T) {
	_, v := hamapping.Map(testDevice, []model.Entity{entity("fan_only", hacatalog.PlatformClimate,
		model.Description{Extra: map[string]any{"modes": []string{"off", "fan_only"}}})})
	if v[0].State != eligibility.StateUnmappable || v[0].Reason == "" {
		t.Fatalf("verdict %+v", v[0])
	}
}

func TestLock(t *testing.T) {
	r := serve(t, entity("lock", hacatalog.PlatformLock, model.Description{}))
	wantType(t, r, "lock", device.DoorLockType(), eligibility.StateMappable)
	r.apply("lock", value("LOCKED"))
	st, _ := doorlockdef.StateFrom(r.values("lock", doorlockdef.ClusterID))
	if st.LockState == nil || *st.LockState != doorlockdef.LockStateLocked {
		t.Fatalf("LockState %v", st.LockState)
	}
	r.invokeTimed("lock", doorlockdef.ClusterID, doorlockdef.CmdUnlockDoor, doorlockdef.UnlockDoorRequest{})
	expect(t, r.command(), "", "UNLOCK")
	r.invokeTimed("lock", doorlockdef.ClusterID, doorlockdef.CmdLockDoor, doorlockdef.LockDoorRequest{})
	expect(t, r.command(), "", "LOCK")
}

func TestLockOpenIsPartial(t *testing.T) {
	_, v := hamapping.Map(testDevice, []model.Entity{entity("lock", hacatalog.PlatformLock,
		model.Description{Extra: map[string]any{"supported_features": 1}})})
	if v[0].State != eligibility.StatePartial {
		t.Fatalf("verdict %+v", v[0])
	}
}

func TestFan(t *testing.T) {
	e := entity("fan", hacatalog.PlatformFan, model.Description{Extra: map[string]any{
		"supported_features": []string{"SET_SPEED", "OSCILLATE", "DIRECTION", "PRESET_MODE"},
		"preset_modes":       []string{"low", "medium", "high", "auto"},
	}})
	r := serve(t, e)
	wantType(t, r, "fan", device.FanType(), eligibility.StatePartial)
	r.apply("fan", on(map[string]any{"state": "on", "percentage": 50, "oscillating": true, "direction": "reverse"}))
	st, _ := fancontroldef.StateFrom(r.values("fan", fancontroldef.ClusterID))
	if st.PercentCurrent != 50 || st.RockSetting != fancontroldef.RockRockLeftRight ||
		st.AirflowDirection != fancontroldef.AirflowDirectionReverse {
		t.Fatalf("fan %+v", st)
	}
	r.write("fan", fancontroldef.ClusterID, fancontroldef.AttrPercentSetting, spec.Uint8(75))
	expect(t, r.command(), "percentage", "75")
	r.write("fan", fancontroldef.ClusterID, fancontroldef.AttrFanMode, spec.Uint8(uint8(fancontroldef.FanModeOff)))
	expect(t, r.command(), "", "OFF")
	r.write("fan", fancontroldef.ClusterID, fancontroldef.AttrFanMode, spec.Uint8(uint8(fancontroldef.FanModeHigh)))
	expect(t, r.command(), "preset_mode", "high")
	r.write("fan", fancontroldef.ClusterID, fancontroldef.AttrRockSetting, spec.Uint8(0))
	expect(t, r.command(), "oscillation", "oscillate_off")
	r.write("fan", fancontroldef.ClusterID, fancontroldef.AttrAirflowDirection, spec.Uint8(uint8(fancontroldef.AirflowDirectionForward)))
	expect(t, r.command(), "direction", "forward")
	r.noMore()
}

func TestUnmappablePlatforms(t *testing.T) {
	for _, p := range []hacatalog.Platform{
		hacatalog.PlatformNumber, hacatalog.PlatformSelect, hacatalog.PlatformText,
	} {
		_, v := hamapping.Map(testDevice, []model.Entity{entity("x", p, model.Description{})})
		if v[0].State != eligibility.StateUnmappable || v[0].Reason == "" {
			t.Errorf("%s: verdict %+v", p, v[0])
		}
	}
}
