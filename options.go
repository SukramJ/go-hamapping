// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/SukramJ/go-hamqtt/model"
	modeselectdef "thingfabric.dev/go-fabric/cluster/spec/modeselect"
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

// The limits a list must fit: a mode is a uint8 (ModeOptionStruct.Mode,
// cluster/spec/modeselect/definition_gen.go:56 at v0.15.4); a label and
// the description at most 64 (:55, :229).
const (
	maxModes     = 256
	maxLabel     = 64
	selectNoName = "Select"
)

// unmaterializable names why a row's options cannot be materialized for
// the entity, or returns "".
func unmaterializable(e model.Entity, row *Row) string {
	for ci := range row.Clusters {
		c := &row.Clusters[ci]
		if c.Options == nil {
			continue
		}
		if why := checkOptions(c.ID, optionList(e, c.Options)); why != "" {
			return why
		}
	}
	return ""
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
	modes := make([]modeselectdef.ModeOptionStruct, len(l))
	for i, s := range l {
		modes[i] = modeselectdef.ModeOptionStruct{Label: cut(s, maxLabel), Mode: uint8(i)} //nolint:gosec // len(l) <= maxModes (unmaterializable)
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
