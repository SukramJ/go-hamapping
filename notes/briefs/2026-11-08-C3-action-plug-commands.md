# Brief C3 (go-hamapping) — every command of an action plug is bound: the on-family activates, Off is a visible discard (v0.4.6)

Owner decision (2026-11-08, go-ha2fabric #51): a scene, script, automation or
button plug answers Off, Toggle, OffWithEffect, OnWithRecallGlobalScene and
OnWithTimedOff with Success and the host never hears of them — go-fabric's own
OnOff server runs them (`cluster/onoff/server.go:748-752` at v0.18.0: an
unregistered command falls to `s.srv.MatterInvoke`). Toggle on OnOff=false runs
the server's `on`, which stores OnOff true without a service call. One pull
request on `fix/action-plug-commands`, squash-merged, auto-merge; release v0.4.6.
Host follow-up: go-ha2fabric (`bridge.command_discarded`, 0.5.7).

Sources, verified 2026-11-08 (cite them, do not re-derive):
- go-fabric v0.18.0 `cluster/onoff/server.go:748-767` (MatterInvoke: a
  registered handler replaces the server's handling for that command only;
  Handle refuses a command the selection does not accept), `:402-456` (on sets
  OnOff true and commits; toggle calls on from off),
  `cluster/spec/onoff/definition_gen.go:312-350` (Off M, On and Toggle
  !OFFONLY, OffWithEffect, OnWithRecallGlobalScene, OnWithTimedOff LT):
  AcceptedCommandList is the definition's; a row cannot trim it.
- home-assistant-matter-hub 664575a `packages/backend/src/matter/behaviors/on-off-server.ts:121-125,168-172`
  (`turnOn`/`turnOff: null`: the command only schedules an auto-reset, no
  action, no log), `:275-299` (PlugOnOffServerBase: OnWithTimedOff turns on
  from off, OnWithRecallGlobalScene turns on unless GlobalSceneControl; matter.js
  Toggle from off is on()). Scenes, scripts, automations, buttons set
  `turnOff: null` (`legacy/scene/index.ts:13-19`).
- This repo v0.4.5: `rows/{button,scene,script,automation}.yaml` (one
  `handle: "On"`), `rows/switch.yaml` (`change: [OnOff]`: the server runs the
  commands and the change hook catches them — the other shape),
  `internal/gen/gen.go:1103-1178` (handlers), `internal/rowfile/rowfile.go:213-231,531-600`
  (Command, check), `binding.go:232-251` (emit), `plan.go:82-94,488-506`
  (Plan, OnChange, deliver), `docs/rows.md:103-122` (Commands).

## §1 Row format: `discard`

`rowfile.Command.Discard bool` (`yaml:"discard"`), with `handle` only: no
`method`, `raw`, `json`, `derive`, `if`, `when`, `otherwise` (check: "discard:
a handle's alone"). `hamapping.Command.Discard bool`. The generator emits for a
discarded handle `d.Handle<X> = func(_ context.Context, _ <Req>) (resp <Resp>,
err error) { b.discard(ci, k); return resp, err }` and `{Kind: "handle", On:
[X], Discard: true}`; a handle with both discard and other commands is a
generator error.

## §2 Plan: `OnDiscard`

```go
// Discard is a controller's command a row answers with the cluster's
// success and hands to no one: the entity has no action for it.
type Discard struct { Entity model.Entity; Cluster string; Command string }
func (p *Plan) OnDiscard(fn func(Discard))
```
`binding.discard(ci, k)` delivers `Discard{b.entity, cluster.Name, cmd.On[0]}`
inline, as `deliver`. nil drops them (the default: as today, but the row states
it).

## §3 Rows

`button`, `scene`, `script`, `automation` (`rows/*.yaml`), OnOff commands:
`On`, `Toggle`, `OnWithTimedOff`, `OnWithRecallGlobalScene` each `handle` with
the row's method and payload (press/PRESS, turn_on/ACTIVATE, turn_on/RUN,
trigger/TRIGGER); `Off` and `OffWithEffect` `discard: true`. Cites add
matter-hub `on-off-server.ts:121-125,168-172,275-299`. Reason in the file's
head comment: a momentary plug is always off, so every on-family command
activates and off has nothing to do.

## §3a The other handle-only clusters the guard finds

A probe over the suite (the served AcceptedCommandList against the handled
names, 2026-11-08) finds three more gaps of the same shape; the guard holds
them, so they close here:
- `climate/air_conditioner` OnOff and `media_player/speaker`,
  `media_player/video_player` OnOff: `Toggle` unbound — go-fabric's toggle
  stores the flipped OnOff, no service call. Bind `handle: Toggle` to
  `method: toggle`, `raw: {const: TOGGLE}` (speaker under the `*power` when):
  `climate.toggle` (`climate/services.yaml:145-151` at 2026.10.3, TURN_ON and
  TURN_OFF) and `media_player.toggle` (`media_player/services.yaml:17-24`,
  TURN_ON and TURN_OFF); both rows already require the features. A new method
  constant `MethodToggle`.
- `media_player/speaker_mute` OnOff: `Toggle` `discard: true` — Home Assistant
  has no service that toggles mute (`volume_mute` takes `is_volume_muted`), and
  a handler reads no state.
- `lock/door_lock` DoorLock: `UnlockWithTimeout` unbound — go-fabric's server
  flips LockState and relocks the attribute, Home Assistant is never called.
  Bind `handle: UnlockWithTimeout` to the unlock payload (`raw: {from:
  extra.payload_unlock, catalog: true}`), as matter-hub's unlock
  (`legacy/lock/lock-unlock-with-timeout-bringup.test.ts:21-23,126-141` at
  664575a: "UnlockWithTimeout used to be left to matter.js, which flipped
  lockState"; it unlocks through HA and relocks through HA when the timeout is
  up). The relock after the timeout is left open — a go-hamapping issue, since
  a row has no timer and the question is whether go-fabric's DoorLock server
  should run it with the row's unlock and lock as its sink.
Clusters the probe found complete: cover WindowCovering, vacuum
RvcOperationalState and RvcRunMode, valve, video_player MediaPlayback and
KeypadInput.

## §4 Guard

`table_test.go` `TestHandledClustersBindEveryCommand`, through the rig: `serve`
checks every built endpoint's clusters that have `handle` commands and no
`change` command — the served `AcceptedCommandList` (`spec.AttrAcceptedCommandList`)
names only handled or discarded commands (names from `spec.Lookup`). A cluster
with `change` is the server's; one with `handle` is the entity's and binds every
command the device accepts. Scope: the clusters the row tests build (§3a lists the
probe's result).

## §5 Tests

`rows_input_test.go` `actionPlug` and `TestButton`: Toggle, OnWithTimedOff
(`OnTime 0, OffWaitTime 0, OnOffControl 0`), OnWithRecallGlobalScene each
deliver the method, OnOff stays false; Off and OffWithEffect deliver a
`Discard` (rig gains `discards chan`, `OnDiscard`), no command, OnOff false;
`r.noMore()` also drains discards. `rowfile` check tests for the discard rules;
gen test for the emitted handler. `TestClimateAirConditioner`,
`TestMediaPlayerSpeaker`, `TestMediaPlayerTV`: Toggle delivers `toggle`/TOGGLE;
`TestMediaPlayerMute`: Toggle is a Discard; `TestLock`: UnlockWithTimeout
delivers UNLOCK.

## §6 Docs

`docs/rows.md` Commands: `discard: true` (example line and paragraph).
`docs/by_design.md` "always reads off": one paragraph — matter-hub's
`turnOff: null` discards silently (`on-off-server.ts:168-172`); here the
discard reaches the host (`Plan.OnDiscard`) so it can log it; the on-family
activates as matter-hub's PlugOnOffServer (`:275-299`). `changelog.md` `### Fixed`
with go-ha2fabric #51 (the plugs; Toggle on climate and media players;
UnlockWithTimeout) and `### Added` (`discard`, `Plan.OnDiscard`,
`MethodToggle`). Host: go-ha2fabric's service table gets `toggle` for climate
and media_player. `CLAUDE.md`/`README.md` state lines with the release.

## §7 Guards, commits, release, report

Commits, each green under `make ci`: (1) brief; (2) rowfile + gen + plan + rows
+ generate + tests; (3) docs + changelog; (4) release v0.4.6. Guards in the
background with one summary log: `make ci`, `make race`, `make cover-check`,
`make fuzz-smoke`. Push, pull request, auto-merge, tag. Report in the fixed
shape, under 200 words.
