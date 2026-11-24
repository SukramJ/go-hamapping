# Fact note — cover tilt Type/EndProductType, ModeSelect limits, typed build error (verdict read, Opus in the Fable role, 2026-11-24)

Scope: HR-P2-adpt-33, -34, -36 of go-ha2fabric's review
(`go-ha2fabric-review/notes/review/2026-11-23/phase-2-X2.md`). Read-only.
The implementer writes rows and code from this note. M = read at the pin,
I = inference.

Pins and abbreviations:

- `gf:` go-fabric v0.25.0 (`~/go/pkg/mod/thingfabric.dev/go-fabric@v0.25.0`)
- `wcdef:` `gf:cluster/spec/windowcovering/definition_gen.go`
- `msdef:` `gf:cluster/spec/modeselect/definition_gen.go`
- `ha:` home-assistant/core 2026.10.3
- `mh:` home-assistant-matter-hub 664575a; `mhL/` =
  `packages/backend/src/matter/endpoints/legacy/cover/`, `mhB/` =
  `packages/backend/src/matter/behaviors/`
- `mjs:` matter.js b8979071 (local HEAD; only to confirm go-fabric's
  conformance strings)
- `hm:` go-hamapping main d570fdd (v0.6.0)

## 1. What go-fabric accepts (M)

- `gf:cluster/cover/windowcovering_server.go:375-378`: FeatureMap must
  contain Lift (0x1) or Tilt (0x2); PA_LF needs LF, PA_TL needs TL.
- `:391-399`: Type and EndProductType are checked with
  `inst.CheckValue` against the instance's features; a disallowed member
  is `ErrEnumValue` (`:90`, "cover: Type or EndProductType not allowed
  under the FeatureMap").
- TypeEnum conformance, `wcdef:273-283`:
  - `LF & !TL`: Rollershade 0, Rollershade2Motor 1, RollershadeExterior 2,
    RollershadeExterior2Motor 3, Drapery 4, Awning 5, ProjectorScreen 9
  - `LF & !TL | !LF & TL`: Shutter 6
  - `!LF & TL`: TiltBlindTiltOnly 7
  - `LF & TL`: TiltBlindLift 8
  - `O`: Unknown 255 (any FeatureMap)
- EndProductTypeEnum conformance, `wcdef:104-128`:
  - `LF & !TL`: RollerShade 0 … LayeredShade2D 7, LateralLeftCurtain 14,
    LateralRightCurtain 15, CentralCurtain 16, RollerShutter 17,
    ExteriorVerticalScreen 18, AwningTerracePatio 19,
    AwningVerticalScreen 20
  - `LF & TL`: SheerShade 8, InteriorBlind 10, VerticalBlindStripCurtain 11,
    InteriorVenetianBlind 12, ExteriorVenetianBlind 13
  - `!LF & TL`: TiltOnlyInteriorBlind 9
  - `LF & !TL | !LF & TL`: TiltOnlyPergola 21, SwingingShutter 22,
    SlidingShutter 23
  - `M`: Unknown 255 (any FeatureMap)
- matter.js agrees (`mjs:packages/model/src/standard/elements/window-covering-cluster.element.ts:158,160,175-177,188`).

So under LF+TL the only Types are **TiltBlindLift** and **Unknown**; the
only EndProductTypes are SheerShade, InteriorBlind,
VerticalBlindStripCurtain, InteriorVenetianBlind, ExteriorVenetianBlind
and Unknown. Every pair of today's row is `LF & !TL` (or Unknown) and is
refused under LF+TL: that is HR-P2-adpt-33.

## 2. Home Assistant's inputs (M)

- Device classes, `ha:homeassistant/components/cover/const.py:67-80`:
  awning, blind, curtain, damper, door, garage, gate, shade, shutter,
  window (+ none).
- Features, `:44-55`: OPEN 1, CLOSE 2, SET_POSITION 4, STOP 8,
  OPEN_TILT 16, CLOSE_TILT 32, STOP_TILT 64, SET_TILT_POSITION 128,
  **SPEED 256** (new; the row does not read it, nothing to do here).
- Services require features, `ha:homeassistant/components/cover/services.py`:
  open_cover OPEN `:35-38`, close_cover CLOSE `:42-45`, set_cover_position
  SET_POSITION `:49-57`, open_cover_tilt OPEN_TILT `:72-75`,
  close_cover_tilt CLOSE_TILT `:79-82`, set_cover_tilt_position
  SET_TILT_POSITION `:93-100`.
- Tilt-only covers exist in core: SwitchBot Blind Tilt,
  `ha:homeassistant/components/switchbot/cover.py:176-181`
  (`device_class` BLIND; OPEN_TILT | CLOSE_TILT | STOP_TILT |
  SET_TILT_POSITION = 240, no OPEN/CLOSE/SET_POSITION).

Terms used below (on HA's `supported_features`):

- **tilt** = any of OPEN_TILT, CLOSE_TILT, SET_TILT_POSITION — the row's
  existing Tilt predicate (`hm:rows/cover.yaml:29`); keep exactly this
  predicate for the Type/EndProductType switch so the two cannot disagree.
- **lift** = any of OPEN, CLOSE, SET_POSITION.
- lift only = lift ∧ ¬tilt; lift+tilt = lift ∧ tilt; tilt only =
  ¬lift ∧ tilt; neither = ¬lift ∧ ¬tilt.

## 3. Verdict — the decision table (answer to Q1)

FeatureMap as today (Lift always, Tilt on tilt; see §4). So the Matter
FeatureMap is LF for "lift only" and "neither", LF+TL for "lift+tilt" and
"tilt only"; the table therefore has two columns.

| device_class | LF only (lift only, neither) — Type / EndProductType | LF+TL (lift+tilt, tilt only) — Type / EndProductType |
| --- | --- | --- |
| blind | Rollershade / RollerShade (unchanged) | **TiltBlindLift / InteriorBlind** |
| shade | Rollershade / RollerShade (unchanged) | **TiltBlindLift / SheerShade** |
| "" (none) | Rollershade / RollerShade (unchanged) | **TiltBlindLift / InteriorBlind** |
| shutter | Shutter / RollerShutter (unchanged) | **TiltBlindLift / ExteriorVenetianBlind** |
| curtain | Drapery / CentralCurtain (unchanged) | **TiltBlindLift / VerticalBlindStripCurtain** |
| awning | Awning / AwningTerracePatio (unchanged) | **TiltBlindLift / Unknown** |
| garage, gate, door, damper, window | Unknown / Unknown (unchanged, caveat stays) | Unknown / Unknown (unchanged; already builds, caveat stays) |

All bold cells are accepted by go-fabric under LF+TL (§1); all
unchanged cells are accepted under LF (§1). Only the LF+TL column is new.

Row form (I, from `hm:row.go:229-243` and the climate precedent
`hm:rows/climate.yaml:64-67`): keep the two `initial` enums with
`when: {features: {none: [OPEN_TILT, CLOSE_TILT, SET_TILT_POSITION]}}`
and add a second pair with `when: {features: {any: [OPEN_TILT,
CLOSE_TILT, SET_TILT_POSITION]}}` mapping the classes as in the right
column. Predicates disjoint, so order does not matter.

Why each cell:

- blind → InteriorBlind: HA's blind is the slatted, tiltable cover
  (SwitchBot's tilt blind is `blind`, §2); InteriorBlind is Matter's
  generic LF&TL blind. InteriorVenetianBlind is narrower and would be a
  guess. Same as matter-hub.
- shade → SheerShade: the one LF&TL shade (zebra/dual sheer, the only
  shade that tilts vanes). Same as matter-hub.
- shutter → ExteriorVenetianBlind: a shutter that tilts is the exterior
  slatted kind (Raffstore); RollerShutter/SwingingShutter/SlidingShutter
  are not allowed with LF+TL. Same as matter-hub.
- curtain → VerticalBlindStripCurtain: the LF&TL curtain. Same as
  matter-hub.
- awning → Unknown: no LF&TL awning product; Type TiltBlindLift states
  the axes. Same as matter-hub (Type TiltBlindLift, EndProductType
  Unknown via the `?? Unknown` fallback).
- none → InteriorBlind: **diverges** from matter-hub (SheerShade, its
  feature-derived default). Reason: an unclassed cover that tilts is by
  HA's own model a blind (tilt is the blind's trait), and InteriorBlind is
  the generic one; SheerShade is a specific shade product. Both build;
  this is a deliberate divergence, not a gap.

### Cross-check with matter-hub (M)

- Feature map: Lift and PositionAwareLift always, Tilt on OPEN_TILT or
  SET_TILT_POSITION, PA_TL on SET_TILT_POSITION
  (`mh:mhL/index.ts:39-63`).
- Class table `mh:mhL/behaviors/cover-window-covering-server.ts:30-65`;
  feature correction `:67-111` — with tilt: Type TiltBlindLift,
  EndProductType RollerShutter→ExteriorVenetianBlind,
  InteriorBlind→InteriorBlind, CentralCurtain→VerticalBlindStripCurtain,
  RollerShade→SheerShade, window and the rest → Unknown (`:78-102`);
  lift-only blind → Rollershade/Unknown (`:104-109`).
- No mapping (no class, garage, gate, door, damper) falls to
  `defaultShape` (`mh:mhB/window-covering-server.ts:359-376`):
  LF+TL TiltBlindLift/SheerShade, TL TiltBlindTiltOnly/TiltOnlyInteriorBlind,
  else Rollershade/RollerShade; override applied at `:521-542`.

Divergences, all allowed:

1. none + tilt: InteriorBlind vs SheerShade (above).
2. lift-only blind: ours Rollershade/RollerShade, matter-hub
   Rollershade/Unknown (`:104-109`). Ours builds (`wcdef:104` LF&!TL) and
   says more; keep.
3. garage/gate/door/damper (any features): ours Unknown/Unknown (the
   existing caveat), matter-hub defaultShape (a roller shade or a sheer
   shade). Ours does not claim a product the device is not; keep.
4. window: ours Unknown/Unknown; matter-hub Rollershade/RollerShade
   lift-only and TiltBlindLift/Unknown with tilt, citing Alexa's routine
   picker dropping an EndProductType of Unknown (`:57-63`, their #312).
   **Pre-existing, outside adpt-33**; the Alexa claim is matter-hub's,
   not measured here (I). If the owner wants Alexa parity for windows,
   that is a separate row decision; it is not needed for the build.
5. PositionAwareLift: matter-hub always (Apple Home, their #78); ours
   only with SET_POSITION (`hm:rows/cover.yaml:28`). Pre-existing,
   outside adpt-33; noted because it bears on §4.

No gap in Type/EndProductType remains after the table.

## 4. Does the FeatureMap need to change? (Q2)

**Verdict: no.** Keep Lift always (it is also what `gf:…/windowcovering_server.go:375`
needs for a cover with neither lift nor tilt), Tilt on the tilt predicate.
A tilt-only cover then is LF+TL and takes the right column of §3.

Reasons:

- Builds: LF+TL accepts TiltBlindLift and the five LF&TL products (§1).
- matter-hub keeps Lift for tilt-only covers too and routes the lift
  commands to tilt instead (`mh:mhL/behaviors/cover-window-covering-server.ts:242-258`,
  `:385-395`; their #350, SwitchBot Blind Tilt 240). That is the
  field-tested shape; a tilt-only FeatureMap (TiltBlindTiltOnly /
  TiltOnlyInteriorBlind; Shutter / SwingingShutter; TiltOnlyPergola for an
  awning) is spec-clean but used by neither bridge, and how Apple, Google
  and Alexa render a WindowCovering without Lift is not known here (I).
  If the owner prefers spec-purity, that alternative builds too
  (`wcdef:279-280`, `:113`, `:125-127`) but needs a controller test first.

**Adjacent gap the implementer must close in the same wave** (else a
tilt-only cover builds but its lift commands fail; M):

- Today UpOrOpen/DownOrClose send `payload_open`/`payload_close`
  (`hm:rows/cover.yaml:44-49`), which the host turns into
  open_cover/close_cover (`go-ha2fabric internal/adapt/command.go:44`).
  For a cover without OPEN/CLOSE, HA refuses (`ha:…/services.py:35-45`).
  Same for GoToLiftPercentage without SET_POSITION (`hm:rows/cover.yaml:55-62`).
  Fix: for ¬lift ∧ tilt, route UpOrOpen / DownOrClose /
  GoToLiftPercentage to tilt, as matter-hub (`:385-395`).
- Second, also pre-existing: GoToTiltPercentage always uses
  `method: tilt` (set_cover_tilt_position, `hm:rows/cover.yaml:63-66`),
  which HA refuses without SET_TILT_POSITION (`services.py:93-100`).
  matter-hub falls back to open/close_cover_tilt (`:283-297`, `:307-327`).
  go-hamapping has no method for open_cover_tilt / close_cover_tilt
  (`hm:rows_gen.go:60-89`; host table `command.go:44`), so this needs a
  method (and the host's service) — a cross-repo step; at least the
  SET_TILT_POSITION case (UpOrOpen → tilt 100, DownOrClose → tilt 0) can
  be done with `method: tilt` and a const.

Red tests: blind/255, blind/240 (SwitchBot), shutter/255, curtain/255,
shade/255, awning/255, none/255, garage/255 all build; blind/240's
UpOrOpen sends a tilt command, not open_cover. Plus the review's own
suggestion: a test that every "mappable" cover shape in a features ×
classes matrix builds (that probe found this).

## 5. adpt-34: SupportedModes max (Q3)

M: `msdef:257-263` — SupportedModes `Constraint: spec.Constraint{Text:
"max 255", Max: &spec.Bound{Int: 255}}` (`:262`). Today
`hm:options.go:81` `maxModes = 256`, checked at `:107`. **Set
`maxModes = 255`.** The mode values 0..254 still fit uint8 (`msdef:56`);
the reason string at `:108` follows. The comment at `:77-79` should cite
`msdef:262` at v0.25.0 for the list length, not only the uint8 of Mode.

## 6. adpt-36: equal labels after the 64-byte cut (Q4)

M:

- Label: `string, M, max 64, F` (`msdef:55`, `:98`). No uniqueness
  constraint in go-fabric's generated definition.
- Matter's ModeSelect text requires uniqueness only of **Mode**:
  SupportedModes "Each entry in this list shall have a unique value for
  the Mode field" (`mjs:packages/model/src/standard/resources/mode-select-cluster.resource.ts:65-69`),
  Mode field "shall be unique" (`:151-154`); the Label field has no
  uniqueness rule (`:144-149`). Contrast: Mode **Base** requires a unique
  Label (`mjs:packages/model/src/standard/resources/mode-base.resource.ts:53-55`)
  — not the cluster used here.
- go-fabric's ModeSource contract: "each Mode distinct"
  (`gf:cluster/spec/modeselect/server_gen.go:234-235`), nothing on labels.
- HA: `select` does not require unique options
  (`ha:homeassistant/components/select/__init__.py:125`, `:155-162` only
  checks membership); `input_select` refuses duplicates
  (`ha:homeassistant/components/input_select/__init__.py:51-61`). So equal
  labels arise from the cut and, for `select`, also from exact duplicate
  options.

**Verdict: number the repeats; do not call the select unmappable.**
Equal labels are legal in ModeSelect, so refusing would drop a working
entity (the command goes by index and is right, `hm:options.go:144`) for
a cosmetic reason. Numbering makes the choice readable on a controller
and costs nothing in correctness. Rule (I, for the implementer): after
the cut, a label equal to an earlier one gets the suffix ` (n)` (n = its
occurrence, 2, 3, …); the base is cut so base+suffix ≤ 64 bytes on a
rune boundary (the existing `cut`, `hm:options.go:67-75`); if the
numbered label still collides, increase n. The option sent to HA stays
the original string by index (unchanged). Exact duplicates of a `select`
get numbered the same way. Red test: 255 options sharing a 64-byte
prefix give 255 distinct labels, each ≤ 64 bytes, valid UTF-8, and
ChangeToMode(5) sends option 5.

## 7. A typed build error carrying the entity key (Q5)

M, go-hamapping today:

- Exported errors: only the sentinel `ErrNotBuilt` (`hm:plan.go:21-22`,
  `errors.New("hamapping: plan not built")`); unexported `errNoEndpoint`
  (`hm:events.go:17`). No exported error type yet; no `errors.As` in the
  package.
- Every message is prefixed `hamapping: ` (`hm:plan.go:22,397,404,445`,
  `hm:rowports.go:55,96,105`).
- Build's two failure sites already format the key and row:
  `"hamapping: entity %s (row %s): %w"` (`hm:plan.go:404`, device) and
  `"hamapping: entity %s (row %s): part %s: %w"` (`:445`, a part).
- `Endpoint` holds `Key`, `Row`, `Riders` (`hm:plan.go:40-60`); riders
  are other entities' keys built onto the same device (`:57-59`, `:411`).

M, go-fabric's convention for typed errors (the house style to mirror):
`<What>Error` struct, pointer receiver, documented fields, an `Error()`
with the package prefix, optional `Is` to a sentinel —
`gf:client/exchange/errors.go:90-100` (`PeerUnresponsiveError`),
`gf:client/peer/connection.go:40-51` (`UnreachableError`).

Recommendation (I):

- `type BuildError struct { Key, Row string; Part string; Riders []string; Err error }`
  in `plan.go`, doc comment saying it is [Plan.Build]'s refusal of one
  endpoint; `Error()` keeps today's exact text (both forms, with "part %s"
  when Part != ""); `Unwrap() error { return e.Err }` so
  `errors.Is(err, cover.ErrEnumValue)` keeps working. Return `*BuildError`
  from both sites (`:404`, `:445`).
- Name `BuildError` (no stutter: `hamapping.BuildError`); no new sentinel
  needed. The host does `errors.As(err, &be)` and drops `be.Key`;
  `Riders` tells it that those entities lose this endpoint too and are
  re-planned without it (I: how the host re-plans riders is the host's
  call, HR-P2-data-35).
- A `changelog.md` entry under `[Unreleased]` (API addition).

## Surprises

- None of the three lift+tilt Types except TiltBlindLift exist: Shutter is
  allowed only on one axis, so a tilting shutter cannot stay "Shutter".
- go-hamapping already called Unknown/Unknown for garage etc. — those
  are the only classes that build with tilt today, as the review found.
- The tilt-only and non-positional-tilt command paths (§4) are broken
  independently of adpt-33: HA refuses open_cover on a SwitchBot Blind
  Tilt, and set_cover_tilt_position on a tilt cover without
  SET_TILT_POSITION. Fixing only Type/EndProductType makes such covers
  appear on the controller with commands that fail.
- HA 2026.10.3 added CoverEntityFeature.SPEED (256).
