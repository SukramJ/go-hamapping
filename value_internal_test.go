// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"
)

// TestCut: cut keeps at most n bytes and never splits a character.
func TestCut(t *testing.T) {
	for _, c := range []struct {
		s    string
		n    int
		want string
	}{
		{"abc", 5, "abc"},
		{"abc", 0, ""},
		{"x" + strings.Repeat("ä", 40), 64, "x" + strings.Repeat("ä", 31)},
		{strings.Repeat("ä", 40), 64, strings.Repeat("ä", 32)},
		{strings.Repeat("v", 31) + "äyyy", 32, strings.Repeat("v", 31)},
		{"€uro", 2, ""},
	} {
		if got := cut(c.s, c.n); got != c.want || !utf8.ValidString(got) {
			t.Errorf("cut(%q, %d) = %q; want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestConv(t *testing.T) {
	if conv[int16](1e9) != math.MaxInt16 || conv[int16](-1e9) != math.MinInt16 || conv[int16](-2.5) != -3 {
		t.Fatal("int16")
	}
	if conv[uint8](-4) != 0 || conv[uint8](300) != 255 || conv[uint8](127.5) != 128 {
		t.Fatal("uint8")
	}
	if conv[int64](1e30) != math.MaxInt64 || conv[uint64](1e30) != math.MaxUint64 {
		t.Fatal("64 bits")
	}
}

func TestToFloatAndText(t *testing.T) {
	u := uint16(7)
	var nilp *uint16
	for v, want := range map[any]float64{json.Number("2.5"): 2.5, " 3 ": 3, int8(-1): -1, uint32(4): 4, float32(1.5): 1.5, &u: 7} {
		if got, ok := toFloat(v); !ok || got != want {
			t.Errorf("toFloat(%v) = %v %v", v, got, ok)
		}
	}
	for _, v := range []any{nil, "x", "NaN", json.Number("y"), nilp, struct{}{}, math.Inf(1)} {
		if _, ok := toFloat(v); ok && v != true {
			t.Errorf("toFloat(%v) ok", v)
		}
	}
	for v, want := range map[any]string{"ON": "on", "true": "on", "False": "off", true: "on", false: "off", 2.5: "2.5"} {
		if got, _ := toText(v); got != want {
			t.Errorf("toText(%v) = %q", v, got)
		}
	}
	if _, ok := toText(nil); ok {
		t.Error("toText(nil)")
	}
	if _, ok := toText(struct{}{}); ok {
		t.Error("toText(struct)")
	}
	if n, _ := toInt(false); n != 0 {
		t.Error("toInt(false)")
	}
	if format(true) != "true" || format(3) != "3" || format(1.5) != "1.5" {
		t.Error("format")
	}
}

func ent(desc model.Description) model.Entity {
	return &model.Basic{EntityKey: "e", EntityPlatform: hacatalog.PlatformLight, Description: desc}
}

func TestConvert(t *testing.T) {
	e := ent(model.Description{Unit: "°F", Extra: map[string]any{"brightness_scale": 0}})
	if y, ok := convert(e, &Value{Units: map[string]Linear{"°F": {Offset: -32, Scale: 5.0 / 9}}}, 212); !ok || math.Abs(y-100) > 1e-9 {
		t.Fatalf("°F %v", y)
	}
	if _, ok := convert(e, &Value{Units: map[string]Linear{"°C": {}}}, 1); ok {
		t.Fatal("a unit the table lacks converted")
	}
	if y, _ := convert(e, &Value{Func: "pow10"}, 2); y != 100 {
		t.Fatalf("pow10 %v", y)
	}
	if _, ok := convert(e, &Value{Func: "reciprocal"}, 0); ok {
		t.Fatal("1/0")
	}
	if _, ok := convert(e, &Value{Func: "cbrt"}, 1); ok {
		t.Fatal("unknown func")
	}
	if _, ok := convert(e, &Value{Func: "log10"}, 0); ok {
		t.Fatal("log10(0) unclamped")
	}
	if _, ok := convert(e, &Value{Per: "extra.brightness_scale"}, 1); ok {
		t.Fatal("divided by zero")
	}
	if _, ok := convert(e, &Value{Times: "extra.nope"}, 1); ok {
		t.Fatal("times an absent value")
	}
	if _, ok := convert(e, &Value{Per: "extra.nope"}, 1); ok {
		t.Fatal("per an absent value")
	}
	if round(1.25, 1) != 1.3 {
		t.Fatal("round")
	}
}

func TestLookups(t *testing.T) {
	st := model.State{Value: map[string]any{"color": map[string]any{"h": 1.0}}, Extra: map[string]any{"x": 2}}
	if v, ok := lookupState(st, "state.color.h"); !ok || v != 1.0 {
		t.Fatal("dotted")
	}
	if v, ok := lookupState(st, "state.x"); !ok || v != 2 {
		t.Fatal("extra")
	}
	if _, ok := lookupState(st, "state.color.h.z"); ok {
		t.Fatal("through a leaf")
	}
	if _, ok := lookupState(st, "nope"); ok {
		t.Fatal("unknown source")
	}
	if _, ok := lookupState(st, "state"); ok {
		t.Fatal("a map without state")
	}
	lo, hi := 1.0, 2.0
	e := ent(model.Description{DeviceClass: "x", Unit: "u", Min: &lo, Max: &hi})
	for from, want := range map[string]any{"device_class": "x", "unit": "u", "desc.min": 1.0, "desc.max": 2.0} {
		if v, ok := describe(e, from); !ok || v != want {
			t.Errorf("describe %s = %v", from, v)
		}
	}
	for _, from := range []string{"desc.step", "nope", "extra.nope"} {
		if _, ok := describe(e, from); ok {
			t.Errorf("describe %s found", from)
		}
	}
	if v, ok := catalogDefault("light", "brightness_scale"); !ok || v == nil {
		t.Fatal("the catalog's brightness_scale")
	}
	if _, ok := catalogDefault("toaster", "x"); ok {
		t.Fatal("a platform the catalog lacks")
	}
	if list(42) != nil || len(list([]any{"a", 1})) != 1 || len(list("a")) != 1 {
		t.Fatal("list")
	}
	obj := map[string]any{"a": 1}
	setPath(obj, "a.b", 2)
	if obj["a"].(map[string]any)["b"] != 2 {
		t.Fatal("setPath over a leaf")
	}
}

func TestFeatures(t *testing.T) {
	e := &model.Basic{
		EntityKey: "c", EntityPlatform: hacatalog.PlatformCover,
		Description: model.Description{Extra: map[string]any{"supported_features": 4 | 128}},
	}
	got := features(e)
	if len(got) != 2 || got[0] != "SET_POSITION" || got[1] != "SET_TILT_POSITION" {
		t.Fatalf("features %v", got)
	}
	e.Description.Extra["supported_features"] = "nope"
	if f := features(e); len(f) != 1 || f[0] != "NOPE" {
		t.Fatalf("names %v", f)
	}
	e.Description.Extra["supported_features"] = struct{}{}
	if features(e) != nil {
		t.Fatal("a struct")
	}
	yes := true
	if matches(e, &When{Options: &yes}) || matches(e, &When{Unit: []string{"x"}}) || matches(e, &When{StateClass: []string{"x"}}) {
		t.Fatal("matched")
	}
	if matches(e, &When{Extra: map[string]Set{"tilt_optimistic": {True: &yes}}}) {
		t.Fatal("a false key is true")
	}
}

func TestPlanLifecycle(t *testing.T) {
	light := &model.Basic{EntityKey: "l", EntityPlatform: hacatalog.PlatformLight}
	p, v := Map(&model.Device{Name: model.L("Hall")}, []model.Entity{light, nil})
	if len(v) != 1 || len(p.Endpoints) != 1 {
		t.Fatalf("%+v", v)
	}
	if got := p.Endpoints[0].Options.Name; got != "Hall" {
		t.Fatalf("name %q", got)
	}
	if got := p.Apply(context.Background(), "l", model.State{}); len(got) != 1 || !errors.Is(got[0].Err, ErrNotBuilt) {
		t.Fatalf("apply before build: %+v", got)
	}
	if p.Apply(context.Background(), "nope", model.State{}) != nil || p.DeviceFor("nope") != nil {
		t.Fatal("an unknown key")
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Build(context.Background()); err == nil {
		t.Fatal("built twice")
	}
	p.OnChange(nil)
	p.deliver(model.Command{})
	p.Close()
	for _, c := range []struct{ dev, ent, want string }{
		{"", "", "l"}, {"Hall", "Lamp", "Hall (Lamp)"}, {"Hall", "Hall lamp", "Hall lamp"}, {"", "Lamp", "Lamp"},
	} {
		light.Description.Name = model.L(c.ent)
		p := &Plan{Device: &model.Device{Name: model.L(c.dev)}}
		if got := p.options(light).Name; got != c.want {
			t.Errorf("%q + %q = %q, want %q", c.dev, c.ent, got, c.want)
		}
	}
}
