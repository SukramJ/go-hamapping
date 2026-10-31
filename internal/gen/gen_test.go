// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package gen

import (
	"strings"
	"testing"

	"github.com/SukramJ/go-hamapping/internal/rowfile"
)

const base = `platform: lock
rows:
  - id: l
    when: {}
    type: DoorLock
    verdict: mappable
    cites: [x]
    clusters:
      - cluster: DoorLock
        features: [PinCredential]
        initial: [{attribute: ActuatorEnabled, const: true}]
        start_null: [LockState]
        attributes:
          - {attribute: LockState, from: state, enum: {locked: LockStateLocked}}
        commands:
          - handle: LockDoor
            raw: {const: LOCK}

          - write: [ActuatorEnabled]
            if: {field: ActuatorEnabled, in: ["true"]}
            raw: {field: LockState, enum: {LockStateLocked: locked}}
          - change: [LockState]
            if: {field: LockState, not_in: [LockStateLocked]}
            method: m
            json: {a.b: {field: ActuatorEnabled, enum: {"true": "on"}}}
        events:
          - on: state
            emit: [{event: DoorLockAlarm, literal: {AlarmCode: AlarmCodeLockJammed}}]
`

func generate(t *testing.T, src string) error {
	t.Helper()
	f, err := rowfile.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	f.Name = "x.yaml"
	_, err = GenerateFiles([]rowfile.File{f})
	return err
}

func TestGenerateRefuses(t *testing.T) {
	for name, edit := range map[string][2]string{
		"device type":          {"type: DoorLock", "type: Toaster"},
		"cluster not typed":    {"cluster: DoorLock", "cluster: OnOff"},
		"cluster unknown":      {"cluster: DoorLock", "cluster: Toaster"},
		"attribute":            {"attribute: LockState, from", "attribute: Lockstate, from"},
		"initial":              {"attribute: ActuatorEnabled", "attribute: Nope"},
		"initial type":         {"ActuatorEnabled, const: true", "ActuatorEnabled, const: 3"},
		"start_null":           {"start_null: [LockState]", "start_null: [Nope]"},
		"handler":              {"handle: LockDoor", "handle: Explode"},
		"if field":             {"field: ActuatorEnabled, in", "field: Nope, in"},
		"if bool constant":     {`in: ["true"]`, `in: ["LockStateLocked"]`},
		"number to string":     {"attribute: LockState, from: state, enum: {locked: LockStateLocked}", "attribute: Language, from: state"},
		"command field":        {"raw: {const: LOCK}", "raw: {field: Nope}"},
		"write veto field":     {"write: [ActuatorEnabled]", "write: [Nope]"},
		"change attribute":     {"change: [LockState]", "change: [Nope]"},
		"event":                {"event: DoorLockAlarm", "event: Explosion"},
		"event field":          {"literal: {AlarmCode: AlarmCodeLockJammed}", "literal: {Nope: 1}"},
		"no materializer":      {"        events:\n", "        options: {list: desc.options, current: state, method: m}\n        events:\n"},
		"no port materializer": {"        events:\n", "        ports: host\n        events:\n"},
		"part type":            {"    clusters:\n      - cluster: DoorLock", "    parts: [{type: Toaster}]\n    clusters:\n      - cluster: DoorLock"},
	} {
		t.Run(name, func(t *testing.T) {
			src := strings.Replace(base, edit[0], edit[1], 1)
			if src == base {
				t.Fatalf("edit %q does not apply", edit[0])
			}
			if err := generate(t, src); err == nil {
				t.Fatal("generated")
			}
		})
	}
	if err := generate(t, base); err != nil {
		t.Fatalf("the base: %v", err)
	}
}

func TestGenerateOptions(t *testing.T) {
	src := `platform: select
rows:
  - id: s
    when: {}
    type: ModeSelect
    verdict: mappable
    cites: [x]
    clusters:
      - cluster: ModeSelect
        options: {list: desc.options, current: state, method: select_option}
`
	f, err := rowfile.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateFiles([]rowfile.File{f})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"b.optionsModeSelect(&d, ci)", "n += b.optionsModeSelectApply(&p, &st, ci)", "MethodSelectOption"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("no %q", want)
		}
	}
	if !strings.Contains(Table([]rowfile.File{f}), "[options desc.options at state to select_option]") {
		t.Error("the table names no options")
	}
}

func TestGeneratePorts(t *testing.T) {
	src := `platform: camera
rows:
  - id: c
    when: {}
    type: Camera
    verdict: mappable
    cites: [x]
    clusters:
      - cluster: CameraAvStreamManagement
        features: [Video, Audio, Snapshot]
        ports: host
      - cluster: WebRtcTransportProvider
        ports: host
`
	f, err := rowfile.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateFiles([]rowfile.File{f})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"if !b.portsAvStream(&d) {", "if !b.portsWebRTCProvider(&d) {", "b.unserved(ci)"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("no %q", want)
		}
	}
	if !strings.Contains(Table([]rowfile.File{f}), "WebRtcTransportProvider [ports: host]") {
		t.Error("the table names no ports")
	}
	// A cluster without a port materializer is refused by name.
	err = generate(t, strings.Replace(base, "        events:\n", "        ports: host\n        events:\n", 1))
	if err == nil || !strings.Contains(err.Error(), "DoorLock has no port materializer") {
		t.Errorf("ports on DoorLock: %v", err)
	}
}

const vacuum = `platform: vacuum
rows:
  - id: v
    when: {}
    type: RoboticVacuumCleaner
    verdict: mappable
    cites: [x]
    clusters:
      - cluster: RvcRunMode
        commands:
          - handle: ChangeToMode
            if: {field: NewMode, in: ["0", "1"]}
            method: run_mode
            raw: {field: NewMode}
            otherwise: {Status: ModeChangeStatusUnsupportedMode}
`

func TestGenerateOtherwise(t *testing.T) {
	out, err := GenerateFiles([]rowfile.File{parse(t, vacuum)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "resp = rvcrunmodedef.ChangeToModeResponse{Status: rvcrunmodedef.ModeChangeStatusUnsupportedMode}") {
		t.Fatalf("no otherwise response:\n%s", out)
	}
	for name, edit := range map[string][2]string{
		"field": {"{Status: ModeChangeStatusUnsupportedMode}", "{Nope: 1}"},
		"twice": {"            otherwise: {Status: ModeChangeStatusUnsupportedMode}\n", "            otherwise: {Status: ModeChangeStatusUnsupportedMode}\n          - handle: ChangeToMode\n            if: {field: NewMode, in: [\"2\"]}\n            raw: {const: x}\n            otherwise: {Status: ModeChangeStatusUnsupportedMode}\n"},
	} {
		if _, err := GenerateFiles([]rowfile.File{parse(t, strings.Replace(vacuum, edit[0], edit[1], 1))}); err == nil {
			t.Errorf("%s: generated", name)
		}
	}
	noFields := strings.Replace(base, "          - handle: LockDoor\n            raw: {const: LOCK}",
		"          - handle: LockDoor\n            if: {field: PinCode, in: [\"1\"]}\n            raw: {const: LOCK}\n            otherwise: {Status: X}", 1)
	if err := generate(t, noFields); err == nil {
		t.Error("an otherwise on a status-only command generated")
	}
}

func parse(t *testing.T, src string) rowfile.File {
	t.Helper()
	f, err := rowfile.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f
}

func TestGenerateRepository(t *testing.T) {
	src, err := Generate("../../rows")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "DO NOT EDIT") || !strings.Contains(string(src), "var rows = []Row{") {
		t.Fatal("no table")
	}
	if _, err := Generate(t.TempDir()); err == nil {
		t.Fatal("generated from nothing")
	}
}

func TestTableAndSplice(t *testing.T) {
	files, err := rowfile.Load("../../rows")
	if err != nil {
		t.Fatal(err)
	}
	table := Table(files)
	if !strings.Contains(table, "`light/extended_color`") || !strings.Contains(table, "rides as the battery measurement") {
		t.Fatalf("table:\n%s", table)
	}
	page := []byte("a\n" + TableBegin + "\nold\n" + TableEnd + "\nb\n")
	out, err := Splice(page, "new\n")
	if err != nil || !strings.Contains(string(out), "new") || strings.Contains(string(out), "old") {
		t.Fatalf("%s %v", out, err)
	}
	if _, err := Splice([]byte("no markers"), table); err == nil {
		t.Fatal("spliced without markers")
	}
}

func TestMethodConst(t *testing.T) {
	for in, want := range map[string]string{"tilt": "MethodTilt", "preset_mode": "MethodPresetMode", "temperature_high": "MethodTemperatureHigh", "a__b": "MethodAB"} {
		if got := MethodConst(in); got != want {
			t.Errorf("MethodConst(%q) = %q; want %q", in, got, want)
		}
	}
	if methodsSource(nil) != "" {
		t.Error("no rows: want no constant block")
	}
	if _, err := SpliceBetween([]byte("no markers"), KeysBegin, KeysEnd, ""); err == nil {
		t.Error("a page without the key markers spliced")
	}
}
