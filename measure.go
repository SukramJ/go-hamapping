// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"sync"

	"thingfabric.dev/go-fabric/contract"
)

// measure is the measurement source a riding row declares on its primary
// (device.Measurement): the kit serves it as its class's cluster, the
// battery's as PowerSource with BatPercentRemaining from the percentage.
// It is the form go-fabric's cookbook gives for a battery beside a
// measurement (docs/guides/embed-in-a-daemon.md section 3, v0.11.0).
type measure struct {
	class contract.MeasurementClass

	mu       sync.Mutex
	value    float64
	observed bool
	next     int
	notify   map[int]func()
}

var (
	_ contract.MeasurementSource = (*measure)(nil)
	_ contract.ChangeNotifier    = (*measure)(nil)
)

func (m *measure) MatterMeasurementClass() contract.MeasurementClass { return m.class }

func (m *measure) MatterFloatValue() (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.value, m.observed
}

func (m *measure) OnMatterValueChanged(cb func()) (unsubscribe func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.notify == nil {
		m.notify = map[int]func(){}
	}
	id := m.next
	m.next++
	m.notify[id] = cb
	var once sync.Once
	return func() {
		once.Do(func() {
			m.mu.Lock()
			delete(m.notify, id)
			m.mu.Unlock()
		})
	}
}

// set stores a value (observed false: unknown) and notifies a change.
func (m *measure) set(value float64, observed bool) bool {
	m.mu.Lock()
	if m.observed == observed && m.value == value {
		m.mu.Unlock()
		return false
	}
	m.value, m.observed = value, observed
	cbs := make([]func(), 0, len(m.notify))
	for _, cb := range m.notify {
		cbs = append(cbs, cb)
	}
	m.mu.Unlock()
	for _, cb := range cbs {
		cb()
	}
	return true
}
