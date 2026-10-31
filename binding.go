// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"

	colorcontroldef "github.com/SukramJ/go-fabric/cluster/spec/colorcontrol"
	"github.com/SukramJ/go-fabric/contract"
	"github.com/SukramJ/go-fabric/device"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"
)

// binding is one entity bound to its row: what the generated closures of
// rows_gen.go read the entity, its state and its device through.
type binding struct {
	plan   *Plan
	row    *Row
	entity model.Entity
	key    string
	// target is the device the row's clusters live on: the entity's own,
	// or for a row that rides, its primary's.
	target *device.Device
	// measure is the source of a row riding as a measurement.
	measure *measure
	// hostObj is the consumer's host object (Options.Ports), asked once
	// (hostAsked); noPorts names the first `ports: host` cluster it serves
	// no ports for (ports.go).
	hostObj   any
	hostAsked bool
	noPorts   string

	// mu guards last: the baseline of each event's value (events.go).
	mu   sync.Mutex
	last map[[2]int]string
}

// match reports whether the entity satisfies w.
func (b *binding) match(w *When) bool { return matches(b.entity, w) }

func matches(e model.Entity, w *When) bool {
	if w == nil {
		return true
	}
	d := e.Desc()
	if w.DeviceClass != nil && !slices.Contains(w.DeviceClass, string(d.DeviceClass)) {
		return false
	}
	if w.Unit != nil && !slices.Contains(w.Unit, string(d.Unit)) {
		return false
	}
	if w.StateClass != nil && !slices.Contains(w.StateClass, string(d.StateClass)) {
		return false
	}
	if w.Options != nil && *w.Options != (d.Options != nil && len(d.Options.Codes) > 0) {
		return false
	}
	if w.Features != nil && !w.Features.holds(features(e), true) {
		return false
	}
	for key, set := range w.Extra {
		// Present asks the entity; the lists and truth read the catalog's
		// default for a key the entity leaves out.
		own, stated := d.Extra[key]
		v, _ := extra(e, key)
		if !set.holds(list(v), stated && own != nil) {
			return false
		}
		if set.True != nil {
			t, _ := v.(bool)
			if t != *set.True {
				return false
			}
		}
	}
	return true
}

// holds tests the set against a list; present says whether the key was
// there at all.
func (s *Set) holds(l []string, present bool) bool {
	if s.Present != nil && *s.Present != present {
		return false
	}
	for _, x := range s.All {
		if !slices.Contains(l, x) {
			return false
		}
	}
	if len(s.Any) > 0 && !slices.ContainsFunc(s.Any, func(x string) bool { return slices.Contains(l, x) }) {
		return false
	}
	return !slices.ContainsFunc(s.None, func(x string) bool { return slices.Contains(l, x) })
}

// features returns the feature flag names an entity states in
// Extra["supported_features"]: an integer bit mask decoded with the
// catalog's flag values (go-ha-catalog data/features.json), or a list of
// flag names.
func features(e model.Entity) []string {
	v, ok := e.Desc().Extra[keyFeature]
	if !ok {
		return nil
	}
	if names := list(v); names != nil {
		out := make([]string, len(names))
		for i, n := range names {
			out[i] = strings.ToUpper(n)
		}
		return out
	}
	mask, ok := toInt(v)
	if !ok {
		return nil
	}
	flags, err := hacatalog.LoadFeatures()
	if err != nil {
		return nil
	}
	var out []string
	for name, bit := range flags[string(e.Platform())] {
		if bit != 0 && mask&int64(bit) == int64(bit) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// attr returns the raw value of an attribute mapping: from the state, or
// from the description for a mapping that reads no state (an initial
// value). ok is false when the mapping does not apply or the value is
// absent.
func (b *binding) attr(st *model.State, a *Attr) (any, bool) {
	if a.When != nil && !b.match(a.When) {
		return nil, false
	}
	if len(a.StateIn) > 0 {
		if st == nil {
			return nil, false
		}
		v, _ := lookupState(*st, fromState)
		s, _ := toText(v)
		if !slices.Contains(a.StateIn, s) {
			return nil, false
		}
	}
	if a.Value.HasConst {
		return a.Value.Const, true
	}
	if a.Value.From == fromState || strings.HasPrefix(a.Value.From, prefixSt) {
		if st == nil {
			return nil, false
		}
		return lookupState(*st, a.Value.From)
	}
	return describe(b.entity, a.Value.From)
}

func (b *binding) pick(ci, ai int, initial bool) *Attr {
	if ci < 0 {
		return &Attr{Name: "measurement", Value: b.row.RideMeasurement.Value}
	}
	c := &b.row.Clusters[ci]
	if initial {
		return &c.Initial[ai]
	}
	return &c.Attributes[ai]
}

// number is the converted value of attribute mapping ai of cluster ci.
func (b *binding) number(st *model.State, ci, ai int) (float64, bool) {
	a := b.pick(ci, ai, st == nil)
	raw, ok := b.attr(st, a)
	if !ok {
		return 0, false
	}
	x, ok := toFloat(raw)
	if !ok {
		return 0, false
	}
	return convert(b.entity, &a.Value, x)
}

// null reports whether attribute mapping ai reads a value that is there
// but no number: null, "unknown", "unavailable".
func (b *binding) null(st *model.State, ci, ai int) bool {
	a := b.pick(ci, ai, st == nil)
	raw, ok := b.attr(st, a)
	if !ok {
		return false
	}
	_, num := toFloat(raw)
	return !num
}

// text is the mapped string of attribute mapping ai, the key the
// generated switch matches.
func (b *binding) text(st *model.State, ci, ai int) string {
	a := b.pick(ci, ai, st == nil)
	raw, ok := b.attr(st, a)
	if !ok {
		return ""
	}
	s, _ := toText(raw)
	return s
}

// server is the device's server of a cluster.
func (b *binding) server(id uint32) contract.ClusterServer {
	if b.target == nil {
		return nil
	}
	return b.target.Server(id)
}

// changed reports whether a change from a command touched one of ids.
func changed(c contract.Change, ids ...uint32) bool {
	if c.Origin != contract.OriginCommand {
		return false
	}
	return slices.ContainsFunc(ids, func(id uint32) bool { return slices.Contains(c.Attributes, id) })
}

// emit delivers command k of cluster ci, its payload rendered from the
// fields f reads.
func (b *binding) emit(ci, k int, f func(name string) (any, bool)) {
	cmd := &b.row.Clusters[ci].Commands[k]
	if cmd.When != nil && !b.match(cmd.When) {
		return
	}
	payload, ok := b.render(cmd, f)
	if !ok {
		return
	}
	mc := model.Command{Entity: b.entity, Method: cmd.Method, Payload: payload}
	if cmd.Method == "" {
		mc.Role = model.RoleCommand
		if bd, ok := model.Bind(b.entity, model.RoleCommand); ok {
			mc.Slot = bd.Slot
		}
	}
	b.plan.deliver(mc)
}

// render is a command's payload: one value, or a JSON object.
func (b *binding) render(cmd *Command, f func(string) (any, bool)) ([]byte, bool) {
	if cmd.Raw != nil {
		v, ok := b.out(cmd.Raw, f)
		if !ok {
			return nil, false
		}
		return []byte(format(v)), true
	}
	obj := map[string]any{}
	for i := range cmd.JSON {
		fld := &cmd.JSON[i]
		v, ok := b.out(&fld.Value, f)
		if !ok {
			continue
		}
		setPath(obj, fld.Key, v)
	}
	derive(obj, cmd.Derive, f)
	if len(obj) == 0 {
		return nil, false
	}
	out, err := json.Marshal(obj)
	return out, err == nil
}

func setPath(obj map[string]any, key string, v any) {
	parts := strings.Split(key, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := obj[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			obj[p] = next
		}
		obj = next
	}
	obj[parts[len(parts)-1]] = v
}

// out is one payload value: a constant, the entity's value (a catalog
// default), or a command field converted for the entity.
func (b *binding) out(v *Value, f func(string) (any, bool)) (any, bool) {
	if v.HasConst {
		return v.Const, true
	}
	var raw any
	var ok bool
	if v.Field != "" {
		raw, ok = f(v.Field)
	} else {
		raw, ok = describe(b.entity, v.From)
	}
	if !ok || isNil(raw) {
		return nil, false
	}
	if v.Enum != nil {
		n, ok := toInt(raw)
		if !ok {
			return nil, false
		}
		s, ok := v.Enum[n]
		return s, ok
	}
	if s, isString := raw.(string); isString {
		return s, true
	}
	x, ok := toFloat(raw)
	if !ok {
		return nil, false
	}
	y, ok := convert(b.entity, v, x)
	if !ok {
		return nil, false
	}
	return round(y, v.Digits), true
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// derive adds the keys a conversion of several fields computes: the
// colour of an RGB light from the CIE x/y or the hue and saturation a
// controller set, or the hue and saturation of a light without xy from
// the CIE x/y, through go-fabric's colour conversions (matter.js
// ColorConversionUtils.ts xyToRgb, hsvToRgb; colorcontroldef.XYToRGB,
// HSVToRGB), at full brightness. CurrentX/Y are x/y times 65536,
// CurrentHue is the hue times 254/360 and CurrentSaturation the saturation
// times 254 (matter.js color-control.resource.ts:41-47, :58-64, :97-104 at
// 862ad75f).
func derive(obj map[string]any, name string, f func(string) (any, bool)) {
	num := func(field string) (float64, bool) {
		v, ok := f(field)
		if !ok {
			return 0, false
		}
		return toFloat(v)
	}
	var r, g, b float64
	switch name {
	case "rgb_from_xy":
		x, okx := num("CurrentX")
		y, oky := num("CurrentY")
		if !okx || !oky {
			return
		}
		r, g, b = colorcontroldef.XYToRGB(x/65536, y/65536)
	case "rgb_from_hs":
		h, okh := num("CurrentHue")
		s, oks := num("CurrentSaturation")
		if !okh || !oks {
			return
		}
		r, g, b = colorcontroldef.HSVToRGB(h*360/254, s/254, 1)
	case "hs_from_xy":
		x, okx := num("CurrentX")
		y, oky := num("CurrentY")
		if !okx || !oky {
			return
		}
		h, s := colorcontroldef.XYToHSV(x/65536, y/65536)
		setPath(obj, "color.h", round(h, 1))
		setPath(obj, "color.s", round(s*100, 1))
		return
	default:
		return
	}
	setPath(obj, "color.r", math.Round(r*255))
	setPath(obj, "color.g", math.Round(g*255))
	setPath(obj, "color.b", math.Round(b*255))
}
