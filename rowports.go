// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"context"
	"fmt"
	"sync"

	doorlockdef "github.com/SukramJ/go-fabric/cluster/spec/doorlock"
)

// The row ports of a row's `ports: row` clusters: the kit's server runs on
// a port the row itself serves, so the server runs the controller's
// commands — their authorization, state, events and timers — and asks the
// port to act; the row emits the action as the handle command of that
// name. The generated declaration calls b.rowPorts<Cluster>, which sets the
// Declare's port; the generated apply calls b.rowPorts<Cluster>Apply with
// the patch of a state, which the port answers from.

// lockSource is DoorLock's row port (go-fabric cluster/lock StateSource,
// doorlockdef.Declare.StateSource at v0.18.0): the server reads LockState
// from it at every read and stores none (cluster/lock/doorlock_server.go:644-662),
// and LockInvoke moves the lock — a controller's LockDoor or UnlockDoor
// once admitted, UnlockWithTimeout as UnlockDoor, and the auto-relock when
// the timeout is up as LockDoor (:131-132, :826-849, :956-969; matter.js
// DoorLockServer.ts:203-205,956-974 at 862ad75f). The state is the
// entity's, applied; a command changes nothing until Home Assistant says
// so.
type lockSource struct {
	b  *binding
	ci int

	mu    sync.Mutex
	state *doorlockdef.LockStateEnum
}

// rowPortsDoorLock sets DoorLock's StateSource to the row's lock port.
func (b *binding) rowPortsDoorLock(d *doorlockdef.Declare, ci int) {
	if b.ports == nil {
		b.ports = map[int]any{}
	}
	src := &lockSource{b: b, ci: ci}
	b.ports[ci] = src
	d.StateSource = src
}

// rowPortsDoorLockApply takes the LockState of an applied state and
// reports the change: the attribute is served from the port, so no Set
// sees it move (doorlockdef Wrap.MarkChanged).
func (b *binding) rowPortsDoorLockApply(ci int, p doorlockdef.Patch) error {
	src, _ := b.ports[ci].(*lockSource)
	if src == nil {
		return fmt.Errorf("hamapping: %s: DoorLock has no row port", b.key)
	}
	src.mu.Lock()
	src.state = p.LockState
	src.mu.Unlock()
	return doorlockdef.Wrap(b.server(doorlockdef.ClusterID)).MarkChanged(doorlockdef.AttrLockState)
}

// IsJammed implements doorlockdef.StateSource: the entity's jammed state is
// NotFullyLocked.
func (s *lockSource) IsJammed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state != nil && *s.state == doorlockdef.LockStateNotFullyLocked
}

// IsLocked implements doorlockdef.StateSource: observed false while the
// entity's state is not known (a null LockState); Unlatched is not locked.
func (s *lockSource) IsLocked() (locked, observed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return false, false
	}
	return *s.state == doorlockdef.LockStateLocked, true
}

// LockInvoke implements doorlockdef.StateSource: the row's handle command
// named as the lock command is emitted with no fields; a command the row
// does not name fails the operation (the server answers FAILURE and emits
// LockOperationError).
func (s *lockSource) LockInvoke(_ context.Context, cmdID uint32) error {
	var name string
	switch cmdID {
	case doorlockdef.CmdLockDoor:
		name = "LockDoor"
	case doorlockdef.CmdUnlockDoor:
		name = "UnlockDoor"
	case doorlockdef.CmdUnboltDoor:
		name = "UnboltDoor"
	default:
		return fmt.Errorf("hamapping: %s: lock command %#x is no actuation", s.b.key, cmdID)
	}
	cmds := s.b.row.Clusters[s.ci].Commands
	for k := range cmds {
		if cmds[k].Kind == "handle" && len(cmds[k].On) == 1 && cmds[k].On[0] == name {
			s.b.emit(s.ci, k, func(string) (any, bool) { return nil, false })
			return nil
		}
	}
	return fmt.Errorf("hamapping: %s: the row has no %s", s.b.key, name)
}
