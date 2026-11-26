// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"sukramj.dev/go-hamqtt/model"

	"sukramj.dev/go-hamapping"
)

// TestBuildError: a refused endpoint is a *BuildError naming the entity,
// its row and the entities riding on it, with today's text, unwrapping to
// go-fabric's error (HR-P2-data-35).
func TestBuildError(t *testing.T) {
	p, _ := hamapping.Map(testDevice, []model.Entity{sensor("bat", "battery", "%"), sensor("t", "temperature", "°C")})
	defer p.Close()
	ep := &p.Endpoints[0]
	ep.Declarations = append(ep.Declarations, ep.Declarations...) // a cluster declared twice: go-fabric refuses
	_, err := p.Build(context.Background())
	var be *hamapping.BuildError
	if !errors.As(err, &be) {
		t.Fatalf("Build: %v (%T), want a *BuildError", err, err)
	}
	if be.Key != "t" || be.Row != ep.Row || be.Part != "" || !slices.Equal(be.Riders, []string{"bat"}) || be.Err == nil {
		t.Fatalf("BuildError %+v", be)
	}
	if want := "hamapping: entity t (row " + ep.Row + "): " + be.Err.Error(); err.Error() != want {
		t.Fatalf("Error() %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, be.Err) {
		t.Fatal("errors.Is does not reach the cause")
	}
	be.Part = "Some Part"
	if want := "hamapping: entity t (row " + ep.Row + "): part Some Part: " + be.Err.Error(); be.Error() != want {
		t.Fatalf("Error() with a part %q, want %q", be.Error(), want)
	}
}

// TestBuildErrorOwner: the owner of a refusal is measured by building the
// endpoint again — a rider whose declaration go-fabric refuses (a second
// battery on one endpoint) is the owner; a refused primary is Key; a
// refusal no rider isolates is Key.
func TestBuildErrorOwner(t *testing.T) {
	build := func(t *testing.T, p *hamapping.Plan) *hamapping.BuildError {
		t.Helper()
		defer p.Close()
		_, err := p.Build(context.Background())
		var be *hamapping.BuildError
		if !errors.As(err, &be) {
			t.Fatalf("Build: %v (%T), want a *BuildError", err, err)
		}
		if want := "hamapping: entity " + be.Key + " (row " + be.Row + "): " + be.Err.Error(); err.Error() != want {
			t.Errorf("Error() %q, want %q", err.Error(), want)
		}
		return be
	}
	entities := func() []model.Entity {
		return []model.Entity{sensor("bat2", "battery", "%"), sensor("t", "temperature", "°C"), sensor("bat1", "battery", "%")}
	}
	t.Run("rider", func(t *testing.T) {
		p, _ := hamapping.Map(testDevice, entities())
		be := build(t, p)
		if be.Key != "t" || be.Owner != "bat2" || !slices.Equal(be.Riders, []string{"bat1", "bat2"}) {
			t.Fatalf("BuildError %+v; want Key t, Owner bat2 (the second battery)", be)
		}
		if !strings.Contains(be.Err.Error(), "second battery") {
			t.Fatalf("Err %v; want go-fabric's refusal of a second battery", be.Err)
		}
	})
	t.Run("primary", func(t *testing.T) {
		p, _ := hamapping.Map(testDevice, entities())
		p.Endpoints[0].Type = nil // go-fabric refuses the endpoint whatever rides on it
		if be := build(t, p); be.Owner != "t" {
			t.Fatalf("BuildError %+v; want Owner t", be)
		}
	})
	t.Run("not isolated", func(t *testing.T) {
		p, _ := hamapping.Map(testDevice, []model.Entity{sensor("bat", "battery", "%"), sensor("t", "temperature", "°C")})
		ep := &p.Endpoints[0]
		ep.Declarations = append(ep.Declarations, ep.Declarations...) // refused only as a whole
		if be := build(t, p); be.Owner != "t" {
			t.Fatalf("BuildError %+v; want Owner t", be)
		}
	})
	t.Run("no riders", func(t *testing.T) {
		p, _ := hamapping.Map(testDevice, []model.Entity{sensor("t", "temperature", "°C")})
		ep := &p.Endpoints[0]
		ep.Declarations = append(ep.Declarations, ep.Declarations...)
		if be := build(t, p); be.Owner != "t" || len(be.Riders) != 0 {
			t.Fatalf("BuildError %+v; want Owner t", be)
		}
	})
}
