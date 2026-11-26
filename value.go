// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
)

// Value sources of [Value].From.
const (
	fromState  = "state"
	prefixSt   = "state."
	prefixEx   = "extra."
	fromClass  = "device_class"
	fromUnit   = "unit"
	fromMin    = "desc.min"
	fromMax    = "desc.max"
	fromStep   = "desc.step"
	keyFeature = "supported_features"
)

// lookupState returns the value a state carries under from: the primary
// value ("state") or a key of a map value, else of State.Extra. A key
// that is present with a nil value is found.
func lookupState(st model.State, from string) (any, bool) {
	m, isMap := st.Value.(map[string]any)
	if from == fromState {
		if isMap {
			v, ok := m["state"]
			return v, ok
		}
		return st.Value, st.Value != nil || st.Available
	}
	key, ok := strings.CutPrefix(from, prefixSt)
	if !ok {
		return nil, false
	}
	if isMap {
		if v, ok := path(m, key); ok {
			return v, true
		}
	}
	return path(st.Extra, key)
}

// path walks a dotted key through nested maps.
func path(m map[string]any, key string) (any, bool) {
	var cur any = m
	for part := range strings.SplitSeq(key, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = mm[part]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// extra returns Description.Extra[key], else the catalog's default for the
// platform's key (go-ha-catalog data/mqtt.json, the platform's keys and its
// variants' keys).
func extra(e model.Entity, key string) (any, bool) {
	if v, ok := e.Desc().Extra[key]; ok && v != nil {
		return v, true
	}
	return catalogDefault(string(e.Platform()), key)
}

// catalogDefault is the default the catalog states for a platform key.
func catalogDefault(platform, key string) (any, bool) {
	mqtt, err := hacatalog.LoadMQTT()
	if err != nil {
		return nil, false
	}
	schema, ok := mqtt.Platforms[platform]
	if !ok {
		return nil, false
	}
	if k, ok := schema.Keys[key]; ok && k.Default != nil {
		return k.Default, true
	}
	for _, name := range slices.Sorted(maps.Keys(schema.Variants)) {
		if k, ok := schema.Variants[name].Keys[key]; ok && k.Default != nil {
			return k.Default, true
		}
	}
	return nil, false
}

// describe returns a value of the entity's description.
func describe(e model.Entity, from string) (any, bool) {
	d := e.Desc()
	switch from {
	case fromClass:
		return string(d.DeviceClass), true
	case fromUnit:
		return string(d.Unit), d.Unit != ""
	case fromMin:
		return deref(d.Min)
	case fromMax:
		return deref(d.Max)
	case fromStep:
		return deref(d.Step)
	}
	if key, ok := strings.CutPrefix(from, prefixEx); ok {
		return extra(e, key)
	}
	return nil, false
}

func describeNumber(e model.Entity, from string) (float64, bool) {
	v, ok := describe(e, from)
	if !ok {
		return 0, false
	}
	return toFloat(v)
}

func deref(p *float64) (any, bool) {
	if p == nil {
		return nil, false
	}
	return *p, true
}

// list reads a value as a list of strings: a []string, a []any of
// strings, or one string.
func list(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{t}
	}
	return nil
}

// toFloat reads a number: any integer or float kind, a json.Number or a
// numeric string.
func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case nil:
		return 0, false
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
	case bool:
		return 0, false
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		return f, !math.IsNaN(f) && !math.IsInf(f, 0)
	case reflect.Bool:
		if rv.Bool() {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// toInt reads an enum constant or a bool as its integer.
func toInt(v any) (int64, bool) {
	if b, ok := v.(bool); ok {
		if b {
			return 1, true
		}
		return 0, true
	}
	f, ok := toFloat(v)
	return int64(f), ok
}

// toText reads a value as the lower-case string a row's map matches: a
// bool and its "true"/"false" spellings read "on"/"off", as Home
// Assistant's binary states.
func toText(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", false
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		switch s {
		case "true":
			return "on", true
		case "false":
			return "off", true
		}
		return s, s != ""
	case bool:
		if t {
			return "on", true
		}
		return "off", true
	}
	if f, ok := toFloat(v); ok {
		return strconv.FormatFloat(f, 'f', -1, 64), true
	}
	return "", false
}

// unitOf is the unit a value converts from: the entity's, else the row's
// default.
func unitOf(e model.Entity, v *Value) string {
	if u := string(e.Desc().Unit); u != "" {
		return u
	}
	return v.DefaultUnit
}

// convert runs the pipeline of v over x for the entity's unit.
func convert(e model.Entity, v *Value, x float64) (float64, bool) {
	return convertIn(e, v, x, unitOf(e, v))
}

// convertIn runs the pipeline of v over x stated in unit.
func convertIn(e model.Entity, v *Value, x float64, unit string) (float64, bool) {
	lin := v.Linear
	if v.Units != nil {
		u, ok := v.Units[unit]
		if !ok {
			return 0, false
		}
		lin = u
	}
	if v.Per != "" {
		d, ok := describeNumber(e, v.Per)
		if !ok || d == 0 {
			return 0, false
		}
		x /= d
	}
	y := x + lin.Offset
	switch v.Func {
	case "":
	case "log10":
		if y <= 0 {
			y = math.Inf(-1)
		} else {
			y = math.Log10(y)
		}
	case "pow10":
		y = math.Pow(10, y)
	case "reciprocal":
		if y == 0 {
			return 0, false
		}
		y = 1 / y
	default:
		return 0, false
	}
	if lin.Scale == 0 {
		lin.Scale = 1 // an omitted scale; a zero scale is never a conversion
	}
	y = y*lin.Scale + lin.Add
	if v.Times != "" {
		m, ok := describeNumber(e, v.Times)
		if !ok {
			return 0, false
		}
		y *= m
	}
	if v.Min != nil && y < *v.Min {
		y = *v.Min
	}
	if v.Max != nil && y > *v.Max {
		y = *v.Max
	}
	if math.IsNaN(y) || math.IsInf(y, 0) {
		return 0, false
	}
	return y, true
}

// integer is the Go type of an integer attribute.
type integer interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// conv rounds v to T, clamped to T's range.
func conv[T integer](v float64) T {
	var zero T
	bits := unsafe.Sizeof(zero) * 8
	hi := ^zero
	lo := zero
	if ^zero < 0 {
		hi = T(1)<<(bits-1) - 1
		lo = -hi - 1
	}
	r := math.Round(v)
	switch {
	case r <= float64(lo):
		return lo
	case r >= float64(hi):
		return hi
	}
	if lo < 0 {
		return T(int64(r))
	}
	return T(uint64(r))
}

// round keeps digits decimals.
func round(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}

// format is a payload value's text.
func format(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	}
	return fmt.Sprint(v)
}
