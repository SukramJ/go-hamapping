// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
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
