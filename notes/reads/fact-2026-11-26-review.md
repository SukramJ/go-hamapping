# Fact note 2026-11-26 — review read of c93b77b..a3f50a5 (#48 #50 #43 #44 #45 #39) and the tilt refusal

Reference reader (Fable role, run on Opus). M = measured at the pin with
file:line; I = inference, labelled. Read-only except this file.

Pins:
- `gf:` go-fabric v0.27.0 (`~/go/pkg/mod/thingfabric.dev/go-fabric@v0.27.0`)
- `mjs:` matter.js local HEAD **6e751e663** (2026-11-26); the lines below are
  identical at `862ad75f` (model resource) and `db021da0` (WindowCoveringServer)
  — checked with `git show <pin>:<path>`
- `ha:` home-assistant/core 2026.10.3 (`git show 2026.10.3:<path>`)
- `mh:` home-assistant-matter-hub 664575af (RiDDiX/home-assistant-matter-hub,
  read via `gh api …/contents?ref=664575af`, sha 664575af68bd…)
- `hm:` go-hamapping-wt HEAD a3f50a5

## Part A — GoToTiltPercentage of a cover without SET_TILT_POSITION

### A1. Matter tilt direction (M)
- `mjs:packages/model/src/standard/resources/window-covering-cluster.resource.ts`:
  UpOrOpen (`:147`) sets TargetPositionTiltPercent100ths to 0.00 % (`:157`);
  DownOrClose (`:180`) to 100.00 % (`:190`). So **0 = fully open, 10000 =
  fully closed**, as for lift.
- GoToTiltPercentage (`:249`): without the PositionAware feature "a zero
  percentage shall be treated as a UpOrOpen command and a non-zero
  percentage shall be treated as an DownOrClose command" (`:265-266`); the
  lift twin is `:242-243`.
- `mjs:packages/node/src/behaviors/window-covering/WindowCoveringServer.ts:598-614`:
  with positionAwareTilt the value is the target (`:601-606`), else 0 →
  upOrOpen, any other → downOrClose (`:607-613`). Lift: `:576-592`
  (aware `:579-584`, else `:585-591`).
- `gf:cluster/cover/windowcovering_server.go`: Mover doc "percent100ths
  (0 open, 10000 closed)" `:114`; parseCommand UpOrOpen → 0, DownOrClose →
  10000 on every axis `:700-703`; GoToTiltPercentage on a non-aware tilt →
  `openOrClose` `:717-731`, `openOrClose` 0 → open else 10000 `:166-171`.
  `gf:cluster/spec/windowcovering/definition_gen.go:52` GoToTiltPercentage
  conformance "TL & PA_TL, [TL]"; payload range 0..10000 `:343`.

### A2. Home Assistant tilt direction (M)
- `ha:homeassistant/components/cover/__init__.py:174-179`
  current_cover_tilt_position: "None is unknown, 0 is closed, 100 is fully
  open" (`:177`); same scale for current_cover_position `:166-171`.
- `ha:homeassistant/components/cover/services.py:71-76` open_cover_tilt
  needs OPEN_TILT, `:78-83` close_cover_tilt CLOSE_TILT, `:92-101`
  set_cover_tilt_position SET_TILT_POSITION (tilt_position 0..100).

### A3. What the row does today (M, `hm:rows/cover.yaml`)
- `:162-165` SET_TILT_POSITION: `method: tilt`, `TiltPercent100thsValue ×
  -0.01 + 100` (0 → 100 HA open, 10000 → 0 HA closed — direction correct).
- `:166-170` OPEN_TILT|CLOSE_TILT ∧ ¬SET_TILT_POSITION, `in: [0]` →
  `open_tilt` (`OPEN`).
- `:171-175` same `when`, `not_in: [0]` → `close_tilt` (`CLOSE`): every
  non-zero value closes. Comment `:166` "Tilt without PositionAwareTilt: 0
  opens, else closes".
- Features: PositionAwareTilt only with SET_TILT_POSITION (`:43`); the tilt
  position attributes only with SET_TILT_POSITION (`:69-70`).
- Generated: `hm:rows_gen.go:1647-1663` HandleGoToTiltPercentage emits
  command 21/22/23 on the raw `req.TiltPercent100thsValue`.

### A4. matter-hub (M where cited, I where marked)
- `mh:…/legacy/cover/index.ts:52-63`: Tilt on open_tilt or
  set_tilt_position; PositionAwareTilt only with set_tilt_position (#405).
- `mhC:…/cover-window-covering-server.ts:283-297` setTiltPositionAction:
  without tilt position control, HA-scale ≥ 50 → open_cover_tilt, else
  close_cover_tilt. `:307-327` open/closeTiltAction: with open/close tilt
  the dedicated services, else set_cover_tilt_position at 0 / 100 (Matter
  scale).
- `mh:packages/backend/src/matter/behaviors/window-covering-server.ts:801-806`
  goToTiltPercentage calls `super.goToTiltPercentage` (matter.js), so on a
  non-PA_TL cover matter.js's rule (0 opens, else closes; `mjs:…:607-613`)
  runs first; handleMovement `:861-874` routes 0 / 10000 / direction to
  open/close. I: the 50 % threshold of `:283-297` is reached for tilt only
  on the lift-fallback path (`mhC:392-396`) and from open/closeTiltAction
  without open/close tilt; matter-hub's effective GoToTiltPercentage on a
  non-PA_TL cover is matter.js's 0 → open, else → close.

### A5. Verdict
Row lines for the owner decision (replace `hm:rows/cover.yaml:166-175`):
```yaml
          # Without SET_TILT_POSITION: fully open (0) opens, fully closed
          # (10000) closes, a tilt between is refused with FAILURE
          # (docs/by_design.md).
          - handle: GoToTiltPercentage
            when: {features: {any: [OPEN_TILT, CLOSE_TILT], none: [SET_TILT_POSITION]}}
            if: {field: TiltPercent100thsValue, in: [0]}
            method: open_tilt
            raw: {const: OPEN}
          - handle: GoToTiltPercentage
            when: {features: {any: [OPEN_TILT, CLOSE_TILT], none: [SET_TILT_POSITION]}}
            if: {field: TiltPercent100thsValue, in: [10000]}
            method: close_tilt
            raw: {const: CLOSE}
          - handle: GoToTiltPercentage
            when: {features: {any: [OPEN_TILT, CLOSE_TILT], none: [SET_TILT_POSITION]}}
            if: {field: TiltPercent100thsValue, not_in: [0, 10000]}
            refuse: FAILURE
```
Mechanically TiltPercent100ths without PositionAwareTilt behaves as lift
does in the row: the typed handler replaces the server's own handling
("in place of the server", `gf:cluster/spec/windowcovering/server_gen.go:391-393`;
`gf:cluster/cover/windowcovering_server.go:740` runs the host handler before
`parseCommand`/`openOrClose`), so the row sees the raw value; GoToTiltPercentage
stays in AcceptedCommandList on a non-aware tilt (`windowcovering_server.go:155-157`).

**Difference that needs its own by_design entry (M + I):** lift now always
declares PositionAwareLift, so refusing a position between is a server
choice the non-PA rule does not cover. Tilt without PA_TL *is* under the
non-PA rule, which says a non-zero percentage "shall be treated as" a
DownOrClose (`mjs:…resource.ts:265-266`), as matter.js (`:607-613`), go-fabric
(`windowcovering_server.go:166-171, :727-729`) and, effectively, matter-hub
(A4) do. The refusal therefore diverges from a spec "shall"; the by_design
entry must say so, cite those lines, and the comment `cover.yaml:166` goes.
I: the alternative that puts tilt on lift's footing is declaring
PositionAwareTilt always (with attributes from state as for lift);
matter-hub ties PA_TL to set_tilt_position (`index.ts:52-62`, #405). Owner's
call — stated, not decided here.
I: with only one of OPEN_TILT / CLOSE_TILT, 0 or 10000 hands out a method HA
refuses (ServiceNotSupported) — the same shape as the existing lift rows.

## Part B — review of the finished text

| # | Where | Verdict |
|---|---|---|
| 1 | `rows/cover.yaml` cite `matter.js …WindowCoveringServer.ts:576-590 @db021da0 (without positionAwareLift: 0 opens, else closes)` | **WRONG for the row now.** The lines are right (M, `:585-591` at db021da0) but describe a non-PA lift; every cover declares PositionAwareLift (`cover.yaml` features), so the row's lift is matter.js's aware branch `:579-584` plus the row's own refusal. Replace with: `matter.js packages/node/src/behaviors/window-covering/WindowCoveringServer.ts:579-584 @db021da0 (with positionAwareLift the requested LiftPercent100thsValue becomes the target; the row hands out only 0 and 10000 and refuses the rest, docs/by_design.md)`; and for the tilt rows `…:598-614 @db021da0 (without positionAwareTilt: TiltPercent100thsValue 0 opens, else closes — the rule the refusal departs from)` plus `matter.js packages/model/src/standard/resources/window-covering-cluster.resource.ts:265-266 @862ad75f`. |
| 2 | `rows/climate.yaml` cite `services.py:49-58,121-130 (… refused only when low is above high)` | **WRONG ("only").** `ha:climate/services.py:54-59` Inclusive pair ✓, `:121-130` low > high ✓; also refused without TARGET_TEMPERATURE_RANGE (`:105-113`) and when a value is outside min_temp..max_temp (`:149-158`). Suggested: `services.py:49-60,105-113,121-130,149-158 (… together or not at all; refused when low is above high, when either lies outside min_temp/max_temp, or without TARGET_TEMPERATURE_RANGE)`. |
| 3 | `binding.go` comment and `climate.yaml` cite `thermostat_server.go:1165-1172` | **DOUBTFUL (range).** `:1165-1172` is the switch and the `writeSetpoint` call; the OriginControllerWrite notification is `:1183-1184`, after `commitSetpoints` (`gf:cluster/thermo/setpoints.go:180-197`, which lists every setpoint that moved, the companion included). Use `:1165-1184` (+ `setpoints.go:180-197`). `:1737-1752` (RaiseLower: one `NotifyFrom(OriginCommand, …)` at `:1752`) OK. |
| 4 | `docs/rows.md` "`writes: true` … once per admitted attribute" | **DOUBTFUL.** It fires once per admitted write that *moved* a setpoint: `commitSetpoints` lists only changed values (`setpoints.go:183-187`) and an empty list notifies nobody (`gf:cluster/attribute_changes.go:83-87, :109-111`). I: a controller write of a Min/Max*SetpointLimit that moves an occupied setpoint also notifies OriginControllerWrite with it (`thermostat_server.go:1167-1184`, limits in the same case), so it also yields a temperature_range in Auto — consistent with "the pair the server holds", worth one sentence. |
| 5 | `ports.go` "a bound without the port fails the build (device/options.go:297-316)" | **DOUBTFUL (cite).** `options.go:288-315` (hasPorts) only counts a non-zero setting as a port; the refusal is `gf:device/device.go:492-496`. Cite both. In go-hamapping a bound is never set without the port (`ports.go:78-85`), so the sentence describes go-fabric, not a reachable path here. |
| 6 | `options.go` setpointBounds / `autoDeadband` comment, `docs/rows.md`, changelog #45 | OK. M: default 20 × 0.1 °C (`thermostat_server.go:135-139, 549`), ×10 with AUTO (`setpoints.go:65-67`). Predicate checked by hand against `reconcileInitial` (`setpoints.go:389-403`) → `fixLimitDeadband` (`:479-514`) → `violation` (`:315-352`) with min*=absMin*=L, max*=absMax*=H on both sides: H−L ≥ 200 repairs (maxHeat→H−200, minCool→L+200), H−L < 200 leaves minHeat > maxHeat or minHeat < absMin → error. lo > hi → `:324-325`. Default AbsMax 3000 (`thermostat_server.go:366`) ✓ for "was 30 °C". |
| 7 | HA defaults (#45) | OK. M: climate `DEFAULT_MIN_TEMP = 7`, `DEFAULT_MAX_TEMP = 35` (`climate/__init__.py:105-106`), converted from °C (`:653-669`), capability attributes `:239-254`; water_heater 110 / 140 (`water_heater/__init__.py:52-53`) from °F (`:349-365`), capability attributes `:164-172` (the cite says "state attributes": capability attributes are part of the state's attributes — acceptable). MQTT optional + kept default: `mqtt/climate.py:337-338, 498-501`, `mqtt/water_heater.py:146-147, 207-210` ✓. |
| 8 | `docs/by_design.md` "A window is a RollerShade…" | OK. `definition_gen.go:96, 245` (LF & !TL), `:120` (EndProductType Unknown, M), `:253` (TiltBlindLift, LF & TL); mhC `:57-64` (Alexa #312 at `:59-60`), `:78-102` (tilting window Unknown at `:96-99`). |
| 9 | `docs/by_design.md` "No threshold for a lift without SET_POSITION" | OK. `definition_gen.go:37, 40, 51` ✓; mhI `:34-49` ✓; `services.py:48-57` set_cover_position needs SET_POSITION ✓; mhC `:392-410` ≥ 50 % opens ✓. |
| 10 | `rows/cover.yaml` new cites (definition_gen, defaults.go:449-458, `cover/__init__.py:217-218,239-240`, mhI, mhC `57-64,78-102,330-342,392-410`, `services.py:60-62,85-90`) | OK (M at the lines named). |
| 11 | CONCEPT.md / README / changelog cover prose | OK, except that the tilt half of the #44 text will need the by_design reference of A5 once the tilt refusal lands. |
| 12 | `rows/event.yaml` `switchcluster/server_gen.go:181-193,227-239 @v0.27.0` | OK (RecordInitialPress `:185`, RecordShortRelease `:231`, EmitEvent on the mounted endpoint). |
| 13 | `rows/camera.yaml` `webrtctransportprovider/server_gen.go:314-339`; `ports.go` `defaults_camera.go:147-151, 147-160` | OK. Negative bound → `ErrNegativeMaxSessions` `:149-150`; no relation check (`cluster/camera/provider.go:168-169` checks only negatives) ✓. |

### High-impact behaviours
- (i) **Confirmed (M).** `refuse` returns `contract.StatusError{Status: 0x01}`
  (`hm:binding.go:317-327`; `gf:contract/status.go:23-30, :42`). The typed
  handler's error passes unchanged (`gf:cluster/spec/typed.go` HandlerOf
  returns `err`; `gf:cluster/spec/selfheld.go:73-82`;
  `windowcovering_server.go:740-742` returns it as is). The dispatcher's
  concrete invoke maps it via `classifyError` (`gf:endpoint/dispatcher.go:776-779`):
  a host StatusError answers its Status (`:1508-1516`) → IM FAILURE (0x01).
- (ii) **Confirmed (M).** A setpoint write runs assign + reconcile
  (`thermostat_server.go:1397-1398`), `commitSetpoints` under `s.mu`
  (`:1405-1406`, Sink path `:1444-1459`), then `NotifyFrom(OriginControllerWrite,
  changed…)` at `:1184`; listeners are called synchronously outside the
  lock (`attribute_changes.go:83-87`). The generated watch reads both
  setpoints from the server with `Get()` inside the callback
  (`hm:rows_gen.go:981-990`) and fires on OriginCommand or
  OriginControllerWrite (`hm:binding.go` changedOrWritten), so the payload
  carries both held values after the deadband adjustment.
- (iii) **Confirmed (M)**, see Part B rows 6 and 7.
