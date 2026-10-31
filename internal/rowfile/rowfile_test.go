// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package rowfile

import (
	"strings"
	"testing"
)

const good = `platform: switch
rows:
  - id: plug
    when: {device_class: [none, outlet]}
    type: OnOffPlugInUnit
    verdict: mappable
    cites: [x]
    clusters:
      - cluster: OnOff
        features: [Lighting, {name: X, when: {options: true}}]
        initial: [{attribute: A, const: 1}]
        start_null: [B]
        attributes: [{attribute: OnOff, from: state, enum: {"on": true}}]
        commands:
          - change: [OnOff]
            if: {field: OnOff, in: ["true"]}
            raw: {from: extra.payload_on, catalog: true}
          - handle: Off
            json: {state: {const: "OFF"}}
        events:
          - on: state
            emit: [{event: InitialPress, literal: {NewPosition: 1}}, {event: ShortRelease}]
        options: {list: desc.options, current: state, method: select_option}
        ports: host
  - id: rest
    when: {}
    verdict: "unmappable: no"
`

func TestParse(t *testing.T) {
	f, err := Parse([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if f.Platform != "switch" || len(f.Rows) != 2 || f.Rows[0].Clusters[0].Features[1].When == nil ||
		f.Rows[0].Clusters[0].Events[0].On != "state" || len(f.Rows[0].Clusters[0].Events[0].Emit) != 2 ||
		f.Rows[0].Clusters[0].Ports != PortsHost {
		t.Fatalf("%+v", f)
	}
}

func TestParseRefuses(t *testing.T) {
	for name, edit := range map[string][2]string{
		"unknown key":          {"verdict: mappable", "verdict: mappable\n    colour: red"},
		"verdict":              {"verdict: mappable", "verdict: maybe"},
		"reason on mappable":   {"verdict: mappable", "verdict: \"mappable: why\""},
		"partial no reason":    {"verdict: mappable", "verdict: partial"},
		"no type":              {"type: OnOffPlugInUnit", "type: \"\""},
		"no cites":             {"cites: [x]", "cites: []"},
		"ride":                 {"cites: [x]", "cites: [x]\n    ride: sideways"},
		"id":                   {"id: plug", "id: a/b"},
		"twice":                {"id: rest", "id: plug"},
		"two kinds":            {"handle: Off", "handle: Off\n            write: [OnOff]"},
		"no kind":              {"handle: Off", "method: x"},
		"raw and json":         {"json: {state: {const: \"OFF\"}}", "json: {state: {const: \"OFF\"}}\n            raw: {const: x}"},
		"func":                 {"from: state, enum", "from: state, func: sqrt, enum"},
		"units and scale":      {"from: state, enum", "from: state, scale: 2, units: {x: {}}, enum"},
		"attribute no from":    {"{attribute: OnOff, from: state,", "{attribute: OnOff,"},
		"initial two sources":  {"{attribute: A, const: 1}", "{attribute: A, const: 1, from: device_class}"},
		"initial reads state":  {"{attribute: A, const: 1}", "{attribute: A, from: state}"},
		"command two sources":  {"raw: {from: extra.payload_on, catalog: true}", "raw: {from: extra.payload_on, field: X}"},
		"enum constant":        {`enum: {"on": true}`, `enum: {"on": "a b"}`},
		"if without in":        {`if: {field: OnOff, in: ["true"]}`, `if: {field: OnOff}`},
		"if constant":          {`in: ["true"]`, `in: ["a-b"]`},
		"feature name":         {"[Lighting,", "[\"Light ing\","},
		"duplicate cluster":    {"      - cluster: OnOff\n", "      - cluster: OnOff\n      - cluster: OnOff\n"},
		"unmappable with type": {"verdict: \"unmappable: no\"", "verdict: \"unmappable: no\"\n    type: X"},
		"derive":               {"json: {state: {const: \"OFF\"}}", "json: {state: {const: \"OFF\"}}\n            derive: nope"},
		"per":                  {"from: state, enum", "from: state, per: state, enum"},
		"start_null":           {"start_null: [B]", "start_null: [\"b c\"]"},
		"event on":             {"- on: state", "- on: desc.min"},
		"event on key":         {"- on: state", "- on: \"extra.\""},
		"event no emit":        {"emit: [{event: InitialPress, literal: {NewPosition: 1}}, {event: ShortRelease}]", "emit: []"},
		"event name":           {"{event: ShortRelease}", "{event: \"Short Release\"}"},
		"otherwise not handle": {"            raw: {from: extra.payload_on, catalog: true}", "            raw: {from: extra.payload_on, catalog: true}\n            otherwise: {Status: X}"},
		"otherwise without if": {"json: {state: {const: \"OFF\"}}", "json: {state: {const: \"OFF\"}}\n            otherwise: {Status: X}"},
		"options list":         {"list: desc.options", "list: desc.min"},
		"options current":      {"current: state,", "current: device_class,"},
		"options method":       {"method: select_option", "method: \"\""},
		"event field":          {"literal: {NewPosition: 1}", "literal: {\"New Position\": 1}"},
		"ports on a part":      {"    clusters:\n      - cluster: OnOff", "    parts: [{type: X, clusters: [{cluster: OnOff, ports: host}]}]\n    clusters:\n      - cluster: OnOff"},
	} {
		t.Run(name, func(t *testing.T) {
			src := strings.Replace(good, edit[0], edit[1], 1)
			if src == good {
				t.Fatalf("edit %q does not apply", edit[0])
			}
			if _, err := Parse([]byte(src)); err == nil {
				t.Fatal("parsed")
			}
		})
	}
	// A ports value other than host names the row and the cluster.
	if _, err := Parse([]byte(strings.Replace(good, "ports: host", "ports: other", 1))); err == nil ||
		!strings.Contains(err.Error(), "row plug") || !strings.Contains(err.Error(), "cluster OnOff") || !strings.Contains(err.Error(), `ports "other"`) {
		t.Errorf("ports: other: %v", err)
	}
	for _, src := range []string{"", "rows: []", "platform: x", "platform: x\nrows: [{}]"} {
		if _, err := Parse([]byte(src)); err == nil {
			t.Errorf("%q parsed", src)
		}
	}
}

func TestLoad(t *testing.T) {
	files, err := Load("../../rows")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 13 || files[0].Name != "binary_sensor.yaml" {
		t.Fatalf("%d files, first %s", len(files), files[0].Name)
	}
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("an empty directory loaded")
	}
	if err := unique([]File{{Platform: "a", Name: "1"}, {Platform: "a", Name: "2"}}); err == nil {
		t.Fatal("a platform twice")
	}
}

// FuzzParse: the row loader refuses or accepts any bytes without a panic,
// and what it accepts checks again.
func FuzzParse(f *testing.F) {
	f.Add([]byte(good))
	f.Add([]byte("platform: x\nrows:\n  - id: a\n    verdict: \"unmappable: b\"\n"))
	f.Add([]byte("{"))
	f.Fuzz(func(t *testing.T, data []byte) {
		file, err := Parse(data)
		if err != nil {
			return
		}
		if err := file.Check(); err != nil {
			t.Fatalf("accepted, then refused: %v", err)
		}
	})
}
