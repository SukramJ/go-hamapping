# Fact note — thermostat bounds refused at build (#45), one range command in heat_cool (#39) (verdict read, Opus in the Fable role, 2026-11-26)

Scope: issues #45 and #39. Read-only. The implementer builds from this
note. M = read at the pin, I = inference (labelled).

Pins and abbreviations:

- `gf:` go-fabric v0.27.0 (`~/go/pkg/mod/thingfabric.dev/go-fabric@v0.27.0`)
- `ts:` `gf:cluster/thermo/thermostat_server.go`; `sp:` `gf:cluster/thermo/setpoints.go`;
  `tt:` `gf:cluster/thermo/thermostat_typed.go`; `tsg:` `gf:cluster/spec/thermostat/server_gen.go`
- `ha:` home-assistant/core 2026.10.3 (`git show 2026.10.3:<path>`)
- `hm:` go-hamapping worktree at ee9f085 (go.mod pins go-fabric v0.27.0, go-ha-catalog v0.4.0)
- `cat:` go-ha-catalog v0.4.0 `data/mqtt.json`

All Matter temperatures are int16 in 0.01 °C; MinSetpointDeadBand is int8 in 0.1 °C.

## 1. go-fabric: defaults and what refuses at build (M)

- `gf:device/defaults.go:485-532`: the device's Thermostat server is
  `thermo.DefaultThermostatConfig()`, overlaid by the declaration's Initial
  through `configFrom` (`gf:device/defaults_declared.go:86-108`: a Config field
  named like the attribute takes the Initial value and the value is removed).
- `ts:362-372`: defaults AbsMinHeat 700, AbsMaxHeat 3000, AbsMinCool 1600,
  AbsMaxCool 3200, InitialHeating 2000, InitialCooling 2600.
- `ts:135-139`, `:549`, `:557-564`: MinSetpointDeadBand default 20 (2.0 °C);
  a declared one replaces it, a negative one is ErrSetpoints. `sp:65-67`: the
  reconcile's deadband is `MinSetpointDeadBand * 10` (200), and 0 without AUTO.
- `ts:572-591`: per served side the user limits default to the absolute ones
  (`Min*SetpointLimit = or(cfg.Min…, cfg.AbsMin…)`).
- `ts:597-610`: `reconcileInitial`; an error is returned from `New`
  (→ `device.New` fails). `ts:104-108`: `ErrSetpoints`.
- `sp:389-403` reconcileInitial: violation → `fixSetpoints` with the heat
  side's limits, occupied heating setpoint and MinCoolSetpointLimit as
  "changed" → violation again → `ErrSetpoints: <first>; after adjusting, <remaining>`.
- `sp:315-352` violation, in order per served side: `absMin > absMax`
  ("absMin%sSetpointLimit (%d) is above absMax%sSetpointLimit (%d)", `:324-325`);
  user limits not ordered within abs (`:327-329`); a setpoint outside the user
  limits (`:330-334`); with AUTO `maxCool-maxHeat < deadband`,
  `minCool-minHeat < deadband`, `cool-heat < deadband` per pair (`:337-349`).
- `sp:449-475` fixUserLimits returns without repair when `absMin > absMax`
  (`:458-460`) → shape (a) is never repairable.
- `sp:479-515` fixLimitDeadband: heat side "changed" → `newCool = heat + dead`;
  outside the cool abs limits the cool limit is pinned to absMaxCool (upper)
  or absMinCool (lower) and the heat limit to `pinned - dead`.
- `sp:519-554` fixSetpointRange clamps both setpoints into the user limits,
  then moves the cooling setpoint to `heat + dead` (heat "changed").

## 2. go-hamapping: what is declared (M)

- `hm:rows/climate.yaml:59-62`: Heating when modes ∋ heat|heat_cool, Cooling
  when cool|heat_cool, **AutoMode when modes ∋ heat_cool**. (HA mode `auto`
  is not mapped, `:166-167`.)
- `:68-75`: AbsMin/Min Heat **and** Cool limits all from `extra.min_temp`,
  AbsMax/Max Heat and Cool from `extra.max_temp`, °C ×100, °F `(x-32)×55.555…`.
  No MinSetpointDeadBand is declared anywhere (grep of rows/: none).
- `hm:rows_gen.go:808-889`: these go into `d.Initial` only when
  `b.number(nil, …)` is ok; the setpoints are **not** in Initial (grep
  `Initial.Occupied*Setpoint` in rows_gen.go: no hit) → at build the server
  starts at 2000/2600 and `apply` (`:922-979`, `Wrap(…).Set(p)`) sets them later.
- `hm:value.go:73-81`: a missing extra falls back to the catalog default;
  `cat:` climate and water_heater `min_temp`/`max_temp` have `"default": null`
  (read with python over v0.4.0) → a missing bound is **absent** from Initial
  and go-fabric's default of §1 takes its place.
- `hm:value.go:313-334`: `conv` rounds half away from zero, clamped to int16.
- `hm:options.go:89-125`: `invertedBounds` refuses only when **both** bounds are
  stated and `min_temp > max_temp` (float compare, before conversion).
- `hm:rows_climate_test.go:103-159`: the matrix skips 20..20/68..68 for the
  five-mode shape and lists "min 50 alone, max 5 alone" as known gaps.

## 3. Home Assistant (M)

- `ha:homeassistant/components/climate/__init__.py:239-254`: `capability_attributes`
  **always** carries `min_temp` and `max_temp`. `:653-668`: unset, they are
  `DEFAULT_MIN_TEMP`/`DEFAULT_MAX_TEMP` converted from °C into the entity's
  `temperature_unit`; `:105-106` (and `climate/const.py:135-136`) 7 and 35.
- `ha:homeassistant/components/water_heater/__init__.py:52-53, :164-172, :349-366`:
  the same, defaults 110 and 140 **°F**.
- `ha:homeassistant/components/mqtt/climate.py:337-338, :498-501` and
  `mqtt/water_heater.py:146-147, :207-210`: the MQTT config keys are optional;
  unset, the entity keeps the class default above.
- `ha:homeassistant/components/climate/services.py:49-58`: `target_temp_high` and
  `target_temp_low` are `Inclusive(…, "temperature")` — **given together or not
  at all**; `:121-130`: refused only when `low > high` (low == high admitted);
  `:131-157`: each value checked against min_temp/max_temp.
  `climate/const.py:131-132`: the keys are `target_temp_high`, `target_temp_low`.
- HA carries **no deadband** attribute for climate (none in `climate/const.py`
  ATTR_* list, M by grep) — I: the device's own deadband is not in the source.

## Verdict 1 (#45) — what Map must call Unmappable

**Both shapes are measurable; neither needs an invented value.**

(a) One bound stated, the other beyond go-fabric's default. Measured cause:
`sp:324-325` + `sp:458-460`. Today the missing bound *is* a substituted value
— go-fabric's matter.js element default (§1) — while the HA entity carries
7/35 °C (climate) or 110/140 °F (water_heater) for it (§3). Two honest builds:

- **B (recommended): extend the source.** State the missing bound with
  Home Assistant's own entity default, cited to `ha:…climate/__init__.py:653-668`
  / `water_heater/__init__.py:349-366`, converted from **its** unit (°C for
  climate: 700 / 3500; °F for water_heater: conv((110-32)·55.555…) = 4333,
  conv((140-32)·55.555…) = 6000). Where it lives is a decision: go-ha-catalog
  (an entity-class default beside the MQTT default, cat: carries null today)
  or a cited row-level default. Then heat and cool have identical absolute
  limits `[L, H]` and the predicates collapse to:
  1. `L > H` → Unmappable, `"min_temp %v is above max_temp %v"` — the existing
     `invertedBounds`, now over the filled values (name the default:
     `"(Home Assistant's default)"` on the filled side).
  2. AutoMode selected (modes ∋ heat_cool) and `H - L < 200` →
     Unmappable, reason e.g. `"min_temp %v and max_temp %v are closer than the
     thermostat's 2.0 °C heat/cool deadband"`.
  Exactness of (2) (M, by walking `sp:431-554` with abs heat = abs cool =
  [L,H], dead 200): `H-L ≥ 200` always repairs (heat limits → [L, H-200],
  cool → [L+200, H], setpoints clamped); `H-L < 200` always fails (20..21 °C:
  max pair pinned to 2100/1900, min pair to 2000/1800 → "Heat setpoint limits
  [1800, 1900] are not ordered within absolute limits [2000, 2100]").
  20..22 builds. Compare **L and H as the int16 values `conv` produces**
  (`hm:value.go:313-334`), not the HA floats: go-fabric sees the rounded ones.
  Behaviour change to accept knowingly: `{15, nil}` gets AbsMax 3500 instead of
  3000; water_heater `{nil, 25}` becomes unmappable (min 43.33 °C > 25) — which
  is what the HA entity itself states.
- **A (if B is declined): refuse by go-fabric's defaults.** (a) per served
  side `absMin > absMax`, with the unstated one = Heat 700/3000, Cool 1600/3200
  (`ts:362-372`); reason names go-fabric's default. (b) then has no closed form
  (heat and cool abs differ when one bound is missing); the exact predicate is
  `thermo.New(cfg)` returning `thermo.ErrSetpoints` on the config built from
  the same Initial (I: usable as the oracle, it is exported, `ts:533`).

Deadband option, not recommended without the owner: go-fabric would take a
declared `MinSetpointDeadBand` (`ts:177-182`, configFrom by name; I: the Patch
`*int8` converts — unverified). HA's service admits low == high (§3), but
the device's real deadband is **not carried** by HA, so declaring 0 is a
convention → not performable from the source.

State at Map time (M): (b) does **not** depend on target_temp_low/high —
the setpoints are not in Initial (§2) and, with `H-L ≥ 200`, reconcileInitial
repairs any setpoint pair. Apply time (M, `tt:51-70, :102-190, :478-534`):
`Set(p)` is one all-or-nothing change; user limits first, then setpoints in
id order (Cooling 0x0011 before Heating 0x0012, `applyRank` `tt:527-534`),
each asserted within its effective limits (`tt:500-506`, refused as
`spec.ErrInvalidValue`) and reconciled (`tt:508`). Consequences:
- a narrow target range (20..20) is **not refused**: cool 20 stored, then
  heat 20 moves cool to 22 (`sp:537-544`); reported `OriginHostSet`, which the
  `change` watch ignores (`hm:binding.go:233-239`) → Matter shows 20/22 while
  HA holds 20/20 (no echo).
- a target outside the **narrowed** AUTO limits is **refused**: with bounds
  20..23 the heat limit is [20, 21] (I: by the walk above), so HA
  target_temp_low 22 makes `apply` return ErrInvalidValue and nothing of that
  Patch is stored. This is a separate Apply gap (I), not #45.

## Verdict 2 (#39) — one range command in heat_cool

How the three paths reach the host (M):

| Path | Host hook | Calls | Pair seen | vs reconcile |
|---|---|---|---|---|
| Write request with heat **and** cool | veto `OnWrite` (`ts:1150-1154`, `tsg:792-801`) | one per attribute (`gf:im/write.go:413`, `:589/:595`; `gf:im/dispatcher.go:221-224`) | `next` = stored state with the written value in place (`gf:cluster/spec/typed.go:228-230`) | **before** assert and reconcile (`ts:1386-1394` later) |
| same | `Sink.SetSetpoint` (`ts:1420-1431`) | one per attribute | value after reconcile + companions it moves (`ts:327-346`) | after reconcile, before store |
| same | change notification | one per attribute, `OriginControllerWrite`, written + companions (`ts:1165-1172`) | stored state | after store |
| write of one setpoint | as above, once | | | |
| SetpointRaiseLower Both (AUTO) | `Sink.SetSetpoint` once, cool primary + heat companion (`ts:1631-1660`, `:1708-1735`); one notification `OriginCommand` naming both (`ts:1737-1752`) | one | both | no reconcile (`ts:1706`) |

**go-fabric gives no per-write-request hook** (M): the IM loops the request's
writes and calls `Dispatcher.Write` per element; the contract's write
capabilities are per attribute (`gf:contract/change_naming.go:29-35`,
`ChangeReportingWriter`, `contract/matter.go:278`); `im.WriteTransaction`
(`gf:im/write.go:753-771`) is internal (holds, last path), with no callback.
So **"once per write request" cannot be guaranteed** today.

Today (M): `hm:rows_gen.go:891-918` (veto) emits `temperature_low` /
`temperature_high` per attribute in Auto from the pre-reconcile `next`;
`:981-1004` (watch, OriginCommand only) emits both for one RaiseLower Both →
two commands, and HA requires both keys together (§3).

Shape to build (recommended):
- method **`temperature_range`**, JSON payload
  `{"target_temp_low": <OccupiedHeatingSetpoint>, "target_temp_high": <OccupiedCoolingSetpoint>}`
  (keys `ha:climate/const.py:131-132`; values through the existing `*setpoint` /
  `*coolpoint` conversion, digits 1).
- emitted when SystemMode is Auto, from the **post-reconcile pair**: once per
  admitted setpoint write and once per SetpointRaiseLower. Two sources do that:
  - the **change watch** extended to `OriginControllerWrite` + `OriginCommand`
    for these two ids, reading the pair from `Get()` — no new port; I: the
    watch delivery is asynchronous, so two writes in one request may emit the
    final pair twice (harmless duplicate, never an inverted pair);
  - or a declared **Sink** (pair = value + companions, the other read with
    MatterRead, allowed `ts:308-312`) — needs a Sink port and must return nil
    for SetSystemMode / SetTemperatureSetpointHold.
  Not the veto: its pair is pre-reconcile (`ts:1150` vs `:1398`), so heat 24 with
  stored cool 22 would send low > high, which HA refuses (`services.py:121-130`);
  and it fires before the server's limit assert, so a write the server then
  refuses has already reached HA (M: `ts:1150-1154` precede `ts:1391-1394`).
- a request carrying both setpoints yields **two** `temperature_range` commands
  (first: new heat + reconciled cool; second: final pair). Honest wording for
  the issue: "once per admitted setpoint change, always a consistent pair".
  True once-per-request needs a go-fabric hook → a go-fabric issue (per-write-
  interaction commit callback); a time-window coalescer in go-hamapping would be
  a convention (window length not carried by any source) — not recommended.
- **Unchanged outside Auto:** the veto rows `temperature` on heat (SystemMode
  not Auto) and on cool (not Auto) (`rows/climate.yaml:89-104` minus the two
  Auto rows), and the change rows `temperature` in heat/cool
  (`:105-112`). In Auto, the four `temperature_low` / `temperature_high` rows
  (`:93-96`, `:101-104`, `:113-120`) go, or a Both raise and every Auto write
  is reported twice.
