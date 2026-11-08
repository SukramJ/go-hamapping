// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"github.com/SukramJ/go-fabric/contract"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-hamqtt/model"
)

// Row is one row of the table: a Home Assistant platform entity matching
// When becomes a device of Type with the clusters the row declares. The
// rows are data (rows/<platform>.yaml); rows_gen.go is their Go form,
// written by internal/gen and held equal to the YAML by a test.
type Row struct {
	// ID names the row, "<platform>/<name>".
	ID string
	// Platform is the Home Assistant platform the row reads.
	Platform string
	// When is the predicate on the entity; a platform's rows are tried in
	// order and the first that matches decides.
	When When
	// Type is the device type; nil for an unmappable row.
	Type *device.Type
	// State is the row's verdict: Mappable, Partial or Unmappable.
	State eligibility.State
	// Reason explains a Partial or Unmappable verdict; {device_class} is
	// replaced with the entity's device class.
	Reason string
	// Caveats turn a Mappable row Partial for the entities they match.
	Caveats []Caveat
	// Ride names how the row's clusters ride on another endpoint of the
	// same device ("primary": on the device's first mappable endpoint
	// that does not ride itself); empty for an endpoint of its own.
	Ride string
	// Cites are the sources of the row's facts: the matter.js element of
	// the device type, the catalog lines the predicate reads.
	Cites []string
	// Clusters are the clusters the row declares.
	Clusters []Cluster
	// RideMeasurement is how a riding row's value rides on its primary:
	// a measurement source (device.Measurement) the kit serves as the
	// cluster of its class. nil for a row that rides by its clusters.
	RideMeasurement *Measurement
	// Parts are the component devices the device type requires.
	Parts []Part
}

// Measurement is a measurement source a riding row declares on its
// primary endpoint, and the value it carries.
type Measurement struct {
	Class contract.MeasurementClass
	Value Value
}

// Part is a component device of a row's device: a device type and its
// declared clusters.
type Part struct {
	Type     *device.Type
	Clusters []Cluster
}

// Caveat is a condition under which a Mappable row maps only in part.
type Caveat struct {
	When   When
	Reason string
}

// Cluster is one cluster of a row: its declaration, the attribute mapping
// from the entity's state and the command mapping back to the entity.
type Cluster struct {
	// Name is the cluster's name as go-fabric's definition states it.
	Name string
	// ID is the cluster id, from the generated <cluster>def package.
	ID uint32
	// Features are the feature names the row selects, with their
	// conditions.
	Features []Feature
	// Attributes map the entity's state onto attributes.
	Attributes []Attr
	// Initial are the values the declaration starts with that the entity's
	// description gives (a device class, a range).
	Initial []Attr
	// Commands map a controller's command or write back to the entity.
	Commands []Command
	// Events emit the cluster's events when a value of the entity
	// changes.
	Events []Event
	// Options is a list of the entity a materializer turns into the
	// cluster's attributes and command (options.go); nil for none.
	Options *OptionList

	declare func(b *binding, ci int) device.Declaration
	apply   func(b *binding, ci int, st model.State) (any, error)
	watch   func(b *binding, ci int) func()
	fire    func(b *binding, ei int, endpoint uint16) error
}

// OptionList is a list of strings of the entity — a select's options, a
// climate's preset modes — that a cluster-specific materializer
// (options.go) turns into the cluster's attributes, the entry Current
// names into its current one, and a controller's choice into a
// model.Command of Method with the chosen string as the payload.
type OptionList struct {
	// List is "desc.options" (Description.Options' codes, in order) or
	// "extra.<key>" (a list of strings in Description.Extra).
	List string
	// Current is the value naming the current entry: "state",
	// "state.<key>" or "extra.<key>".
	Current string
	// Method is the entity's method a choice goes to.
	Method string
}

// Event emits events of a cluster, in order, through the cluster's typed
// server (<cluster>def.Server.Emit<Event>) when the value On names
// changes from a known value to another non-empty one. The first value —
// after the plan is built, or after the entity was unavailable or its
// value unknown — only sets the baseline: a value that was there before
// is no event.
type Event struct {
	// On is the value: "state", "state.<key>" or "extra.<key>".
	On string
	// Emit names the events, as the cluster's definition does.
	Emit []string
}

// Feature is one feature a cluster declaration selects, when When holds
// (nil: always).
type Feature struct {
	Name string
	When *When
}

// Attr maps a value of the entity onto one attribute (or one field of a
// structure attribute, "Attribute.Field").
type Attr struct {
	Name  string
	Value Value
	// When restricts the mapping to entities it matches (nil: every one).
	When *When
	// StateIn restricts the mapping to states whose primary value is one
	// of these (a climate's hvac mode); empty for every state.
	StateIn []string
}

// Command maps a controller's command or write onto a model.Command.
type Command struct {
	// Kind is how the command is caught: "handle" (the cluster's command
	// handler), "write" (the write veto) or "change" (a change the
	// cluster's server made running a command).
	Kind string
	// On names the command (handle) or the attributes (write, change).
	On []string
	// Method is the entity's named action; empty for its command binding.
	Method string
	// Raw is the payload as one value; JSON a payload object, by dotted
	// key.
	Raw  *Value
	JSON []Field
	// When restricts the command to entities it matches.
	When *When
	// Derive adds the payload keys a conversion of several fields
	// computes: "rgb_from_xy", "rgb_from_hs" (color.r, color.g, color.b),
	// "hs_from_xy" (color.h, color.s).
	Derive string
	// Discard marks a handled command the entity has no action for: it is
	// answered with the cluster's success and reaches [Plan.OnDiscard]
	// instead of [Plan.OnChange].
	Discard bool
}

// Field is one key of a JSON payload.
type Field struct {
	Key   string
	Value Value
}

// Value is where a value comes from and how it converts. The pipeline is
// y = Func(x + Offset) * Scale + Add, clamped to [Min, Max]; Units
// replaces Offset, Scale and Add by the entity's unit.
type Value struct {
	// From is the entity's value: "state" (the primary state value),
	// "state.<key>" (a key of a map state, else of State.Extra),
	// "extra.<key>" (a key of Description.Extra, else the catalog's
	// default for the platform), "device_class", "unit", "desc.min",
	// "desc.max", "desc.step".
	From string
	// Field is the command's field or the cluster's state field a command
	// value reads.
	Field string
	// Const is a constant value; HasConst marks it set.
	Const    any
	HasConst bool
	// Units are the linear conversions by unit; DefaultUnit is the unit of
	// an entity that states none.
	Units       map[string]Linear
	DefaultUnit string
	Linear
	// Func is "" (identity), "log10", "pow10" or "reciprocal".
	Func     string
	Min, Max *float64
	// Digits is the number of decimals a numeric payload value keeps.
	Digits int
	// Per divides the input by a value of the entity (extra.<key>);
	// Times multiplies the output by one.
	Per, Times string
	// Enum maps a command field's value (an enum constant or a bool, by
	// its integer) to the entity's string.
	Enum map[int64]string
	// Catalog marks an extra.<key> whose absence reads the catalog's
	// default for the key.
	Catalog bool
}

// Linear is the linear part of a conversion.
type Linear struct {
	Offset, Scale, Add float64
}

// When is a predicate on an entity. Every field set must hold.
type When struct {
	// DeviceClass lists device classes; "" is an entity without one.
	DeviceClass []string
	Unit        []string
	StateClass  []string
	// Features tests the platform's feature flags (the catalog's
	// features.json names) the entity states in Extra["supported_features"].
	Features *Set
	// Extra tests keys of Description.Extra (list membership, presence,
	// truth), by the catalog's key names.
	Extra map[string]Set
	// Options tests whether the entity carries options.
	Options *bool
}

// Set tests a list (or a single value) for membership.
type Set struct {
	All, Any, None []string
	// Present requires the entity to state the key (true) or not to
	// (false); a catalog default does not count. The lists and True read
	// the catalog's default for a key the entity leaves out.
	Present *bool
	// True requires a true value (a boolean key such as color_temp_kelvin).
	True *bool
}
