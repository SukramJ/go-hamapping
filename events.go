// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"errors"
	"strings"

	"sukramj.dev/go-hamqtt/model"
)

// errNoEndpoint is the error of an event on a device the node has not
// numbered yet: go-fabric records an event on the endpoint its server is
// mounted on (<cluster>def.Server.Record<Event>), which a device has
// once it is served (device.Device.EndpointNumber).
var errNoEndpoint = errors.New("hamapping: the device has no endpoint number yet")

// Home Assistant's states that carry no value: STATE_UNKNOWN and
// STATE_UNAVAILABLE (homeassistant/const.py:303, :313 at 2026.10.3).
const (
	stateUnknown     = "unknown"
	stateUnavailable = "unavailable"
)

// event emits event ei of cluster ci when its value changed from a known
// value to another; it returns the events emitted (nil for none) and the
// first error of an emission. An unknown value clears the baseline, so
// the next known value only sets it.
func (b *binding) event(ci, ei int, st *model.State) ([]string, error) {
	c := &b.row.Clusters[ci]
	ev := &c.Events[ei]
	var raw any
	if ev.On == fromState || strings.HasPrefix(ev.On, prefixSt) {
		raw, _ = lookupState(*st, ev.On)
	} else {
		raw, _ = describe(b.entity, ev.On)
	}
	s, ok := toText(raw)
	known := ok && s != stateUnknown && s != stateUnavailable
	slot := [2]int{ci, ei}
	b.mu.Lock()
	prev, had := b.last[slot]
	switch {
	case !known:
		delete(b.last, slot)
	default:
		if b.last == nil {
			b.last = map[[2]int]string{}
		}
		b.last[slot] = s
	}
	b.mu.Unlock()
	if !known || !had || prev == s {
		return nil, nil
	}
	if _, numbered := b.target.EndpointNumber(); !numbered {
		return ev.Emit, errNoEndpoint
	}
	return ev.Emit, c.fire(b, ei)
}

// forget clears every event's baseline: the entity became unavailable.
func (b *binding) forget() {
	b.mu.Lock()
	b.last = nil
	b.mu.Unlock()
}
