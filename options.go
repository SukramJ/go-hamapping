// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"sukramj.dev/go-hamqtt/model"
	modeselectdef "thingfabric.dev/go-fabric/cluster/spec/modeselect"
	thermostatdef "thingfabric.dev/go-fabric/cluster/spec/thermostat"
	"thingfabric.dev/go-fabric/contract"
)

// The materializers of a row's options (Cluster.Options): a list of the
// entity becomes the cluster's attributes and a controller's choice a
// model.Command of the options' method, its payload the chosen string.
// The generated code calls b.options<Cluster> from the declaration and
// b.options<Cluster>Apply from the state mapping; unmaterializable is
// the verdict's check at Map time.

// fromOptions is Options.List for Description.Options' codes.
const fromOptions = "desc.options"

// statusInvalidCommand is the Interaction Model status INVALID_COMMAND a
// materializer answers with (contract.StatusError), go-fabric
// im/status.go:72 at v0.15.4.
const statusInvalidCommand uint8 = 0x85

// optionList is the list o names for the entity.
func optionList(e model.Entity, o *OptionList) []string {
	if o.List == fromOptions {
		if d := e.Desc().Options; d != nil {
			return d.Codes
		}
		return nil
	}
	key, _ := strings.CutPrefix(o.List, prefixEx)
	return list(e.Desc().Extra[key])
}

// optionCurrent is the entry the state names, as the entity spells it.
func (b *binding) optionCurrent(st *model.State, o *OptionList) (string, bool) {
	var raw any
	var ok bool
	if o.Current == fromState || strings.HasPrefix(o.Current, prefixSt) {
		raw, ok = lookupState(*st, o.Current)
	} else {
		raw, ok = describe(b.entity, o.Current)
	}
	s, isString := raw.(string)
	return s, ok && isString
}

// send delivers a choice to the entity's method.
func (b *binding) send(method, payload string) {
	b.plan.deliver(model.Command{Entity: b.entity, Method: method, Payload: []byte(payload)})
}

// cut shortens s to at most n bytes, between two characters: go-fabric
// bounds a string by its UTF-8 bytes since v0.24.0, as matter.js does
// since #4995 (spec.StringLength; go-fabric CHANGELOG [0.24.0]). s is
// valid UTF-8 (a source's strings come from JSON); n is not negative.
func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// The limits a list must fit: SupportedModes holds at most 255 entries
// (cluster/spec/modeselect/definition_gen.go:262 at v0.25.0, "max 255"),
// so every mode 0..254 is a uint8 (ModeOptionStruct.Mode, :56); a label
// and the description at most 64 bytes (:55, :245).
const (
	maxModes     = 255
	maxLabel     = 64
	selectNoName = "Select"
)

// unmaterializable names why a row's options cannot be materialized for
// the entity, or returns "".
func unmaterializable(e model.Entity, row *Row) string {
	for ci := range row.Clusters {
		c := &row.Clusters[ci]
		if c.ID == thermostatdef.ClusterID {
			if why := setpointBounds(e, row, ci); why != "" {
				return why
			}
		}
		if c.Options == nil {
			continue
		}
		if why := checkOptions(c.ID, optionList(e, c.Options)); why != "" {
			return why
		}
	}
	return ""
}

// autoDeadband is the heat/cool deadband go-fabric's Thermostat server
// keeps under AutoMode, in 0.01 °C: its MinSetpointDeadBand default 20
// (0.1 °C) times ten (go-fabric cluster/thermo/thermostat_server.go:135-139
// and setpoints.go:65-67 at v0.27.0). The rows declare no
// MinSetpointDeadBand.
const autoDeadband = 200

// setpointBounds names why a Thermostat entity's setpoint limits cannot be
// built, or returns "". The limits are the row's AbsMin/AbsMax
// SetpointLimit mappings of cluster ci, a missing min_temp or max_temp
// read as Home Assistant's default for it (the mapping's Absent), each
// compared as the int16 go-fabric is given. go-fabric refuses the device
// (cluster/thermo/setpoints.go:315-352, 449-554 at v0.27.0) when, on a
// served side, the lower limit is above the upper — the bounds are not
// swapped, which one is wrong is the source's to say — and, with
// AutoMode, when the span is below the deadband: a span of at least the
// deadband always repairs, a narrower one never does.
func setpointBounds(e model.Entity, row *Row, ci int) string {
	b := &binding{entity: e, row: row}
	c := &row.Clusters[ci]
	auto := false
	for _, f := range c.Features {
		if f.Name == "AutoMode" && b.match(f.When) {
			auto = true
		}
	}
	for _, side := range []string{"Heat", "Cool"} {
		lo, okLo := b.limit(c, "AbsMin"+side+"SetpointLimit")
		hi, okHi := b.limit(c, "AbsMax"+side+"SetpointLimit")
		if !okLo || !okHi {
			continue
		}
		switch {
		case lo.n > hi.n:
			return fmt.Sprintf("min_temp %s is above max_temp %s", lo.text, hi.text)
		case auto && int(hi.n)-int(lo.n) < autoDeadband:
			return fmt.Sprintf("min_temp %s and max_temp %s are closer than the thermostat's 2.0 °C heat/cool deadband", lo.text, hi.text)
		}
	}
	return ""
}

// boundValue is a setpoint limit as go-fabric is given it, and as the
// entity states it for a reason.
type boundValue struct {
	n    int16
	text string
}

// limit is the value of the cluster's initial mapping of attr that
// applies to the entity.
func (b *binding) limit(c *Cluster, attr string) (boundValue, bool) {
	for i := range c.Initial {
		a := &c.Initial[i]
		if a.Name != attr || !b.applies(nil, a) {
			continue
		}
		y, ok := b.attrNumber(nil, a)
		if !ok {
			return boundValue{}, false
		}
		raw, stated := b.attr(nil, a)
		text := fmt.Sprint(raw)
		if q := a.Value.Absent; !stated && q != nil {
			text = fmt.Sprintf("%v %s (Home Assistant's default)", q.Value, q.Unit)
		}
		return boundValue{n: conv[int16](y), text: text}, true
	}
	return boundValue{}, false
}

// checkOptions is the check of one materializer's list.
func checkOptions(cluster uint32, l []string) string {
	if cluster == modeselectdef.ClusterID {
		if len(l) == 0 {
			return "a select without options has no modes"
		}
		if len(l) > maxModes {
			return fmt.Sprintf("a select with %d options has more than ModeSelect's %d modes", len(l), maxModes)
		}
	}
	return ""
}

// --- ModeSelect ---------------------------------------------------------

// optionsModeSelect serves a select on ModeSelect's generated server: one
// mode per option, its index the mode (0-based, as matter-hub's
// mode-select-server.ts:10-16 at 664575a), the label the option; the
// description the entity's name; ChangeToMode for a known mode delivers
// the option, for another answers INVALID_COMMAND. The kit's server needs
// a live ModeSource (device/defaults_declared.go:152-160 at v0.15.4);
// without one the declaration keeps the generated server, which the
// handler and the Initial serve.
func (b *binding) optionsModeSelect(d *modeselectdef.Declare, ci int) {
	o := b.row.Clusters[ci].Options
	l := slices.Clone(optionList(b.entity, o))
	labels := modeLabels(l)
	modes := make([]modeselectdef.ModeOptionStruct, len(l))
	for i := range l {
		modes[i] = modeselectdef.ModeOptionStruct{Label: labels[i], Mode: uint8(i)} //nolint:gosec // len(l) <= maxModes (unmaterializable)
	}
	desc := cut(b.name(), maxLabel)
	if desc == "" {
		desc = selectNoName
	}
	var first uint8
	d.Initial.SupportedModes = &modes
	d.Initial.Description = &desc
	d.Initial.CurrentMode = &first
	d.Initial.Null = append(d.Initial.Null, modeselectdef.AttrStandardNamespace)
	d.HandleChangeToMode = func(_ context.Context, req modeselectdef.ChangeToModeRequest) (struct{}, error) {
		if int(req.NewMode) >= len(l) {
			return struct{}{}, contract.StatusError{Status: statusInvalidCommand, Message: fmt.Sprintf("mode %d of %d", req.NewMode, len(l))}
		}
		b.send(o.Method, l[req.NewMode])
		return struct{}{}, nil
	}
}

// modeLabels are the options' labels: each cut to maxLabel bytes, and a
// label equal to an earlier one numbered " (n)" (n = 2, 3, … until it is
// new), its base cut so the whole fits. ModeSelect does not require
// distinct labels, but a controller shows only the label; the option sent
// is the list's by index either way.
func modeLabels(l []string) []string {
	out := make([]string, len(l))
	seen := make(map[string]bool, len(l))
	next := map[string]int{}
	for i, s := range l {
		base := cut(s, maxLabel)
		label := base
		for n := max(next[base], 2); seen[label]; n++ {
			suffix := fmt.Sprintf(" (%d)", n)
			label = cut(base, maxLabel-len(suffix)) + suffix
			next[base] = n + 1
		}
		seen[label] = true
		out[i] = label
	}
	return out
}

// optionsModeSelectApply sets CurrentMode to the index of the current
// option, 0 when the state names none of the list.
func (b *binding) optionsModeSelectApply(p *modeselectdef.Patch, st *model.State, ci int) int {
	o := b.row.Clusters[ci].Options
	cur, _ := b.optionCurrent(st, o)
	mode := uint8(max(slices.Index(optionList(b.entity, o), cur), 0)) //nolint:gosec // an index below maxModes
	p.CurrentMode = &mode
	return 1
}

// name is the entity's name as the endpoint's label takes it: the
// source's verbatim name (Options.Name), else the description's.
func (b *binding) name() string {
	if f := b.plan.opts.Name; f != nil {
		if n, verbatim := f(b.entity); verbatim && n != "" {
			return n
		}
	}
	return b.entity.Desc().Name.Default
}
