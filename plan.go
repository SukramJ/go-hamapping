// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/SukramJ/go-fabric/contract"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-hamqtt/model"
)

// ErrNotBuilt is the error of a [Patch] applied before [Plan.Build].
var ErrNotBuilt = errors.New("hamapping: plan not built")

// RideOnPrimary is [Row].Ride for a row whose clusters ride on the
// device's primary endpoint.
const RideOnPrimary = "primary"

// Verdict is the verdict on one entity: eligibility's verdict (Mappable,
// Partial or Unmappable, with a reason an operator can read, and for a
// mapped entity its device type and clusters), the entity's key and the
// row that decided.
type Verdict struct {
	Key string
	Row string
	eligibility.Verdict
}

// Endpoint is one endpoint of a [Plan]: the entity it serves, the device
// type and the declarations [Plan.Build] constructs it from.
type Endpoint struct {
	// Key is the entity's key.
	Key string
	// Row is the row that decided.
	Row string
	// Type is the device type.
	Type *device.Type
	// Options are the device's options: its stable key, name, vendor and
	// addresses.
	Options device.Options
	// Declarations are the cluster declarations of the row, and of the
	// rows that ride on this endpoint.
	Declarations []device.Declaration
	// Features are the entity's feature flag names (the catalog's).
	Features []string
	// Bindings are the entity's bindings.
	Bindings []model.Binding
	// Riders are the keys of the entities whose clusters ride on this
	// endpoint (a battery sensor's PowerSource).
	Riders []string
}

// Patch is one typed change [Plan.Apply] made: the <cluster>def.Patch
// set on the cluster's server, or the reachability set.
type Patch struct {
	Key     string
	Cluster string
	// Value is the typed <cluster>def.Patch; nil for a reachability change.
	Value any
	// Reachable is the reachability set; nil for a cluster patch.
	Reachable *bool
	// Events are the events emitted on the cluster, by name; nil
	// otherwise.
	Events []string
	// Err is what the Set returned.
	Err error
}

// Plan is the mapping of one device's entities onto go-fabric's device
// kit: one endpoint per mapped entity. [Map] makes it; [Plan.Build]
// constructs the devices, [Plan.Apply] carries an entity's state onto
// them and [Plan.OnChange] hands a controller's command back as a
// model.Command.
type Plan struct {
	Device    *model.Device
	Endpoints []Endpoint

	opts     Options
	bindings map[string]*binding
	devices  map[string]*device.Device

	mu    sync.Mutex
	fn    func(model.Command)
	unsub []func()
	built bool
}

// Options adjust [Map]. The zero value is the default.
type Options struct {
	// StableKey returns the identity of an entity that survives a change
	// of its [model.Entity.Key]: a registry id the source keeps across a
	// rename. An endpoint's stable key (device.Options.StableKey, which
	// go-fabric turns into the endpoint number) is the device's UID, "/"
	// and this identity. Nil, or an empty result, is the entity's Key.
	// Key stays the address of everything else: the verdict, the
	// endpoint's Key and ChannelAddress, [Plan.Apply] and
	// [Plan.DeviceFor].
	//
	// go-hamqtt's model has no entity identity besides Key (model.Entity,
	// model/entity.go:27-40 at v0.38.0), so the source supplies it here.
	StableKey func(model.Entity) string

	// Name returns an entity's name as the source holds it. When verbatim
	// is true and name is not empty, name is the endpoint's NodeLabel as
	// given — without the device name in front, cut to go-fabric's 32-byte
	// limit the way device.ComposeNodeLabel cuts — because a name a user
	// typed in a UI is final. When verbatim is false, or name is empty, or
	// Name is nil, the label is composed as without the option: the
	// device name and the entity name (device.ComposeNodeLabel).
	Name func(model.Entity) (name string, verbatim bool)

	// Ports returns the host object whose interfaces the clusters a row
	// marks `ports: host` run on — go-fabric's port interfaces of those
	// clusters (cluster/spec/webrtctransportprovider.ProviderPort;
	// cluster/spec/cameraavstreammanagement.VideoStreamPort,
	// AudioStreamPort, SnapshotStreamPort, SnapshotCapturePort). nil, or
	// an object that lacks a port the cluster needs, skips the row for
	// that entity with a verdict ("<row>: the consumer serves no host
	// ports for <cluster>"). It is asked once per entity, during [Map].
	Ports func(e model.Entity) any
}

// Map plans dev's entities: every entity gets a verdict, every mapped one
// an endpoint — but a row that rides ([RideOnPrimary]) puts its clusters
// on the device's primary endpoint when there is one. The entities are
// taken in order; the primary endpoint is the first mapped entity whose
// row does not ride. opts is at most one [Options]; a later one's
// non-nil fields override an earlier one's.
func Map(dev *model.Device, entities []model.Entity, opts ...Options) (*Plan, []Verdict) {
	p := &Plan{Device: dev, bindings: map[string]*binding{}, devices: map[string]*device.Device{}}
	for _, o := range opts {
		if o.StableKey != nil {
			p.opts.StableKey = o.StableKey
		}
		if o.Name != nil {
			p.opts.Name = o.Name
		}
		if o.Ports != nil {
			p.opts.Ports = o.Ports
		}
	}
	verdicts := make([]Verdict, 0, len(entities))
	type pending struct {
		b *binding
		v int
	}
	var riders []pending
	for _, e := range entities {
		if e == nil || e.Desc() == nil {
			continue
		}
		row := find(e)
		v := verdict(e, row)
		verdicts = append(verdicts, v)
		if v.State == eligibility.StateUnmappable {
			continue
		}
		b := &binding{plan: p, row: row, entity: e, key: e.Key()}
		if row.Ride == RideOnPrimary {
			p.bindings[b.key] = b
			riders = append(riders, pending{b, len(verdicts) - 1})
			continue
		}
		ep := p.endpoint(b)
		if b.noPorts != "" {
			verdicts[len(verdicts)-1] = unservedVerdict(b)
			continue
		}
		p.bindings[b.key] = b
		p.Endpoints = append(p.Endpoints, ep)
	}
	primaries := len(p.Endpoints)
	for _, r := range riders {
		if primaries == 0 {
			ep := p.endpoint(r.b)
			if r.b.noPorts != "" {
				delete(p.bindings, r.b.key)
				verdicts[r.v] = unservedVerdict(r.b)
				continue
			}
			p.Endpoints = append(p.Endpoints, ep)
			continue
		}
		primary := &p.Endpoints[0]
		if m := r.b.row.RideMeasurement; m != nil {
			r.b.measure = &measure{class: m.Class}
			primary.Declarations = append(primary.Declarations, device.Measurement(r.b.measure))
		} else {
			decls := declarations(r.b)
			if r.b.noPorts != "" {
				delete(p.bindings, r.b.key)
				verdicts[r.v] = unservedVerdict(r.b)
				continue
			}
			primary.Declarations = append(primary.Declarations, decls...)
		}
		primary.Riders = append(primary.Riders, r.b.key)
		verdicts[r.v].DeviceType = uint16(primary.Type.ID)
	}
	return p, verdicts
}

// find returns the first row of the entity's platform that matches it,
// or nil.
func find(e model.Entity) *Row {
	platform := string(e.Platform())
	for i := range rows {
		if rows[i].Platform == platform && matches(e, &rows[i].When) {
			return &rows[i]
		}
	}
	return nil
}

// verdict is the verdict of row on e.
func verdict(e model.Entity, row *Row) Verdict {
	v := Verdict{Key: e.Key()}
	if row == nil {
		v.State = eligibility.StateUnmappable
		v.Reason = fmt.Sprintf("no row for platform %s", e.Platform())
		if dc := e.Desc().DeviceClass; dc != "" {
			v.Reason += fmt.Sprintf(" with device_class %s", dc)
		}
		return v
	}
	v.Row = row.ID
	reason := strings.ReplaceAll(row.Reason, "{device_class}", string(e.Desc().DeviceClass))
	if row.Type == nil || row.State == eligibility.StateUnmappable {
		v.State, v.Reason = eligibility.StateUnmappable, reason
		return v
	}
	if missing := unconvertible(e, row); missing != "" {
		v.State, v.Reason = eligibility.StateUnmappable, missing
		return v
	}
	if why := unmaterializable(e, row); why != "" {
		v.State, v.Reason = eligibility.StateUnmappable, why
		return v
	}
	v.State, v.Reason = row.State, reason
	var caveats []string
	if reason != "" {
		caveats = append(caveats, reason)
	}
	for i := range row.Caveats {
		if matches(e, &row.Caveats[i].When) {
			caveats = append(caveats, strings.ReplaceAll(row.Caveats[i].Reason, "{device_class}", string(e.Desc().DeviceClass)))
		}
	}
	if len(caveats) > 0 {
		v.State, v.Reason = eligibility.StatePartial, strings.Join(caveats, "; ")
	}
	v.DeviceType = uint16(row.Type.ID)
	for i := range row.Clusters {
		v.Clusters = append(v.Clusters, row.Clusters[i].ID)
	}
	return v
}

// unconvertible names a unit the row cannot convert for the entity, or
// returns "".
func unconvertible(e model.Entity, row *Row) string {
	for i := range row.Clusters {
		for j := range row.Clusters[i].Attributes {
			a := &row.Clusters[i].Attributes[j]
			if a.Value.Units == nil || (a.When != nil && !matches(e, a.When)) {
				continue
			}
			u := unitOf(e, &a.Value)
			if _, ok := a.Value.Units[u]; !ok {
				return fmt.Sprintf("unit %q of device_class %s has no conversion in row %s", u, e.Desc().DeviceClass, row.ID)
			}
		}
	}
	return ""
}

// endpoint is the endpoint of b's entity on its own device.
func (p *Plan) endpoint(b *binding) Endpoint {
	e := b.entity
	return Endpoint{
		Key: b.key, Row: b.row.ID, Type: b.row.Type,
		Options:      p.options(e),
		Declarations: declarations(b),
		Features:     features(e),
		Bindings:     slices.Clone(e.Bindings()),
	}
}

// declarations are the cluster declarations of b's row.
func declarations(b *binding) []device.Declaration {
	out := make([]device.Declaration, 0, len(b.row.Clusters))
	for ci := range b.row.Clusters {
		if d := b.row.Clusters[ci].declare; d != nil {
			out = append(out, d(b, ci))
		}
	}
	return out
}

// options are the device options of an entity's endpoint.
func (p *Plan) options(e model.Entity) device.Options {
	var devName, vendor, uid string
	if p.Device != nil {
		devName, vendor, uid = p.Device.Name.Default, p.Device.Manufacturer, p.Device.UID()
	}
	key := e.Key()
	stable := key
	if p.opts.StableKey != nil {
		if id := p.opts.StableKey(e); id != "" {
			stable = id
		}
	}
	if uid != "" {
		stable = uid + "/" + stable
	}
	name := e.Desc().Name.Default
	verbatim := false
	if p.opts.Name != nil {
		if n, v := p.opts.Name(e); v && n != "" {
			name, verbatim = n, true
		}
	}
	switch {
	case verbatim: // final as given; only the 32-byte cut below applies
	case name == "" && devName == "":
		name = key
	case name == "":
		name = devName
	case devName != "" && !strings.HasPrefix(name, devName):
		name = device.ComposeNodeLabel(devName, name)
	}
	return device.Options{
		StableKey: device.StringKey(stable), Name: device.ComposeNodeLabel(name, ""),
		Vendor: vendor, DeviceAddress: uid, ChannelAddress: key,
	}
}

// Build constructs the plan's devices, in endpoint order, and installs
// the command mapping on them. A plan builds once.
func (p *Plan) Build(_ context.Context) ([]*device.Device, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.built {
		return nil, errors.New("hamapping: plan built twice")
	}
	out := make([]*device.Device, 0, len(p.Endpoints))
	for i := range p.Endpoints {
		ep := &p.Endpoints[i]
		d, err := device.New(ep.Type, ep.Options, ep.Declarations...)
		if err != nil {
			return nil, fmt.Errorf("hamapping: entity %s (row %s): %w", ep.Key, ep.Row, err)
		}
		if err := p.parts(d, ep); err != nil {
			return nil, err
		}
		out = append(out, d)
		p.devices[ep.Key] = d
		for _, key := range append([]string{ep.Key}, ep.Riders...) {
			p.bindings[key].target = d
		}
	}
	for _, key := range p.order() {
		b := p.bindings[key]
		for ci := range b.row.Clusters {
			if w := b.row.Clusters[ci].watch; w != nil {
				p.unsub = append(p.unsub, w(b, ci))
			}
		}
	}
	p.built = true
	return out, nil
}

// parts builds the component devices the endpoint's row requires and
// adds them to d.
func (p *Plan) parts(d *device.Device, ep *Endpoint) error {
	b := p.bindings[ep.Key]
	for i := range b.row.Parts {
		part := &b.row.Parts[i]
		decls := make([]device.Declaration, 0, len(part.Clusters))
		for ci := range part.Clusters {
			decls = append(decls, part.Clusters[ci].declare(b, ci))
		}
		opts := ep.Options
		opts.StableKey = device.StringKey(fmt.Sprintf("%v#%d", ep.Options.StableKey, i+1))
		opts.ChannelAddress = fmt.Sprintf("%s#%d", ep.Options.ChannelAddress, i+1)
		pd, err := device.New(part.Type, opts, decls...)
		if err == nil {
			err = d.AddPart(pd)
		}
		if err != nil {
			return fmt.Errorf("hamapping: entity %s (row %s): part %s: %w", ep.Key, ep.Row, part.Type.Name, err)
		}
	}
	return nil
}

// order is the keys of the bindings in endpoint order, riders after their
// primary.
func (p *Plan) order() []string {
	var out []string
	for i := range p.Endpoints {
		ep := &p.Endpoints[i]
		out = append(out, ep.Key)
		out = append(out, ep.Riders...)
	}
	return out
}

// DeviceFor returns the built device serving the entity key (its primary's
// for a riding entity), or nil.
func (p *Plan) DeviceFor(key string) *device.Device {
	p.mu.Lock()
	defer p.mu.Unlock()
	if b, ok := p.bindings[key]; ok {
		return b.target
	}
	return nil
}

// Apply carries the entity's state onto its device: the typed Set of
// every cluster the row maps (value, unit and scale per row) and the
// device's reachability from State.Available. It returns what it set; an
// unknown key sets nothing. A riding entity's availability does not move
// its primary's reachability.
func (p *Plan) Apply(ctx context.Context, key string, st model.State) []Patch {
	p.mu.Lock()
	b, ok := p.bindings[key]
	built := p.built
	p.mu.Unlock()
	if !ok {
		return nil
	}
	if !built || b.target == nil {
		return []Patch{{Key: key, Err: ErrNotBuilt}}
	}
	var out []Patch
	if b.row.Ride == "" || b.target == p.devices[key] {
		reachable := st.Available
		b.target.SetReachable(ctx, reachable)
		out = append(out, Patch{Key: key, Reachable: &reachable})
		if !reachable {
			b.forget()
			return out
		}
	}
	if b.measure != nil {
		m := b.row.RideMeasurement
		v, ok := b.number(&st, -1, -1)
		changed := b.measure.set(v, ok)
		if changed {
			kind, _ := contract.MeasurementKindFor(m.Class)
			out = append(out, Patch{Key: key, Cluster: kind.Name, Value: v})
		}
		return out
	}
	for ci := range b.row.Clusters {
		c := &b.row.Clusters[ci]
		if c.apply == nil {
			continue
		}
		v, err := c.apply(b, ci, st)
		if v == nil && err == nil {
			continue
		}
		out = append(out, Patch{Key: key, Cluster: c.Name, Value: v, Err: err})
	}
	for ci := range b.row.Clusters {
		c := &b.row.Clusters[ci]
		for ei := range c.Events {
			if names, err := b.event(ci, ei, &st); names != nil || err != nil {
				out = append(out, Patch{Key: key, Cluster: c.Name, Events: names, Err: err})
			}
		}
	}
	return out
}

// OnChange sets the function a controller's command or write reaches the
// daemon through, as the model.Command its Commander takes: the entity,
// the command role and slot (or the named method) and the payload in the
// entity's command form. fn runs inline, in the controller's request (the
// command handler, the write veto, the change a command made), so it must
// not block: hand the command to the daemon's own queue. nil drops the
// commands.
func (p *Plan) OnChange(fn func(model.Command)) {
	p.mu.Lock()
	p.fn = fn
	p.mu.Unlock()
}

func (p *Plan) deliver(c model.Command) {
	p.mu.Lock()
	fn := p.fn
	p.mu.Unlock()
	if fn != nil {
		fn(c)
	}
}

// Close removes the change hooks Build installed.
func (p *Plan) Close() {
	p.mu.Lock()
	unsub := p.unsub
	p.unsub = nil
	p.mu.Unlock()
	for _, u := range unsub {
		u()
	}
}

// Rows returns the table, in the order Map tries it.
func Rows() []Row { return slices.Clone(rows) }
