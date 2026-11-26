# Fact note — go-fabric v0.27.0 bump, WebRTC session bounds, switch Record cite, cover stop-tilt, window and PositionAwareLift (verdict read, Opus in the Fable role, 2026-11-26)

Scope: go-hamapping #50 (Q1), #48 (Q2), #43 (Q3), #44 (Q4), the
v0.26.0 → v0.27.0 facade check (Q5). Read-only. M = read at the pin,
I = inference.

Pins and abbreviations:

- `gf:` go-fabric v0.27.0 (`~/go/pkg/mod/thingfabric.dev/go-fabric@v0.27.0`);
  `gf26:` v0.26.0 beside it; `gfCL:` `git -C go-fabric show v0.27.0:CHANGELOG.md`
- `wp:` `gf:cluster/spec/webrtctransportprovider/server_gen.go`;
  `wcdef:` `gf:cluster/spec/windowcovering/definition_gen.go`;
  `wcs:` `gf:cluster/cover/windowcovering_server.go`
- `ha:` home-assistant/core 2026.10.3 (`git show 2026.10.3:<path>`)
- `mh:` home-assistant-matter-hub 664575af (RiDDiX/home-assistant-matter-hub,
  read via `gh api …/contents?ref=664575af`; no local checkout);
  `mhC:` `packages/backend/src/matter/endpoints/legacy/cover/behaviors/cover-window-covering-server.ts`,
  `mhI:` `packages/backend/src/matter/endpoints/legacy/cover/index.ts`
- `hm:` go-hamapping-wt ee9f085 (go.mod: go-fabric v0.26.0); `h2f:` go-ha2fabric 0a47095
- prior note `notes/reads/fact-2026-11-24-cover-tilt.md` §3 (divergences 4, 5)

## Q1 (#50) — MaxSessions / MaxSessionsPerFabric

M:

- Both fields exist since v0.26.0 (`gf26:…/server_gen.go:331,339`); v0.27.0
  changes only their validation (gfCL:39, issue #600).
- `wp:314-331` MaxSessions bounds every CurrentSessions entry, the host's
  included; over it ProvideOffer answers RESOURCE_EXHAUSTED, SolicitOffer
  gets an id, DeferredOffer false, then End(OutOfResources).
  `wp:332-339` MaxSessionsPerFabric bounds the accessing fabric's entries,
  refused the same way. Both: 0 = no bound; negative refused at build.
- Independent: `gf:cluster/camera/provider_cap.go:36-56` checks the total
  (`:43`, only if > 0), then the fabric count (`:46-55`, only if > 0). No
  relation between them is validated (`gf:cluster/camera/provider.go:164-170`
  checks `< 0` only); perFabric > total is accepted.
- Build, `gf:device/defaults_camera.go:147-160`: negative (either) →
  `camera.ErrNegativeMaxSessions` (`:149-151`), judged before the port and
  not `notServed`, so `gf:device/device.go:490-491` wraps it in
  `ErrConfiguration` (`device.go:31`). No ProviderPort → `notServed(errNoPort)`
  (`:152-154`); a non-zero bound makes the cluster "declared"
  (`gf:device/options.go:297-316`, `:334-336`), so `device.go:492-496`
  fails the build. Beside a HostServer → `ErrConfiguration`
  (`gf:device/options.go:259-260`).
- `hm:ports.go:56-66` asserts `webrtctransportproviderdef.ProviderPort`,
  returns false without it (entity unserved, `hm:ports.go:14-19`), else
  sets `d.ProviderPort`. No HostServer is used in non-test `hm:*.go`.
  Its cite (`:57-58`, `defaults_camera.go:147-151 at v0.15.5`) is stale.

**Verdict.** Optional host interface, copied inside the port's `ok`
branch only, both values verbatim and independently:

```go
// WebRTCSessionBounds is the camera's session bounds (go-fabric
// cluster/spec/webrtctransportprovider/server_gen.go:314-339 at v0.27.0):
// total over every session, perFabric over one fabric's; 0 is no bound.
type WebRTCSessionBounds interface {
	WebRTCSessionBounds() (total, perFabric int)
}
// in portsWebRTCProvider, after d.ProviderPort = p:
if sb, ok := b.host().(WebRTCSessionBounds); ok {
	d.MaxSessions, d.MaxSessionsPerFabric = sb.WebRTCSessionBounds()
}
```

No clamping, no multiple or relation rule (none exists in go-fabric);
negatives pass through and fail `Plan.Build` with `ErrConfiguration`
wrapping `camera.ErrNegativeMaxSessions` (I: reached through
`BuildError`'s unwrap, `hm:plan_build_test.go:33-40`). Name the results
`total, perFabric`, not `max` (I: shadows the builtin; revive flags it).
Cite `device/defaults_camera.go:147-160 at v0.27.0` at `hm:ports.go:57-58`.
Red tests: bounds (5, 2) reach the Declare; bounds without ProviderPort →
unserved, no build error; (-1, 0) → `errors.Is(err, camera.ErrNegativeMaxSessions)`.

## Q2 (#48) — the switch event cite

M: `hm:rows_gen.go:1741,1744` call `RecordInitialPress`/`RecordShortRelease`
(the generator resolves `"Record"+event`, `hm:internal/gen/gen.go:758,770`).
At v0.27.0 `RecordInitialPress` is `gf:cluster/spec/switchcluster/server_gen.go:181-193`,
`RecordShortRelease` `:227-239`; `EmitInitialPress` `:195-202` and
`EmitShortRelease` `:241-248` are Deprecated (`:197-199`, `:243-245`).
The row's other cite `definition_gen.go:30-31,38,40,54-55` still names the
same lines at v0.27.0 (`gf:cluster/spec/switchcluster/definition_gen.go:30-31,38,40,54-55`).

**Verdict.** `hm:rows/event.yaml:15` becomes
`go-fabric cluster/spec/switchcluster/server_gen.go:181-193,227-239 @v0.27.0 (RecordInitialPress, RecordShortRelease: emitted on the endpoint the server is mounted on)`;
regenerate. Other old pins: 61 go-fabric cites in `hm:rows/*.yaml` carry
a pin below v0.26.0 (v0.11.0 ×3, v0.15.4 ×39, v0.15.5 ×6, v0.18.0 ×8,
v0.25.0 ×5). Matching the identifiers they name against the 514
Deprecated declarations of v0.27.0 (non-test, outside examples/internal/script)
finds **only `event.yaml:15`** (the match flags EmitInitialPress /
EmitShortRelease, so it reacts). Line drift of the other cites: not checked.

## Q3 (#43) — stop_cover_tilt

M:

- `ha:homeassistant/components/cover/services.py:60-62` stop_cover requires
  STOP; `:85-90` stop_cover_tilt requires STOP_TILT. Default features with
  a tilt position include STOP_TILT (`ha:…/cover/__init__.py:242-248`).
  Catalog `STOP_TILT: 64` (`go-ha-catalog@v0.4.0 data/features.json:34`).
- StopMotion is mandatory (`wcdef:50`). Parsed as a stop (`wcs:703-704`),
  sent to a Mover's `Stop` (`wcs:123-126`, `:789-799`); without a Mover the
  kit "moves at once" (`gf:device/defaults.go:436-444`). A
  `Declare.HandleStopMotion` replaces it inline
  (`gf:cluster/spec/windowcovering/server_gen.go:385-387`).
- `hm:rows/cover.yaml:93-95`: StopMotion only with STOP, payload_stop →
  `stop_cover` (`h2f:internal/adapt/command.go:44`). Without STOP no
  handler (`hm:rows_gen.go:1594-1602`): a STOP_TILT-only cover answers
  SUCCESS from the kit and nothing reaches HA.
- Method constants come from rows' `method:` (`hm:internal/gen/keys.go:69-92`);
  host cover methods at `h2f:internal/adapt/command.go:60-63`.
- matter-hub sends only `cover.stop_cover` (`mhC:382`); no stop_cover_tilt.

**Verdict.** After the STOP line:

```yaml
          - handle: StopMotion
            when: {features: {all: [STOP_TILT], none: [STOP]}}
            method: stop_tilt
            raw: {const: STOP}
```

(as the open_tilt lines, `hm:rows/cover.yaml:75-78`). Generates
`MethodStopTilt`; the host adds `hamapping.MethodStopTilt: "stop_cover_tilt"`
at `h2f:command.go:63`. STOP and STOP_TILT: `stop_cover` only (no source
says otherwise; matter-hub the same). Cite
`home-assistant/core homeassistant/components/cover/services.py:60-62,85-90 @2026.10.3`.
Neither: unchanged (I: a `discard: true` line would surface it to `Plan.OnDiscard`).

## Q4 (#44) — window and PositionAwareLift (owner: adopt matter-hub)

FeatureMaps emitted (M, `hm:rows_gen.go:1443-1453`): Lift always, Tilt on
any of OPEN_TILT/CLOSE_TILT/SET_TILT_POSITION. "Lift only" and "neither"
are LF; "lift+tilt" **and "tilt only"** are LF+TL (no tilt-only FeatureMap
is emitted). Only LF/TL enter Type/EndProductType conformance.

### 4a. Type / EndProductType for window

M: `wcdef:245` Rollershade `LF & !TL`, `:253` TiltBlindLift `LF & TL`,
`:255` Unknown `O`; `wcdef:96` RollerShade `LF & !TL`, `:120` Unknown `M`;
refusal `wcs:391-399` (`ErrEnumValue`, `:90`). mh: window lift-only
Rollershade/RollerShade (`mhC:57-64`, Alexa #312); with tilt
TiltBlindLift/Unknown (`mhC:78-102`, "a tilting roof window is no sheer shade").

| window | today | verdict (builds at v0.27.0) |
| --- | --- | --- |
| LF (lift only, neither) | Unknown / Unknown | **TypeRollershade / EndProductTypeRollerShade** (`wcdef:245,96`) |
| LF+TL (lift+tilt, tilt only) | Unknown / Unknown | **TypeTiltBlindLift / EndProductTypeUnknown** (`wcdef:253,120`) |

Rollershade is not allowed under LF+TL. The brief said "keep today's
choice" there; I recommend matter-hub's measured TiltBlindLift (the Type
every other tilting class has in the row; the owner chose matter-hub).
If the owner keeps Unknown/Unknown, only the `window` key of the tilt
Type map stays. Edits: `window: TypeRollershade` / `EndProductTypeRollerShade`
in the `none:`-tilt maps, `window: TypeTiltBlindLift` in the `any:`-tilt
Type map. Drop `window` from the Unknown caveat; add
`{when: {device_class: [window], features: {any: [OPEN_TILT, CLOSE_TILT, SET_TILT_POSITION]}}, reason: "a tilting window has no WindowCovering product: EndProductType is Unknown"}`.
Rewrite `hm:docs/by_design.md:155-163` (the Alexa claim is matter-hub's, I).

### 4b. PositionAwareLift always

M, go-fabric: PA_LF needs LF only (`wcs:374-377`). With PA_LF,
Target/CurrentPositionLiftPercent100ths are mandatory and nullable
(`wcdef:37,40`, "LF & PA_LF … X"); CurrentPositionLiftPercentage is served
too (`wcs:135-137`); **GoToLiftPercentage becomes mandatory** (`wcdef:51`,
"LF & PA_LF, [LF]"; without PA_LF it is the served optional, `wcs:151-153`).
No initial position reads null (`gf:device/defaults.go:449-458`); only a
known one is checked (`wcs:403-414`).

M, HA: `current_position` is in the state whenever `current_cover_position`
is not None (`ha:…/cover/__init__.py:217-218`), whatever the features;
default features add SET_POSITION exactly when a position exists
(`:239-240`), explicit features win (`:229-230`). So without SET_POSITION
it is normally absent but may be present. `set_cover_position` requires
SET_POSITION (`ha:…/cover/services.py:49-57`).

M, matter-hub: Lift and PositionAwareLift always (`mhI:34-49`; #78 Apple
"clima", matter-hub's claim, I). Lift position: `current_position`, else
state closed → 0, open → 100 (HA scale), else null (`mhC:330-342`).
GoToLiftPercentage without position control: ≥ 50 % open → open_cover,
else close_cover (`mhC:392-410`), a threshold.

**Verdict — row shape.**

1. `- PositionAwareLift` unconditional (replaces `hm:rows/cover.yaml:33`).
2. Lift position: a state line first, then the position line ungated (a
   later line wins when it has a value, `hm:docs/rows.md:106-109`; an
   absent key sets nothing, `hm:binding.go:142-170`):

```yaml
          - {attribute: CurrentPositionLiftPercent100ths, from: state, enum: {open: 0, closed: 10000, opening: null, closing: null}, when: {features: {none: [SET_POSITION]}}}
          - {attribute: TargetPositionLiftPercent100ths, from: state, enum: {open: 0, closed: 10000, opening: null, closing: null}, when: {features: {none: [SET_POSITION]}}}
          - {attribute: CurrentPositionLiftPercent100ths, from: state.current_position, offset: -100, scale: -100, min: 0, max: 10000}
          - {attribute: TargetPositionLiftPercent100ths, from: state.current_position, offset: -100, scale: -100, min: 0, max: 10000}
```

3. GoToLiftPercentage without SET_POSITION makes no wrong call: 0 and
   10000 keep their exact call, anything between is refused.
   - OPEN/CLOSE group and "neither" group: `not_in: [0]` → `in: [10000]`
     (close), plus a refusing line `if: {field: LiftPercent100thsValue, not_in: [0, 10000]}`.
   - tilt-only without SET_TILT_POSITION: 0 → open_tilt, 10000 →
     close_tilt, else refuse.
   - SET_POSITION (set_position) and tilt-only with SET_TILT_POSITION
     (`method: tilt`): unchanged, both exact.
   The row format has no refusal (`hm:row.go:156-179`; `discard` answers
   success): add one, e.g. `refuse: FAILURE` → `err = spec.Errorf(im.StatusFailure, …)`
   in the typed handler (it already returns `err`, `hm:rows_gen.go:1604`).
   Status is I: UNSUPPORTED_COMMAND is wrong for a command in
   AcceptedCommandList, CONSTRAINT_ERROR is for values outside 0..10000;
   FAILURE is go-fabric's answer to a refused move (`wcs:794`, `:851`).
   **Diverges** from matter-hub's 50 % threshold (`mhC:399-410`): record
   in `docs/by_design.md` (a threshold turns 40 % into "close fully").
4. Cites: `wcdef:37,40,51`, `device/defaults.go:449-458` @v0.27.0;
   `ha cover/__init__.py:217-218,239-240`; `mhI:34-49`,
   `mhC:57-64,78-102,330-342,392-410` @664575a. Drop the
   `windowcovering_server.go:144-158,624-634 @v0.18.0` cite
   (`hm:rows/cover.yaml:27`): its no-PA_LF path no longer applies.

Red tests: window/3 → Rollershade/RollerShade; window/255 → TiltBlindLift/Unknown;
features 3, state closed, no position → lift 10000, opening → null; GoToLift
0 → open_cover, 10000 → close_cover, 4000 → FAILURE, no command; 240 + 4000 → tilt 60.

## Q5 — v0.26.0 → v0.27.0 on go-hamapping's facade

M:

- gfCL:15: `make apidiff` reports no incompatible change; five
  deprecations, removable v0.29.0 (gfCL:52-55): `cluster/core.NewAccessControl`,
  `AccessControl.SetLimits`, `ARLClusterID`, `AccessControl.NotifyLocalEntryInstalled`,
  `cluster/camera.Provider.PeerGone`. The Deprecated-declaration diff
  (509 → 514) finds exactly these. None is used in `hm` (name grep over
  `*.go`, `internal/*/*.go`, tests included).
- Non-test diff: `schema`, `eligibility`, `cluster/cover`, `fabrictest`
  unchanged; `contract` comments only (`dataversion.go:26-28`, `matter.go:238`);
  `cluster/spec`: 132 `server_gen.go` change only the HostServer doc
  ("the ports and the kit settings … are refused beside it", e.g.
  `airquality/server_gen.go:161-162`), `spec/server.go` an unexported
  callback; `device`: `options.go:259-260,288-323`,
  `defaults_camera.go:149-151,155` (Q1's refusal: behaviour, not API).
  `node`, `cluster/camera`, `controller/client` changed (gfCL:38-43);
  `hm` uses them only in tests.
- Build check: `hm` against v0.27.0 via a scratch `-modfile`
  (`go get thingfabric.dev/go-fabric@v0.27.0`, repo files untouched):
  `go vet ./...` exit 0, `go test ./...` every package ok.

**Verdict.** No breaking change, no deprecation in `hm`'s use: the bump
is `go.mod`/`go.sum` plus the cites of Q1/Q2. The one behaviour change
on its path (a bound without a port or beside a HostServer fails the
build) is avoided by Q1's copy rule.
