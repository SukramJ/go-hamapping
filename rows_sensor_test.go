// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"testing"

	airqualitydef "github.com/SukramJ/go-fabric/cluster/spec/airquality"
	booleanstatedef "github.com/SukramJ/go-fabric/cluster/spec/booleanstate"
	co2def "github.com/SukramJ/go-fabric/cluster/spec/carbondioxideconcentrationmeasurement"
	codef "github.com/SukramJ/go-fabric/cluster/spec/carbonmonoxideconcentrationmeasurement"
	epmdef "github.com/SukramJ/go-fabric/cluster/spec/electricalpowermeasurement"
	flowdef "github.com/SukramJ/go-fabric/cluster/spec/flowmeasurement"
	no2def "github.com/SukramJ/go-fabric/cluster/spec/nitrogendioxideconcentrationmeasurement"
	radondef "github.com/SukramJ/go-fabric/cluster/spec/radonconcentrationmeasurement"
	soildef "github.com/SukramJ/go-fabric/cluster/spec/soilmeasurement"
	tvocdef "github.com/SukramJ/go-fabric/cluster/spec/totalvolatileorganiccompoundsconcentrationmeasurement"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

func TestSensorCarbonDioxide(t *testing.T) {
	r := serve(t, sensor("co2", "carbon_dioxide", "ppm"))
	wantType(t, r, "co2", device.AirQualitySensorType(), eligibility.StateMappable)
	if st, _ := co2def.StateFrom(r.values("co2", co2def.ClusterID)); st.MeasuredValue != nil ||
		st.MeasurementUnit != co2def.MeasurementUnitPpm || st.MeasurementMedium != co2def.MeasurementMediumAir {
		t.Fatalf("before a state %+v", st)
	}
	r.apply("co2", value(612.0))
	st, _ := co2def.StateFrom(r.values("co2", co2def.ClusterID))
	if st.MeasuredValue == nil || *st.MeasuredValue != 612 {
		t.Fatalf("MeasuredValue %v, want 612", st.MeasuredValue)
	}
	if aq, _ := airqualitydef.StateFrom(r.values("co2", airqualitydef.ClusterID)); aq.AirQuality != airqualitydef.AirQualityUnknown {
		t.Fatalf("AirQuality %v, want Unknown", aq.AirQuality)
	}
}

// TestSensorConcentrationUnits: a class with ratio and mass units keeps
// its unit's family on the wire — no molar mass is assumed.
func TestSensorConcentrationUnits(t *testing.T) {
	r := serve(t, sensor("co_ppb", "carbon_monoxide", "ppb"), sensor("co_mg", "carbon_monoxide", "mg/m³"),
		sensor("no2", "nitrogen_dioxide", "ppm"), sensor("tvoc", "volatile_organic_compounds", "mg/m³"),
		sensor("radon", "radon", "pCi/L"))
	r.apply("co_ppb", value(1500.0))
	r.apply("co_mg", value(2.5))
	r.apply("no2", value(0.04))
	r.apply("tvoc", value(0.3))
	r.apply("radon", value(2.0))
	ppb, _ := codef.StateFrom(r.values("co_ppb", codef.ClusterID))
	mg, _ := codef.StateFrom(r.values("co_mg", codef.ClusterID))
	no2, _ := no2def.StateFrom(r.values("no2", no2def.ClusterID))
	tvoc, _ := tvocdef.StateFrom(r.values("tvoc", tvocdef.ClusterID))
	radon, _ := radondef.StateFrom(r.values("radon", radondef.ClusterID))
	for _, c := range []struct {
		name       string
		got, want  float32
		unit, wunt uint8
	}{
		{"CO ppb", *ppb.MeasuredValue, 1.5, uint8(ppb.MeasurementUnit), uint8(codef.MeasurementUnitPpm)},
		{"CO mg/m³", *mg.MeasuredValue, 2500, uint8(mg.MeasurementUnit), uint8(codef.MeasurementUnitUgm3)},
		{"NO2 ppm", *no2.MeasuredValue, 40, uint8(no2.MeasurementUnit), uint8(no2def.MeasurementUnitPpb)},
		{"TVOC mg/m³", *tvoc.MeasuredValue, 300, uint8(tvoc.MeasurementUnit), uint8(tvocdef.MeasurementUnitUgm3)},
		{"radon pCi/L", *radon.MeasuredValue, 74, uint8(radon.MeasurementUnit), uint8(radondef.MeasurementUnitBqm3)},
	} {
		if c.got < c.want*0.999 || c.got > c.want*1.001 || c.unit != c.wunt {
			t.Errorf("%s: %v unit %d, want %v unit %d", c.name, c.got, c.unit, c.want, c.wunt)
		}
	}
}

func TestSensorAirQualityPredicates(t *testing.T) {
	for class, unit := range map[string]string{
		"pm1": "μg/m³", "pm25": "μg/m³", "pm10": "μg/m³", "ozone": "μg/m³",
		"volatile_organic_compounds_parts": "ppb",
	} {
		_, v := hamapping.Map(testDevice, []model.Entity{sensor("s", class, unit)})
		if v[0].Row != "sensor/"+class || v[0].State != eligibility.StateMappable ||
			v[0].DeviceType != uint16(device.AirQualitySensorType().ID) {
			t.Errorf("%s: %+v", class, v[0])
		}
	}
}

func TestSensorElectricalQuantities(t *testing.T) {
	r := serve(t, sensor("va", "apparent_power", "kVA"), sensor("var", "reactive_power", "var"),
		sensor("hz", "frequency", "Hz"), sensor("pf", "power_factor", "%"), sensor("ratio", "power_factor", ""))
	for _, k := range []string{"va", "var", "hz", "pf", "ratio"} {
		wantType(t, r, k, device.ElectricalSensorType(), eligibility.StateMappable)
	}
	r.apply("va", value(1.5))
	r.apply("var", value(230.0))
	r.apply("hz", value(50.02))
	r.apply("pf", value(-95.5))
	r.apply("ratio", value(0.87))
	read := func(k string) epmdef.State {
		st, _ := epmdef.StateFrom(r.values(k, epmdef.ClusterID))
		return st
	}
	for _, c := range []struct {
		name string
		got  *int64
		want int64
	}{
		{"ApparentPower", read("va").ApparentPower, 1500000},
		{"ReactivePower", read("var").ReactivePower, 230000},
		{"Frequency", read("hz").Frequency, 50020},
		{"PowerFactor %", read("pf").PowerFactor, -9550},
		{"PowerFactor ratio", read("ratio").PowerFactor, 8700},
	} {
		if c.got == nil || *c.got != c.want {
			t.Errorf("%s %v, want %d", c.name, c.got, c.want)
		}
	}
	if a := read("hz").Accuracy; len(a) != 1 || a[0].MeasurementType != epmdef.MeasurementTypeFrequency {
		t.Errorf("Accuracy %+v", a)
	}
}

func TestSensorFlowAndSoil(t *testing.T) {
	r := serve(t, sensor("flow", "volume_flow_rate", "L/min"), sensor("soil", "moisture", "%"))
	wantType(t, r, "flow", device.FlowSensorType(), eligibility.StateMappable)
	wantType(t, r, "soil", device.SoilSensorType(), eligibility.StateMappable)
	r.apply("flow", value(12.5)) // 0.75 m³/h
	r.apply("soil", value(41.6))
	if st, _ := flowdef.StateFrom(r.values("flow", flowdef.ClusterID)); st.MeasuredValue == nil || *st.MeasuredValue != 8 {
		t.Fatalf("flow MeasuredValue %v, want 8 (0.75 m³/h in 0.1 m³/h)", st.MeasuredValue)
	}
	st, _ := soildef.StateFrom(r.values("soil", soildef.ClusterID))
	if st.SoilMoistureMeasuredValue == nil || *st.SoilMoistureMeasuredValue != 42 ||
		st.SoilMoistureMeasurementLimits.MeasurementType != soildef.MeasurementTypeSoilMoisture ||
		st.SoilMoistureMeasurementLimits.MaxMeasuredValue != 100 {
		t.Fatalf("soil %+v", st)
	}
}

func TestBinarySensorCold(t *testing.T) {
	r := serve(t, binary("cold", "cold"))
	wantType(t, r, "cold", device.WaterFreezeDetectorType(), eligibility.StateMappable)
	r.apply("cold", value("on"))
	if st, _ := booleanstatedef.StateFrom(r.values("cold", booleanstatedef.ClusterID)); !st.StateValue {
		t.Fatal("cold on: StateValue false, want true")
	}
}
