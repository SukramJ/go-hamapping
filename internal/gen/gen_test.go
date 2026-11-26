// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package gen

import (
	"strings"
	"testing"

	"sukramj.dev/go-hamapping/internal/rowfile"
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
		"device type":            {"type: DoorLock", "type: Toaster"},
		"cluster not typed":      {"cluster: DoorLock", "cluster: OnOff"},
		"cluster unknown":        {"cluster: DoorLock", "cluster: Toaster"},
		"attribute":              {"attribute: LockState, from", "attribute: Lockstate, from"},
		"initial":                {"attribute: ActuatorEnabled", "attribute: Nope"},
		"initial type":           {"ActuatorEnabled, const: true", "ActuatorEnabled, const: 3"},
		"start_null":             {"start_null: [LockState]", "start_null: [Nope]"},
		"handler":                {"handle: LockDoor", "handle: Explode"},
		"if field":               {"field: ActuatorEnabled, in", "field: Nope, in"},
		"if bool constant":       {`in: ["true"]`, `in: ["LockStateLocked"]`},
		"enum null not nullable": {"attribute: LockState, from: state, enum: {locked: LockStateLocked}", "attribute: ActuatorEnabled, from: state, enum: {locked: null}"},
		"number to string":       {"attribute: LockState, from: state, enum: {locked: LockStateLocked}", "attribute: Language, from: state"},
		"command field":          {"raw: {const: LOCK}", "raw: {field: Nope}"},
		"write veto field":       {"write: [ActuatorEnabled]", "write: [Nope]"},
		"change attribute":       {"change: [LockState]", "change: [Nope]"},
		"event":                  {"event: DoorLockAlarm", "event: Explosion"},
		"event field":            {"literal: {AlarmCode: AlarmCodeLockJammed}", "literal: {Nope: 1}"},
		"no materializer":        {"        events:\n", "        options: {list: desc.options, current: state, method: m}\n        events:\n"},
		"no port materializer":   {"        events:\n", "        ports: host\n        events:\n"},
		"part type":              {"    clusters:\n      - cluster: DoorLock", "    parts: [{type: Toaster}]\n    clusters:\n      - cluster: DoorLock"},
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

func TestGenerateRowPort(t *testing.T) {
	lock := "          - handle: LockDoor\n            raw: {const: LOCK}"
	rowPort := strings.Replace(base, "      - cluster: DoorLock\n", "      - cluster: DoorLock\n        ports: row\n", 1)
	out, err := GenerateFiles([]rowfile.File{parse(t, rowPort)})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"b.rowPortsDoorLock(&d, ci)", "b.rowPortsDoorLockApply(ci, p)", `Ports: "row"`} {
		if !strings.Contains(strings.Join(strings.Fields(s), " "), want) { // gofmt aligns the literal's values
			t.Errorf("no %s:\n%s", want, s)
		}
	}
	if strings.Contains(s, "d.HandleLockDoor") {
		t.Error("a handler registered on a row-port cluster")
	}
	withIf := strings.Replace(rowPort, lock, "          - handle: LockDoor\n            if: {field: PinCode, in: [\"1\"]}\n            raw: {const: LOCK}", 1)
	if err := generate(t, withIf); err == nil {
		t.Error("a handle with an if under ports: row generated")
	}
	if _, err := GenerateFiles([]rowfile.File{parse(t, strings.Replace(vacuum, "      - cluster: RvcRunMode\n", "      - cluster: RvcRunMode\n        ports: row\n", 1))}); err == nil {
		t.Error("ports: row on a cluster without a row port generated")
	}
}

func TestGenerateStateIf(t *testing.T) {
	lock := "          - handle: LockDoor\n            raw: {const: LOCK}"
	src := strings.Replace(base, lock, "          - handle: LockDoor\n            if: {state: ActuatorEnabled, in: [\"true\"]}\n            raw: {const: LOCK}", 1)
	out, err := GenerateFiles([]rowfile.File{parse(t, src)})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"st := doorlockdef.Wrap(b.server(doorlockdef.ClusterID)).Get()", "st.ActuatorEnabled"} {
		if !strings.Contains(s, want) {
			t.Errorf("no %s:\n%s", want, s)
		}
	}
	if err := generate(t, strings.Replace(src, "state: ActuatorEnabled", "state: Nope", 1)); err == nil {
		t.Error("an if on an unknown state attribute generated")
	}
}

func TestGenerateEnumNull(t *testing.T) {
	src := strings.Replace(base, "enum: {locked: LockStateLocked}}", "enum: {locked: LockStateLocked, jammed: null}}", 1)
	out, err := GenerateFiles([]rowfile.File{parse(t, src)})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"case \"jammed\":", "p.Null = append(p.Null, doorlockdef.AttrLockState)",
		"p.Null = drop(p.Null, doorlockdef.AttrLockState)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("no %s:\n%s", want, s)
		}
	}
}

func TestGenerateDiscard(t *testing.T) {
	lock := "          - handle: LockDoor\n            raw: {const: LOCK}"
	out, err := GenerateFiles([]rowfile.File{parse(t, strings.Replace(base, lock, "          - handle: LockDoor\n            discard: true", 1))})
	if err != nil {
		t.Fatal(err)
	}
	if s := string(out); !strings.Contains(s, "b.discard(ci, ") || !strings.Contains(s, "Discard: true") {
		t.Fatalf("no discard handler:\n%s", s)
	}
	twice := strings.Replace(base, lock, lock+"\n          - handle: LockDoor\n            discard: true", 1)
	if err := generate(t, twice); err == nil {
		t.Error("a discard beside another command of the handler generated")
	}
}

func TestGenerateRefuse(t *testing.T) {
	lock := "          - handle: LockDoor\n            raw: {const: LOCK}"
	refuse := lock + "\n          - handle: LockDoor\n            if: {state: LockState, in: [LockStateLocked]}\n            refuse: FAILURE"
	out, err := GenerateFiles([]rowfile.File{parse(t, strings.Replace(base, lock, refuse, 1))})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"b.refuse(ci, 1)", "Refuse: contract.StatusFailure", "\"thingfabric.dev/go-fabric/contract\""} {
		if !strings.Contains(s, want) {
			t.Errorf("no %s:\n%s", want, s)
		}
	}
	// A handler of refusals alone reads no field for a payload.
	only := strings.Replace(base, lock, "          - handle: LockDoor\n            refuse: FAILURE", 1)
	out, err = GenerateFiles([]rowfile.File{parse(t, only)})
	if err != nil {
		t.Fatal(err)
	}
	s = string(out)
	_, body, ok := strings.Cut(s, "d.HandleLockDoor = func(")
	body, _, _ = strings.Cut(body, "return resp, err")
	if !ok || !strings.Contains(body, "b.refuse(ci, 0)") || strings.Contains(body, "f := ") {
		t.Fatalf("refusal-only handler:\n%s", s)
	}
}

func TestGenerateWritesAndDoc(t *testing.T) {
	change := "          - change: [LockState]\n"
	src := strings.Replace(base, change, change+"            writes: true\n            doc: \"locks or unlocks the door, from a command or a write\"\n", 1)
	out, err := GenerateFiles([]rowfile.File{parse(t, src)})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"if changedOrWritten(c, doorlockdef.AttrLockState)", "// MethodM locks or unlocks the door, from a command or a write\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("no %q:\n%s", want, s)
		}
	}
	twice := strings.Replace(src, "          - handle: LockDoor\n            raw: {const: LOCK}", "          - handle: LockDoor\n            method: m\n            doc: other\n            raw: {const: LOCK}", 1)
	if err := generate(t, twice); err == nil || !strings.Contains(err.Error(), "two docs") {
		t.Errorf("two docs for one method: %v", err)
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
	if methodsSource(nil, nil) != "" {
		t.Error("no rows: want no constant block")
	}
	if _, err := SpliceBetween([]byte("no markers"), KeysBegin, KeysEnd, ""); err == nil {
		t.Error("a page without the key markers spliced")
	}
}
