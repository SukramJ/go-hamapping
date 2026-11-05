// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/SukramJ/go-fabric/cluster/spec"
	bdbidef "github.com/SukramJ/go-fabric/cluster/spec/bridgeddevicebasicinformation"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/fabrictest"
	"github.com/SukramJ/go-fabric/node"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

// entity is a go-hamqtt entity of platform p with a command binding.
func entity(key string, p hacatalog.Platform, desc model.Description) model.Entity {
	return &model.Basic{
		EntityKey: key, EntityPlatform: p, Description: desc,
		Binds: []model.Binding{
			{Role: model.RoleState, Mode: model.Read, Slot: model.S("dev1", key, model.BucketValues, "state")},
			{Role: model.RoleCommand, Mode: model.Write, Slot: model.S("dev1", key, model.BucketValues, "set")},
		},
	}
}

var testDevice = &model.Device{
	Identity: model.Identity{IDs: []model.Identifier{{Namespace: "test", Value: "dev1"}}},
	Name:     model.L("Test device"), Manufacturer: "Acme",
}

// rig is one plan built and served by a fabrictest node, commissioned by
// the site's controller.
type rig struct {
	t      *testing.T
	ctx    context.Context
	site   *fabrictest.Site
	pair   *fabrictest.Pair
	plan   *hamapping.Plan
	devs   []*device.Device
	cmds   chan model.Command
	verdct []hamapping.Verdict
}

// serve maps the entities, builds the plan and commissions a node serving
// it. Every built device must pass go-fabric's device-type validator.
func serve(t *testing.T, entities ...model.Entity) *rig {
	t.Helper()
	plan, verdicts := hamapping.Map(testDevice, entities)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	devs, err := plan.Build(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r := &rig{t: t, ctx: ctx, plan: plan, devs: devs, cmds: make(chan model.Command, 16), verdct: verdicts}
	plan.OnChange(func(c model.Command) { r.cmds <- c })
	eps := make([]node.Endpoint, len(devs))
	for i, d := range devs {
		eps[i] = d
	}
	r.site = fabrictest.NewSite(t, fabrictest.Options{})
	r.pair = r.site.CommissionedPair(fabrictest.PairOptions{Device: fabrictest.Topology{Bridged: eps}})
	if v := r.pair.Device.Node().Bridge().DeviceTypeViolations(); len(v) != 0 {
		t.Fatalf("device-type violations: %+v", v)
	}
	t.Cleanup(plan.Close)
	return r
}

// ep is the endpoint number of the entity's device.
func (r *rig) ep(key string) uint16 {
	r.t.Helper()
	d := r.plan.DeviceFor(key)
	if d == nil {
		r.t.Fatalf("no device for %s", key)
	}
	n, ok := d.EndpointNumber()
	if !ok {
		r.t.Fatalf("%s has no endpoint number", key)
	}
	return n
}

// apply applies a state and fails on a refused Set.
func (r *rig) apply(key string, st model.State) {
	r.t.Helper()
	for _, p := range r.plan.Apply(r.ctx, key, st) {
		if p.Err != nil {
			r.t.Fatalf("apply %s %s: %v", key, p.Cluster, p.Err)
		}
	}
}

// values reads one cluster of the entity's endpoint, for the cluster's
// generated StateFrom.
func (r *rig) values(key string, cluster uint32) spec.AttributeValues {
	r.t.Helper()
	values, err := fabrictest.ReadCluster(r.ctx, r.pair, r.ep(key), cluster)
	if err != nil {
		r.t.Fatal(err)
	}
	return values
}

// command waits for the next command the plan hands out.
func (r *rig) command() model.Command {
	r.t.Helper()
	var got model.Command
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()
	err := r.site.Eventually(ctx, "a command", func() bool {
		select {
		case got = <-r.cmds:
			return true
		default:
			return false
		}
	})
	if err != nil {
		r.t.Fatalf("no command: %v", err)
	}
	return got
}

// noMore fails when the plan hands out another command once the site is
// quiet: one controller command is one model.Command.
func (r *rig) noMore() {
	r.t.Helper()
	if err := r.site.Settle(r.ctx); err != nil {
		r.t.Fatal(err)
	}
	select {
	case c := <-r.cmds:
		r.t.Fatalf("an extra command: %s %s", c.Method, c.Payload)
	default:
	}
}

// expect checks a command's method, role and payload.
func expect(t *testing.T, got model.Command, method, payload string) {
	t.Helper()
	if got.Method != method {
		t.Errorf("method %q, want %q", got.Method, method)
	}
	if method == "" && got.Role != model.RoleCommand {
		t.Errorf("role %q, want %q", got.Role, model.RoleCommand)
	}
	if method == "" && got.Slot.Leaf() != "set" {
		t.Errorf("slot %+v, want the command binding's", got.Slot)
	}
	if !samePayload(got.Payload, []byte(payload)) {
		t.Errorf("payload %s, want %s", got.Payload, payload)
	}
}

func samePayload(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil {
		return reflect.DeepEqual(x, y)
	}
	return bytes.Equal(a, b)
}

// reachable reads the endpoint's Reachable.
func (r *rig) reachable(key string) bool {
	st, _ := bdbidef.StateFrom(r.values(key, bdbidef.ClusterID))
	return st.Reachable
}
