// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"errors"
	"testing"

	modeselectdef "github.com/SukramJ/go-fabric/cluster/spec/modeselect"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-fabric/fabrictest"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

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
