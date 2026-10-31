// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package gen

import (
	"reflect"

	airqualitydef "github.com/SukramJ/go-fabric/cluster/spec/airquality"
	booleanstatedef "github.com/SukramJ/go-fabric/cluster/spec/booleanstate"
	cameraavstreammanagementdef "github.com/SukramJ/go-fabric/cluster/spec/cameraavstreammanagement"
	carbondioxideconcentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/carbondioxideconcentrationmeasurement"
	carbonmonoxideconcentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/carbonmonoxideconcentrationmeasurement"
	colorcontroldef "github.com/SukramJ/go-fabric/cluster/spec/colorcontrol"
	doorlockdef "github.com/SukramJ/go-fabric/cluster/spec/doorlock"
	electricalenergymeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/electricalenergymeasurement"
	electricalpowermeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/electricalpowermeasurement"
	fancontroldef "github.com/SukramJ/go-fabric/cluster/spec/fancontrol"
	flowmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/flowmeasurement"
	illuminancemeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/illuminancemeasurement"
	keypadinputdef "github.com/SukramJ/go-fabric/cluster/spec/keypadinput"
	levelcontroldef "github.com/SukramJ/go-fabric/cluster/spec/levelcontrol"
	mediaplaybackdef "github.com/SukramJ/go-fabric/cluster/spec/mediaplayback"
	modeselectdef "github.com/SukramJ/go-fabric/cluster/spec/modeselect"
	nitrogendioxideconcentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/nitrogendioxideconcentrationmeasurement"
	occupancysensingdef "github.com/SukramJ/go-fabric/cluster/spec/occupancysensing"
	onoffdef "github.com/SukramJ/go-fabric/cluster/spec/onoff"
	ozoneconcentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/ozoneconcentrationmeasurement"
	pm10concentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/pm10concentrationmeasurement"
	pm1concentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/pm1concentrationmeasurement"
	pm25concentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/pm25concentrationmeasurement"
	powersourcedef "github.com/SukramJ/go-fabric/cluster/spec/powersource"
	powertopologydef "github.com/SukramJ/go-fabric/cluster/spec/powertopology"
	pressuremeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/pressuremeasurement"
	radonconcentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/radonconcentrationmeasurement"
	relativehumiditymeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/relativehumiditymeasurement"
	rvcoperationalstatedef "github.com/SukramJ/go-fabric/cluster/spec/rvcoperationalstate"
	rvcrunmodedef "github.com/SukramJ/go-fabric/cluster/spec/rvcrunmode"
	smokecoalarmdef "github.com/SukramJ/go-fabric/cluster/spec/smokecoalarm"
	soilmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/soilmeasurement"
	switchclusterdef "github.com/SukramJ/go-fabric/cluster/spec/switchcluster"
	temperaturemeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/temperaturemeasurement"
	thermostatdef "github.com/SukramJ/go-fabric/cluster/spec/thermostat"
	totalvolatileorganiccompoundsconcentrationmeasurementdef "github.com/SukramJ/go-fabric/cluster/spec/totalvolatileorganiccompoundsconcentrationmeasurement"
	valveconfigurationandcontroldef "github.com/SukramJ/go-fabric/cluster/spec/valveconfigurationandcontrol"
	webrtctransportproviderdef "github.com/SukramJ/go-fabric/cluster/spec/webrtctransportprovider"
	windowcoveringdef "github.com/SukramJ/go-fabric/cluster/spec/windowcovering"
)

// def is what the generator knows of one go-fabric <cluster>def package:
// its import path and alias, the cluster's name and id, and the types it
// reads field names and field types from.
type def struct {
	name    string
	id      uint32
	path    string
	alias   string
	declare reflect.Type
	patch   reflect.Type
	state   reflect.Type
	// server is the typed server (*<cluster>def.Server), whose
	// Emit<Event> methods name the cluster's events and their data.
	server reflect.Type
}

const specPath = "github.com/SukramJ/go-fabric/cluster/spec/"

func newDef[D, P, S, V any](name string, id uint32, pkg string) def {
	return def{
		name: name, id: id, path: specPath + pkg, alias: pkg + "def",
		declare: reflect.TypeFor[D](), patch: reflect.TypeFor[P](), state: reflect.TypeFor[S](),
		server: reflect.TypeFor[*V](),
	}
}

// defs are the cluster packages rows may name. A row naming another
// cluster fails generation until its package is listed here.
var defs = func() map[string]def {
	all := []def{
		newDef[airqualitydef.Declare, airqualitydef.Patch, airqualitydef.State, airqualitydef.Server](airqualitydef.Name, airqualitydef.ClusterID, "airquality"),
		newDef[carbondioxideconcentrationmeasurementdef.Declare, carbondioxideconcentrationmeasurementdef.Patch, carbondioxideconcentrationmeasurementdef.State, carbondioxideconcentrationmeasurementdef.Server](carbondioxideconcentrationmeasurementdef.Name, carbondioxideconcentrationmeasurementdef.ClusterID, "carbondioxideconcentrationmeasurement"),
		newDef[carbonmonoxideconcentrationmeasurementdef.Declare, carbonmonoxideconcentrationmeasurementdef.Patch, carbonmonoxideconcentrationmeasurementdef.State, carbonmonoxideconcentrationmeasurementdef.Server](carbonmonoxideconcentrationmeasurementdef.Name, carbonmonoxideconcentrationmeasurementdef.ClusterID, "carbonmonoxideconcentrationmeasurement"),
		newDef[pm1concentrationmeasurementdef.Declare, pm1concentrationmeasurementdef.Patch, pm1concentrationmeasurementdef.State, pm1concentrationmeasurementdef.Server](pm1concentrationmeasurementdef.Name, pm1concentrationmeasurementdef.ClusterID, "pm1concentrationmeasurement"),
		newDef[pm25concentrationmeasurementdef.Declare, pm25concentrationmeasurementdef.Patch, pm25concentrationmeasurementdef.State, pm25concentrationmeasurementdef.Server](pm25concentrationmeasurementdef.Name, pm25concentrationmeasurementdef.ClusterID, "pm25concentrationmeasurement"),
		newDef[pm10concentrationmeasurementdef.Declare, pm10concentrationmeasurementdef.Patch, pm10concentrationmeasurementdef.State, pm10concentrationmeasurementdef.Server](pm10concentrationmeasurementdef.Name, pm10concentrationmeasurementdef.ClusterID, "pm10concentrationmeasurement"),
		newDef[totalvolatileorganiccompoundsconcentrationmeasurementdef.Declare, totalvolatileorganiccompoundsconcentrationmeasurementdef.Patch, totalvolatileorganiccompoundsconcentrationmeasurementdef.State, totalvolatileorganiccompoundsconcentrationmeasurementdef.Server](totalvolatileorganiccompoundsconcentrationmeasurementdef.Name, totalvolatileorganiccompoundsconcentrationmeasurementdef.ClusterID, "totalvolatileorganiccompoundsconcentrationmeasurement"),
		newDef[nitrogendioxideconcentrationmeasurementdef.Declare, nitrogendioxideconcentrationmeasurementdef.Patch, nitrogendioxideconcentrationmeasurementdef.State, nitrogendioxideconcentrationmeasurementdef.Server](nitrogendioxideconcentrationmeasurementdef.Name, nitrogendioxideconcentrationmeasurementdef.ClusterID, "nitrogendioxideconcentrationmeasurement"),
		newDef[ozoneconcentrationmeasurementdef.Declare, ozoneconcentrationmeasurementdef.Patch, ozoneconcentrationmeasurementdef.State, ozoneconcentrationmeasurementdef.Server](ozoneconcentrationmeasurementdef.Name, ozoneconcentrationmeasurementdef.ClusterID, "ozoneconcentrationmeasurement"),
		newDef[radonconcentrationmeasurementdef.Declare, radonconcentrationmeasurementdef.Patch, radonconcentrationmeasurementdef.State, radonconcentrationmeasurementdef.Server](radonconcentrationmeasurementdef.Name, radonconcentrationmeasurementdef.ClusterID, "radonconcentrationmeasurement"),
		newDef[flowmeasurementdef.Declare, flowmeasurementdef.Patch, flowmeasurementdef.State, flowmeasurementdef.Server](flowmeasurementdef.Name, flowmeasurementdef.ClusterID, "flowmeasurement"),
		newDef[soilmeasurementdef.Declare, soilmeasurementdef.Patch, soilmeasurementdef.State, soilmeasurementdef.Server](soilmeasurementdef.Name, soilmeasurementdef.ClusterID, "soilmeasurement"),
		newDef[switchclusterdef.Declare, switchclusterdef.Patch, switchclusterdef.State, switchclusterdef.Server](switchclusterdef.Name, switchclusterdef.ClusterID, "switchcluster"),
		newDef[rvcrunmodedef.Declare, rvcrunmodedef.Patch, rvcrunmodedef.State, rvcrunmodedef.Server](rvcrunmodedef.Name, rvcrunmodedef.ClusterID, "rvcrunmode"),
		newDef[rvcoperationalstatedef.Declare, rvcoperationalstatedef.Patch, rvcoperationalstatedef.State, rvcoperationalstatedef.Server](rvcoperationalstatedef.Name, rvcoperationalstatedef.ClusterID, "rvcoperationalstate"),
		newDef[valveconfigurationandcontroldef.Declare, valveconfigurationandcontroldef.Patch, valveconfigurationandcontroldef.State, valveconfigurationandcontroldef.Server](valveconfigurationandcontroldef.Name, valveconfigurationandcontroldef.ClusterID, "valveconfigurationandcontrol"),
		newDef[mediaplaybackdef.Declare, mediaplaybackdef.Patch, mediaplaybackdef.State, mediaplaybackdef.Server](mediaplaybackdef.Name, mediaplaybackdef.ClusterID, "mediaplayback"),
		newDef[keypadinputdef.Declare, keypadinputdef.Patch, keypadinputdef.State, keypadinputdef.Server](keypadinputdef.Name, keypadinputdef.ClusterID, "keypadinput"),
		newDef[booleanstatedef.Declare, booleanstatedef.Patch, booleanstatedef.State, booleanstatedef.Server](booleanstatedef.Name, booleanstatedef.ClusterID, "booleanstate"),
		newDef[colorcontroldef.Declare, colorcontroldef.Patch, colorcontroldef.State, colorcontroldef.Server](colorcontroldef.Name, colorcontroldef.ClusterID, "colorcontrol"),
		newDef[doorlockdef.Declare, doorlockdef.Patch, doorlockdef.State, doorlockdef.Server](doorlockdef.Name, doorlockdef.ClusterID, "doorlock"),
		newDef[electricalenergymeasurementdef.Declare, electricalenergymeasurementdef.Patch, electricalenergymeasurementdef.State, electricalenergymeasurementdef.Server](electricalenergymeasurementdef.Name, electricalenergymeasurementdef.ClusterID, "electricalenergymeasurement"),
		newDef[electricalpowermeasurementdef.Declare, electricalpowermeasurementdef.Patch, electricalpowermeasurementdef.State, electricalpowermeasurementdef.Server](electricalpowermeasurementdef.Name, electricalpowermeasurementdef.ClusterID, "electricalpowermeasurement"),
		newDef[fancontroldef.Declare, fancontroldef.Patch, fancontroldef.State, fancontroldef.Server](fancontroldef.Name, fancontroldef.ClusterID, "fancontrol"),
		newDef[illuminancemeasurementdef.Declare, illuminancemeasurementdef.Patch, illuminancemeasurementdef.State, illuminancemeasurementdef.Server](illuminancemeasurementdef.Name, illuminancemeasurementdef.ClusterID, "illuminancemeasurement"),
		newDef[levelcontroldef.Declare, levelcontroldef.Patch, levelcontroldef.State, levelcontroldef.Server](levelcontroldef.Name, levelcontroldef.ClusterID, "levelcontrol"),
		newDef[modeselectdef.Declare, modeselectdef.Patch, modeselectdef.State, modeselectdef.Server](modeselectdef.Name, modeselectdef.ClusterID, "modeselect"),
		newDef[occupancysensingdef.Declare, occupancysensingdef.Patch, occupancysensingdef.State, occupancysensingdef.Server](occupancysensingdef.Name, occupancysensingdef.ClusterID, "occupancysensing"),
		newDef[onoffdef.Declare, onoffdef.Patch, onoffdef.State, onoffdef.Server](onoffdef.Name, onoffdef.ClusterID, "onoff"),
		newDef[powersourcedef.Declare, powersourcedef.Patch, powersourcedef.State, powersourcedef.Server](powersourcedef.Name, powersourcedef.ClusterID, "powersource"),
		newDef[powertopologydef.Declare, powertopologydef.Patch, powertopologydef.State, powertopologydef.Server](powertopologydef.Name, powertopologydef.ClusterID, "powertopology"),
		newDef[pressuremeasurementdef.Declare, pressuremeasurementdef.Patch, pressuremeasurementdef.State, pressuremeasurementdef.Server](pressuremeasurementdef.Name, pressuremeasurementdef.ClusterID, "pressuremeasurement"),
		newDef[relativehumiditymeasurementdef.Declare, relativehumiditymeasurementdef.Patch, relativehumiditymeasurementdef.State, relativehumiditymeasurementdef.Server](relativehumiditymeasurementdef.Name, relativehumiditymeasurementdef.ClusterID, "relativehumiditymeasurement"),
		newDef[smokecoalarmdef.Declare, smokecoalarmdef.Patch, smokecoalarmdef.State, smokecoalarmdef.Server](smokecoalarmdef.Name, smokecoalarmdef.ClusterID, "smokecoalarm"),
		newDef[temperaturemeasurementdef.Declare, temperaturemeasurementdef.Patch, temperaturemeasurementdef.State, temperaturemeasurementdef.Server](temperaturemeasurementdef.Name, temperaturemeasurementdef.ClusterID, "temperaturemeasurement"),
		newDef[thermostatdef.Declare, thermostatdef.Patch, thermostatdef.State, thermostatdef.Server](thermostatdef.Name, thermostatdef.ClusterID, "thermostat"),
		newDef[cameraavstreammanagementdef.Declare, cameraavstreammanagementdef.Patch, cameraavstreammanagementdef.State, cameraavstreammanagementdef.Server](cameraavstreammanagementdef.Name, cameraavstreammanagementdef.ClusterID, "cameraavstreammanagement"),
		newDef[webrtctransportproviderdef.Declare, webrtctransportproviderdef.Patch, webrtctransportproviderdef.State, webrtctransportproviderdef.Server](webrtctransportproviderdef.Name, webrtctransportproviderdef.ClusterID, "webrtctransportprovider"),
		newDef[windowcoveringdef.Declare, windowcoveringdef.Patch, windowcoveringdef.State, windowcoveringdef.Server](windowcoveringdef.Name, windowcoveringdef.ClusterID, "windowcovering"),
	}
	m := make(map[string]def, len(all))
	for _, d := range all {
		m[d.name] = d
	}
	return m
}()

// aliasOf is the alias of a def package by import path.
func aliasOf(path string) (string, bool) {
	for _, d := range defs {
		if d.path == path {
			return d.alias, true
		}
	}
	return "", false
}
