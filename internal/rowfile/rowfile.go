// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

// Package rowfile reads the table's rows: rows/<platform>.yaml, the data
// form of go-hamapping's table. internal/gen generates the Go form from
// what it reads; docs/rows.md describes the format.
package rowfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// File is one rows/<platform>.yaml.
type File struct {
	// Name is the file's base name; Load sets it.
	Name     string `yaml:"-"`
	Platform string `yaml:"platform"`
	// Cites are the sources every row of the file reads (the catalog's
	// platform schema).
	Cites []string `yaml:"cites"`
	Rows  []Row    `yaml:"rows"`
}

// Row is one row (hamapping.Row).
type Row struct {
	ID string `yaml:"id"`
	// When is the predicate on the entity.
	When When `yaml:"when"`
	// Type is the device type's name as go-fabric's device package names
	// it (device.<Type>Type); empty for an unmappable row.
	Type string `yaml:"type"`
	// Verdict is "mappable", "partial: <reason>" or "unmappable: <reason>".
	Verdict  string    `yaml:"verdict"`
	Ride     string    `yaml:"ride"`
	Caveats  []Caveat  `yaml:"caveats"`
	Cites    []string  `yaml:"cites"`
	Clusters []Cluster `yaml:"clusters"`
	// RideMeasurement is how a riding row's value rides on the primary
	// endpoint: a go-fabric measurement source (device.Measurement) of a
	// class; its clusters serve an endpoint of its own.
	RideMeasurement *RideMeasurement `yaml:"ride_measurement"`
	// Parts are the component devices the device type requires.
	Parts []Part `yaml:"parts"`
}

// RideMeasurement is a measurement source a riding row declares on its
// primary.
type RideMeasurement struct {
	Class string `yaml:"class"`
	Value `yaml:",inline"`
}

// MeasurementClasses are the measurement classes a row may ride as, by
// go-fabric contract constant.
var MeasurementClasses = map[string]string{"battery": "MeasurementBattery"}

// Part is a component device: its type and declared clusters (features
// and constant initial values only).
type Part struct {
	Type     string    `yaml:"type"`
	Clusters []Cluster `yaml:"clusters"`
}

// Caveat turns a mappable row partial for the entities When matches.
type Caveat struct {
	When   When   `yaml:"when"`
	Reason string `yaml:"reason"`
}

// When is a predicate on an entity (hamapping.When). A device class
// "none" is an entity without one.
type When struct {
	DeviceClass []string       `yaml:"device_class"`
	Unit        []string       `yaml:"unit"`
	StateClass  []string       `yaml:"state_class"`
	Features    *Set           `yaml:"features"`
	Extra       map[string]Set `yaml:"extra"`
	Options     *bool          `yaml:"options"`
}

// Set is a membership test (hamapping.Set).
type Set struct {
	All     []string `yaml:"all"`
	Any     []string `yaml:"any"`
	None    []string `yaml:"none"`
	Present *bool    `yaml:"present"`
	True    *bool    `yaml:"true"`
}

// Cluster is one cluster of a row.
type Cluster struct {
	Cluster  string    `yaml:"cluster"`
	Features []Feature `yaml:"features"`
	Initial  []Attr    `yaml:"initial"`
	// StartNull are the attributes that start null (unknown).
	StartNull  []string  `yaml:"start_null"`
	Attributes []Attr    `yaml:"attributes"`
	Commands   []Command `yaml:"commands"`
	// Events emit the cluster's events when a value of the entity
	// changes.
	Events []Event `yaml:"events"`
	// Options materialize a list of the entity into the cluster's
	// attributes and its command (a cluster with a materializer).
	Options *Options `yaml:"options"`
	// Ports is "host" for a cluster whose kit server runs on ports the
	// consumer supplies (hamapping.Options.Ports): the row does not apply
	// to an entity whose host lacks one the cluster needs.
	Ports string `yaml:"ports"`
}

// PortsHost is the one value of Cluster.Ports.
const PortsHost = "host"

// Options is a list of strings of the entity, the value that names its
// current entry and the method a controller's choice goes to.
type Options struct {
	// List is "desc.options" (Description.Options' codes) or
	// "extra.<key>" (a list of strings in Description.Extra).
	List string `yaml:"list"`
	// Current is the value whose entry in List is the current one:
	// "state", "state.<key>" or "extra.<key>".
	Current string `yaml:"current"`
	// Method is the entity's method; the payload is the chosen string.
	Method string `yaml:"method"`
}

// Event emits events of a cluster when the value On names changes from
// a known value to another; the first value (after start or after the
// entity was unavailable) only sets the baseline.
type Event struct {
	// On is the value: "state", "state.<key>" or "extra.<key>".
	On string `yaml:"on"`
	// Emit are the events, emitted in order.
	Emit []Emit `yaml:"emit"`
}

// Emit is one event and its data, by field name of the event's
// structure in the cluster's <cluster>def package.
type Emit struct {
	Event   string         `yaml:"event"`
	Literal map[string]any `yaml:"literal"`
}

// Feature is a feature selection, a name or {name, when}.
type Feature struct {
	Name string `yaml:"name"`
	When *When  `yaml:"when"`
}

// UnmarshalYAML takes a plain name as a feature without a condition.
func (f *Feature) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		f.Name = n.Value
		return nil
	}
	type plain Feature
	return n.Decode((*plain)(f))
}

// Attr maps a value onto an attribute ("Attribute" or
// "Attribute.Field" of a structure attribute).
type Attr struct {
	Attribute string `yaml:"attribute"`
	Value     `yaml:",inline"`
	When      *When    `yaml:"when"`
	StateIn   []string `yaml:"state_in"`
	// Literal is an initial value as it stands (a list of structures).
	Literal any `yaml:"literal"`
}

// Value is a value source and its conversion (hamapping.Value).
type Value struct {
	From        string            `yaml:"from"`
	Field       string            `yaml:"field"`
	Const       any               `yaml:"const"`
	Units       map[string]Linear `yaml:"units"`
	DefaultUnit string            `yaml:"default_unit"`
	Offset      *float64          `yaml:"offset"`
	Scale       *float64          `yaml:"scale"`
	Add         *float64          `yaml:"add"`
	Func        string            `yaml:"func"`
	Min         *float64          `yaml:"min"`
	Max         *float64          `yaml:"max"`
	Digits      int               `yaml:"digits"`
	// Per divides the input by a value of the entity (extra.<key>, the
	// catalog's default when absent); Times multiplies the output by one.
	Per   string `yaml:"per"`
	Times string `yaml:"times"`
	// Enum maps the entity's string to a constant name (or a bool) for an
	// attribute, a constant name (or "true"/"false") to the entity's string
	// for a command value.
	Enum    map[string]any `yaml:"enum"`
	Catalog bool           `yaml:"catalog"`
}

// Linear is a linear conversion.
type Linear struct {
	Offset float64  `yaml:"offset"`
	Scale  *float64 `yaml:"scale"`
	Add    float64  `yaml:"add"`
}

// Command maps a controller's command or write onto the entity.
type Command struct {
	// Exactly one of Handle (a command name), Write (attribute names)
	// and Change (attribute names a command changes).
	Handle string   `yaml:"handle"`
	Write  []string `yaml:"write"`
	Change []string `yaml:"change"`
	// Unless skips a change that also touches one of these attributes.
	Unless []string `yaml:"unless"`
	// If restricts the command to a field value.
	If     *Cond            `yaml:"if"`
	When   *When            `yaml:"when"`
	Method string           `yaml:"method"`
	Raw    *Value           `yaml:"raw"`
	JSON   map[string]Value `yaml:"json"`
	// Derive adds keys a conversion of several fields computes (Derives).
	Derive string `yaml:"derive"`
	// Otherwise is a handler's response, by field name of its response
	// structure, when the if of none of the handler's commands holds: the
	// command is refused with a status of the cluster's own (a handle
	// with an if only).
	Otherwise map[string]any `yaml:"otherwise"`
}

// Derives are the derived payloads a command may add, with the fields
// each reads.
var Derives = map[string][]string{
	"rgb_from_xy": {"CurrentX", "CurrentY"},
	"rgb_from_hs": {"CurrentHue", "CurrentSaturation"},
	"hs_from_xy":  {"CurrentX", "CurrentY"},
}

// Cond tests a command field against constant names.
type Cond struct {
	Field string   `yaml:"field"`
	In    []string `yaml:"in"`
	NotIn []string `yaml:"not_in"`
}

// Verdicts.
const (
	Mappable   = "mappable"
	Partial    = "partial"
	Unmappable = "unmappable"
)

// Funcs are the conversion functions a value may name.
var Funcs = []string{"", "log10", "pow10", "reciprocal"}

// Load reads every rows/*.yaml of dir, by name.
func Load(dir string) ([]File, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	var out []File
	for _, name := range names {
		data, err := os.ReadFile(name) //nolint:gosec // the repository's row files
		if err != nil {
			return nil, err
		}
		f, err := Parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(name), err)
		}
		f.Name = filepath.Base(name)
		out = append(out, f)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("rowfile: no rows in %s", dir)
	}
	if err := unique(out); err != nil {
		return nil, err
	}
	return out, nil
}

// Parse reads one row file and checks its form; an unknown key is an
// error.
func Parse(data []byte) (File, error) {
	var f File
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		if errors.Is(err, io.EOF) {
			return File{}, errors.New("rowfile: empty file")
		}
		return File{}, fmt.Errorf("rowfile: %w", err)
	}
	if err := f.Check(); err != nil {
		return File{}, err
	}
	return f, nil
}

// Check reports the first defect of the file's form.
func (f *File) Check() error {
	if f.Platform == "" {
		return errors.New("rowfile: no platform")
	}
	if len(f.Rows) == 0 {
		return errors.New("rowfile: no rows")
	}
	seen := map[string]bool{}
	for i := range f.Rows {
		r := &f.Rows[i]
		if r.ID == "" || strings.ContainsAny(r.ID, "/ \t\n\"\\`") {
			return fmt.Errorf("rowfile: row %d: id %q", i, r.ID)
		}
		if seen[r.ID] {
			return fmt.Errorf("rowfile: row %s twice", r.ID)
		}
		seen[r.ID] = true
		if err := r.check(); err != nil {
			return fmt.Errorf("rowfile: row %s: %w", r.ID, err)
		}
	}
	return nil
}

// ParseVerdict splits a verdict into its state and reason.
func ParseVerdict(v string) (state, reason string, err error) {
	state, reason, _ = strings.Cut(v, ":")
	state, reason = strings.TrimSpace(state), strings.TrimSpace(reason)
	switch state {
	case Mappable:
		if reason != "" {
			return "", "", fmt.Errorf("verdict %q: a mappable row states no reason", v)
		}
	case Partial, Unmappable:
		if reason == "" {
			return "", "", fmt.Errorf("verdict %q: no reason", v)
		}
	default:
		return "", "", fmt.Errorf("verdict %q", v)
	}
	return state, reason, nil
}

func (r *Row) check() error {
	state, _, err := ParseVerdict(r.Verdict)
	if err != nil {
		return err
	}
	if state == Unmappable {
		if r.Type != "" || len(r.Clusters) > 0 || r.Ride != "" {
			return errors.New("an unmappable row names no type, clusters or ride")
		}
		return nil
	}
	if r.Type == "" {
		return errors.New("no type")
	}
	if len(r.Cites) == 0 {
		return errors.New("no cites")
	}
	if r.Ride != "" && r.Ride != "primary" {
		return fmt.Errorf("ride %q", r.Ride)
	}
	for i := range r.Caveats {
		if r.Caveats[i].Reason == "" {
			return fmt.Errorf("caveat %d: no reason", i)
		}
	}
	if m := r.RideMeasurement; m != nil {
		if r.Ride == "" {
			return errors.New("ride_measurement without ride")
		}
		if _, ok := MeasurementClasses[m.Class]; !ok {
			return fmt.Errorf("ride_measurement class %q", m.Class)
		}
		if err := m.check(false); err != nil {
			return fmt.Errorf("ride_measurement: %w", err)
		}
	}
	for i, p := range r.Parts {
		if p.Type == "" {
			return fmt.Errorf("part %d: no type", i)
		}
		for j := range p.Clusters {
			c := &p.Clusters[j]
			if len(c.Attributes) > 0 || len(c.Commands) > 0 || len(c.Events) > 0 || c.Options != nil || c.Ports != "" {
				return fmt.Errorf("part %s: cluster %s: a part declares no attributes, commands, events, options or ports", p.Type, c.Cluster)
			}
			for k := range c.Initial {
				if a := &c.Initial[k]; a.From != "" {
					return fmt.Errorf("part %s: initial %s: a part's initial values are constants", p.Type, a.Attribute)
				}
			}
			if err := c.check(); err != nil {
				return fmt.Errorf("part %s: cluster %s: %w", p.Type, c.Cluster, err)
			}
		}
	}
	names := map[string]bool{}
	for i := range r.Clusters {
		c := &r.Clusters[i]
		if c.Cluster == "" || names[c.Cluster] {
			return fmt.Errorf("cluster %d: name %q", i, c.Cluster)
		}
		names[c.Cluster] = true
		if err := c.check(); err != nil {
			return fmt.Errorf("cluster %s: %w", c.Cluster, err)
		}
	}
	return nil
}

func (c *Cluster) check() error {
	for _, f := range c.Features {
		if !ident(f.Name) {
			return fmt.Errorf("feature %q", f.Name)
		}
	}
	for i := range c.Initial {
		a := &c.Initial[i]
		if !attrName(a.Attribute) {
			return fmt.Errorf("initial %q", a.Attribute)
		}
		sources := 0
		if a.Literal != nil {
			sources++
		}
		if a.From != "" {
			sources++
		}
		if a.Const != nil {
			sources++
		}
		if sources != 1 {
			return fmt.Errorf("initial %s: one of literal, from and const", a.Attribute)
		}
		if strings.HasPrefix(a.From, "state") || a.Field != "" {
			return fmt.Errorf("initial %s: reads no state or field", a.Attribute)
		}
		if err := a.check(false); err != nil {
			return fmt.Errorf("initial %s: %w", a.Attribute, err)
		}
	}
	for _, n := range c.StartNull {
		if !ident(n) {
			return fmt.Errorf("null %q", n)
		}
	}
	for i := range c.Attributes {
		a := &c.Attributes[i]
		if !attrName(a.Attribute) || a.Literal != nil || a.Field != "" {
			return fmt.Errorf("attribute %q", a.Attribute)
		}
		if a.From == "" && a.Const == nil {
			return fmt.Errorf("attribute %s: no from", a.Attribute)
		}
		if err := a.check(false); err != nil {
			return fmt.Errorf("attribute %s: %w", a.Attribute, err)
		}
	}
	for i := range c.Commands {
		if err := c.Commands[i].check(); err != nil {
			return fmt.Errorf("command %d: %w", i, err)
		}
	}
	for i := range c.Events {
		if err := c.Events[i].check(); err != nil {
			return fmt.Errorf("event %d: %w", i, err)
		}
	}
	if o := c.Options; o != nil {
		if k, ok := strings.CutPrefix(o.List, "extra."); o.List != "desc.options" && (!ok || !valueFrom(o.List) || k == "") {
			return fmt.Errorf("options: list %q: desc.options or extra.<key>", o.List)
		}
		if !valueFrom(o.Current) {
			return fmt.Errorf("options: current %q: state, state.<key> or extra.<key>", o.Current)
		}
		if !ident(o.Method) {
			return fmt.Errorf("options: method %q", o.Method)
		}
	}
	if c.Ports != "" && c.Ports != PortsHost {
		return fmt.Errorf("ports %q: the one value is %s", c.Ports, PortsHost)
	}
	return nil
}

func (e *Event) check() error {
	if !valueFrom(e.On) {
		return fmt.Errorf("on %q: state, state.<key> or extra.<key>", e.On)
	}
	if len(e.Emit) == 0 {
		return errors.New("no emit")
	}
	for _, x := range e.Emit {
		if !ident(x.Event) {
			return fmt.Errorf("event %q", x.Event)
		}
		for k := range x.Literal {
			if !ident(k) {
				return fmt.Errorf("event %s: field %q", x.Event, k)
			}
		}
	}
	return nil
}

// valueFrom reports whether s names a value of the entity a change can
// be seen on: "state", "state.<key>" or "extra.<key>".
func valueFrom(s string) bool {
	if s == "state" {
		return true
	}
	for _, p := range []string{"state.", "extra."} {
		if k, ok := strings.CutPrefix(s, p); ok {
			return k != "" && !strings.Contains(k, "..")
		}
	}
	return false
}

func (cmd *Command) check() error {
	kinds := 0
	if cmd.Handle != "" {
		kinds++
		if !ident(cmd.Handle) {
			return fmt.Errorf("handle %q", cmd.Handle)
		}
	}
	if len(cmd.Write) > 0 {
		kinds++
	}
	if len(cmd.Change) > 0 {
		kinds++
	}
	if kinds != 1 {
		return errors.New("one of handle, write and change")
	}
	if len(cmd.Unless) > 0 && len(cmd.Change) == 0 {
		return errors.New("unless is a change's")
	}
	for _, n := range slices.Concat(cmd.Write, cmd.Change, cmd.Unless) {
		if !ident(n) {
			return fmt.Errorf("attribute %q", n)
		}
	}
	if cmd.If != nil && (!ident(cmd.If.Field) || (len(cmd.If.In) == 0) == (len(cmd.If.NotIn) == 0)) {
		return errors.New("if: a field and one of in and not_in")
	}
	if cmd.If != nil {
		for _, n := range slices.Concat(cmd.If.In, cmd.If.NotIn) {
			if !ident(n) && !number(n) {
				return fmt.Errorf("if: constant %q", n)
			}
		}
	}
	if cmd.Otherwise != nil {
		if cmd.Handle == "" || cmd.If == nil {
			return errors.New("otherwise: a handle's with an if")
		}
		for k := range cmd.Otherwise {
			if !ident(k) {
				return fmt.Errorf("otherwise: field %q", k)
			}
		}
	}
	if cmd.Derive != "" {
		if _, ok := Derives[cmd.Derive]; !ok || cmd.Raw != nil {
			return fmt.Errorf("derive %q: a json payload's", cmd.Derive)
		}
	}
	if (cmd.Raw == nil) == (len(cmd.JSON) == 0 && cmd.Derive == "") {
		return errors.New("one of raw and json")
	}
	if cmd.Raw != nil {
		return cmd.Raw.check(true)
	}
	for k := range cmd.JSON {
		if k == "" || strings.Contains(k, "..") {
			return fmt.Errorf("json key %q", k)
		}
		v := cmd.JSON[k]
		if err := v.check(true); err != nil {
			return fmt.Errorf("json %s: %w", k, err)
		}
	}
	return nil
}

func (v *Value) check(command bool) error {
	if !slices.Contains(Funcs, v.Func) {
		return fmt.Errorf("func %q", v.Func)
	}
	if command {
		sources := 0
		for _, set := range []bool{v.From != "", v.Field != "", v.Const != nil} {
			if set {
				sources++
			}
		}
		if sources != 1 {
			return errors.New("one of from, field and const")
		}
		if v.Field != "" && !ident(v.Field) {
			return fmt.Errorf("field %q", v.Field)
		}
		for k := range v.Enum {
			if k != "true" && k != "false" && !ident(k) {
				return fmt.Errorf("enum key %q", k)
			}
		}
	} else {
		for _, x := range v.Enum {
			switch t := x.(type) {
			case bool, int:
			case string:
				if !ident(t) {
					return fmt.Errorf("enum constant %q", t)
				}
			default:
				return fmt.Errorf("enum value %v", x)
			}
		}
	}
	switch c := v.Const.(type) {
	case nil, string, bool, int, float64:
	default:
		return fmt.Errorf("const %v (%T)", c, c)
	}
	if v.Units != nil && (v.Offset != nil || v.Scale != nil || v.Add != nil) {
		return errors.New("units replace offset, scale and add")
	}
	for _, src := range []string{v.Per, v.Times} {
		if src != "" && !strings.HasPrefix(src, "extra.") {
			return fmt.Errorf("per/times %q: an extra.<key>", src)
		}
	}
	if v.Digits < 0 || v.Digits > 9 {
		return fmt.Errorf("digits %d", v.Digits)
	}
	return nil
}

// ident reports whether s is a Go identifier the generator may emit.
func ident(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func number(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func attrName(s string) bool {
	a, b, dotted := strings.Cut(s, ".")
	return ident(a) && (!dotted || ident(b))
}

// unique refuses a row id or a platform that occurs twice.
func unique(files []File) error {
	platforms := map[string]string{}
	for _, f := range files {
		if prev, dup := platforms[f.Platform]; dup {
			return fmt.Errorf("rowfile: platform %s in %s and %s", f.Platform, prev, f.Name)
		}
		platforms[f.Platform] = f.Name
	}
	return nil
}
