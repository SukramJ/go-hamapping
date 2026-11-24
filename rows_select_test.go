// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"
	modeselectdef "thingfabric.dev/go-fabric/cluster/spec/modeselect"
	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/eligibility"
	"thingfabric.dev/go-fabric/fabrictest"

	"github.com/SukramJ/go-hamapping"
)

// statusOf is the IM status of a refused invoke, 0 for none.
func statusOf(err error) uint8 {
	var se *fabrictest.StatusError
	if errors.As(err, &se) {
		return uint8(se.Status)
	}
	return 0
}

func TestSelect(t *testing.T) {
	e := entity("sel", hacatalog.PlatformSelect, model.Description{
		Name: model.L("Program"), Options: &model.Enum{Codes: []string{"eco", "normal", "intensive"}},
	})
	r := serve(t, e)
	wantType(t, r, "sel", device.ModeSelectType(), eligibility.StateMappable)
	r.apply("sel", value("normal"))
	st, _ := modeselectdef.StateFrom(r.values("sel", modeselectdef.ClusterID))
	if st.CurrentMode != 1 || len(st.SupportedModes) != 3 || st.SupportedModes[2].Label != "intensive" ||
		st.SupportedModes[2].Mode != 2 || st.Description != "Program" || st.StandardNamespace != nil {
		t.Fatalf("mode select %+v", st)
	}
	r.invoke("sel", modeselectdef.ClusterID, modeselectdef.CmdChangeToMode, modeselectdef.ChangeToModeRequest{NewMode: 2})
	expect(t, r.command(), hamapping.MethodSelectOption, "intensive")
	_, err := fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep("sel"), modeselectdef.ClusterID, modeselectdef.CmdChangeToMode),
		modeselectdef.ChangeToModeRequest{NewMode: 3})
	if statusOf(err) != 0x85 {
		t.Fatalf("ChangeToMode 3: %v, want INVALID_COMMAND", err)
	}
	r.apply("sel", value("unknown"))
	if st, _ := modeselectdef.StateFrom(r.values("sel", modeselectdef.ClusterID)); st.CurrentMode != 0 {
		t.Fatalf("CurrentMode %d for a state outside the options, want 0", st.CurrentMode)
	}
	r.noMore()
}

func TestSelectWithoutOptions(t *testing.T) {
	_, v := hamapping.Map(testDevice, []model.Entity{entity("sel", hacatalog.PlatformSelect, model.Description{})})
	if v[0].State != eligibility.StateUnmappable || v[0].Reason != "a select without options has no modes" {
		t.Fatalf("verdict %+v", v[0])
	}
}

func options(n int, prefix string) []string {
	l := make([]string, n)
	for i := range l {
		l[i] = fmt.Sprintf("%s%03d", prefix, i)
	}
	return l
}

// TestSelectModeLimit: SupportedModes holds at most 255 entries
// (modeselect definition_gen.go:262): 255 options build, 256 are
// unmappable with a reason (HR-P2-adpt-34). What Map calls mappable builds.
func TestSelectModeLimit(t *testing.T) {
	for _, n := range []int{255, 256} {
		e := entity("sel", hacatalog.PlatformSelect, model.Description{Options: &model.Enum{Codes: options(n, "o")}})
		p, v := hamapping.Map(testDevice, []model.Entity{e})
		_, err := p.Build(context.Background())
		p.Close()
		switch {
		case n == 255 && (v[0].State != eligibility.StateMappable || err != nil):
			t.Errorf("255 options: %+v, build %v", v[0], err)
		case n == 256 && (v[0].State != eligibility.StateUnmappable || v[0].Reason != "a select with 256 options has more than ModeSelect's 255 modes"):
			t.Errorf("256 options: %+v (build %v), want unmappable", v[0], err)
		}
	}
}

// TestSelectRepeatedLabels: labels equal after the 64-byte cut, or equal
// options, are numbered " (n)" within 64 bytes on a character boundary;
// the option sent stays the list's by index (HR-P2-adpt-36).
func TestSelectRepeatedLabels(t *testing.T) {
	long := options(255, strings.Repeat("ä", 40)) // 80 bytes before the number
	for _, l := range [][]string{long, {"a", "a", "a (2)", "b"}} {
		e := entity("sel", hacatalog.PlatformSelect, model.Description{Options: &model.Enum{Codes: l}})
		r := serve(t, e)
		st, _ := modeselectdef.StateFrom(r.values("sel", modeselectdef.ClusterID))
		seen := map[string]bool{}
		for _, m := range st.SupportedModes {
			if len(m.Label) > 64 || !utf8.ValidString(m.Label) || seen[m.Label] {
				t.Fatalf("mode %d label %q: over 64 bytes, invalid or repeated", m.Mode, m.Label)
			}
			seen[m.Label] = true
		}
		if len(seen) != len(l) {
			t.Fatalf("%d labels, want %d", len(seen), len(l))
		}
		if l[1] == "a" && (st.SupportedModes[1].Label != "a (2)" || st.SupportedModes[2].Label != "a (2) (2)" || st.SupportedModes[3].Label != "b") {
			t.Fatalf("labels %+v", st.SupportedModes)
		}
		if l[1] != "a" && st.SupportedModes[1].Label != cutTo(l[1], 60)+" (2)" {
			t.Fatalf("label 1 %q", st.SupportedModes[1].Label)
		}
		r.invoke("sel", modeselectdef.ClusterID, modeselectdef.CmdChangeToMode, modeselectdef.ChangeToModeRequest{NewMode: 2})
		expect(t, r.command(), hamapping.MethodSelectOption, l[2])
		r.noMore()
	}
}

// cutTo is the longest prefix of s within n bytes on a character boundary.
func cutTo(s string, n int) string {
	for n > 0 && n < len(s) && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:min(n, len(s))]
}
