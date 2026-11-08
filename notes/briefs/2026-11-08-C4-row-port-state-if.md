# Brief C4 (go-hamapping) — the lock as the DoorLock server's actuator (`ports: row`), a handle that branches on the cluster's state (`if: {state: …}`) (v0.4.7)

Owner decisions (2026-11-08, wizard): #23 — the relock after UnlockWithTimeout
runs in go-fabric's DoorLock server with the row as its actuator; #24 — Toggle
on `media_player/speaker_mute` branches on the cluster's current OnOff and calls
`volume_mute` with the inverse. One pull request on `feat/row-port-state-if`,
squash-merged, auto-merge; release v0.4.7. Host follow-up: go-ha2fabric bump
(0.5.8).

Sources, verified 2026-11-08 (cite them, do not re-derive):
- go-fabric v0.18.0 `cluster/lock/doorlock_server.go:131-145` (StateSource:
  IsJammed, IsLocked, LockInvoke; UnlockWithTimeout and an auto-relock arrive
  as UnlockDoor and LockDoor), `:826-849` (lockCommand: an unlock arms the
  auto-relock with the request's timeout), `:956-969` (LockInvoke gets
  `lockdef.CmdLockDoor`, `CmdUnlockDoor`, `CmdUnboltDoor`), `:644-662`
  (sourceAdapter: LockState is read from the source at every read),
  `:970-981` (setActuated with a source bumps the version and reports
  LockState, which the source now answers); `cluster/spec/doorlock/server_gen.go:695-712,807-808`
  (`Declare.StateSource`), `:463-470` (`Wrap(srv).MarkChanged`: an attribute
  served from a Source, so no Set sees it change); `device/defaults.go:519-538`
  (the kit hands `Declare.StateSource` to `lock.New` as `Source`).
- matter.js 862ad75f `packages/node/src/behaviors/door-lock/DoorLockServer.ts:203-205`
  (unlockWithTimeout schedules the auto-relock with the request's timeout),
  `:956-974` (the timer; a lock operation meanwhile replaces or stops it).
- home-assistant-matter-hub 664575a `legacy/lock/lock-unlock-with-timeout-bringup.test.ts:21-23,126-141`
  (unlock through HA, relock through HA when the timeout is up).
- Home Assistant 2026.10.3 `media_player/services.yaml:3-24` (no service
  toggles mute; `volume_mute` takes `is_volume_muted`).
- This repo v0.4.6: `rows/lock.yaml`, `rows/media_player.yaml` (speaker_mute),
  `ports.go` (`ports: host`), `internal/gen/gen.go:1022-1028` (port
  materializer), `:1063-1105` (cond), `:1253-1266` (apply), `:1103-1178`
  (handlers), `:1306-1310` (the change hook's `st`), `internal/rowfile/rowfile.go:114-121,249-253,495-497`.

## §1 `ports: row`

`rowfile.PortsRow = "row"`: the kit's server runs on a port the row itself
serves (`rowports.go`), beside `ports: host`. `rowPortMaterializers`
(`gen.go`): `DoorLock → DoorLock`; the generator refuses `ports: row` on any
other cluster, and a `handle` with `if`, `otherwise` or `discard` under it.
Generated: the declare calls `b.rowPortsDoorLock(&d, ci)` and registers no
handlers (the server runs the commands); the apply ends in
`b.rowPortsDoorLockApply(ci, p)` instead of `Set(p)`. `hamapping.Cluster.Ports`
carries the value; the rows table shows `[ports: row]`.

`rowports.go`: `lockSource{b, ci, mu, state *LockStateEnum}` implements
`doorlockdef.StateSource` — IsJammed: NotFullyLocked; IsLocked: Locked →
(true, true), Unlocked or Unlatched → (false, true), nil → (false, false), a
null LockState; LockInvoke: the row's `handle` whose name is the command
(LockDoor, UnlockDoor, UnboltDoor) is emitted with no fields, none is an
error (the server answers FAILURE and emits LockOperationError). Apply stores
`p.LockState` and calls `doorlockdef.Wrap(server).MarkChanged(AttrLockState)`.
The binding keeps its row ports in `ports map[int]any`.

`rows/lock.yaml`: `ports: row`; `handle: LockDoor` and `handle: UnlockDoor`
stay as the actuations; `handle: UnlockWithTimeout` goes (the server runs it
and relocks through LockInvoke(LockDoor) when the timeout is up; AutoRelockTime
the same way). Cites above.

## §2 `if: {state: <Attribute>}`

`rowfile.Cond.State`; exactly one of `field` and `state`. `gen.cond` resolves a
`state` on the cluster's state type and compares `st.<Attribute>`; a handler
with such a condition starts with `st := <alias>.Wrap(b.server(<alias>.ClusterID)).Get()`
(as the change hook does). `rows/media_player.yaml` speaker_mute: Toggle with
`if: {state: OnOff, in: ["true"]}` → `volume_mute` true, with `["false"]` →
false (the plug's OnOff is "not muted"); the `discard` goes.

## §3 Guard, tests

`bindsEveryCommand` skips a cluster with `Ports` set: the kit's server runs the
commands. `TestLock`: LockState follows the applied state through the source
(LOCKED → Locked, UNLOCKED → Unlocked, JAMMED → NotFullyLocked, an unknown
state → null); UnlockDoor emits UNLOCK and LockState stays what Home Assistant
says; UnlockWithTimeout{Timeout: 1} emits UNLOCK and then LOCK within the
rig's wait (the site's clock is the system clock); LockDoor emits LOCK.
`TestMediaPlayerMute`: Toggle while unmuted → `volume_mute` true, after the
state muted → false. rowfile: `if` with field and state refused, `ports: row`
parses; gen: `ports: row` on DoorLock generates the materializer calls and
no handlers, with an `if` under it is refused, `if: {state: …}` generates the
`st :=` read.

## §4 Docs, release

`docs/rows.md`: Ports (`row`), Commands (`if` with `state`), the example.
`docs/by_design.md`: the lock entry — the relock runs in go-fabric's server
with the row as its actuator, as matter-hub relocks through HA; the mute
Toggle. `changelog.md` 0.4.7: Added (`ports: row`, `if: {state}`), Fixed (the
relock, go-hamapping #23; the mute Toggle, #24). State lines. Guards: `make ci`,
`make race`, `make cover-check`, `make fuzz-smoke`; push, pull request,
auto-merge, tag.
