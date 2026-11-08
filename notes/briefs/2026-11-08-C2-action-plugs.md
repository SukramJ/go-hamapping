# Brief C2 (go-hamapping) — script, scene and automation as plug-in units that always read off (v0.4.5)

Owner decision (2026-11-08): scripts exposed to Assist must become endpoints
(go-ha2fabric web UI: 4 `script.*` entities "Nicht abbildbar", reason "no row
for platform script", `plan.go:228`). Scene and automation share the shape and
come in the same wave. One pull request on `feat/action-plugs`, squash-merged,
auto-merge; release v0.4.5 (the tag is the release). Host follow-up: go-ha2fabric
brief C2 (the adapter's services, fixtures, e2e, release 0.5.6).

Sources, verified 2026-11-08 (cite them in `cites`, do not re-derive):
- Home Assistant 2026.10.3: `script/services.yaml:4-8` (`script.turn_on`,
  target domain script, no fields), `script/__init__.py:606-610` (`is_on` is
  `script.is_running`: on only while it runs); `scene/services.yaml:3-14`
  (`scene.turn_on`, optional `transition`), `scene/__init__.py:104-110`
  (`state` is the last activation's timestamp or None);
  `automation/services.yaml:22-30` (`automation.trigger`, `skip_condition`
  default true), `automation/__init__.py:122-127` (`is_on` means enabled, not
  running). Read from raw files, never run.
- matter.js 862ad75f `packages/model/src/standard/elements/on-off-plug-in-unit.element.ts:27-31`
  (OnOff mandatory, feature LT) — as `rows/button.yaml:10`.
- home-assistant-matter-hub 664575a: `packages/backend/src/matter/endpoints/legacy/script/index.ts:8-14`
  (`PlugOnOffServer({isOn: () => false, turnOn: action "script.turn_on", turnOff: null})`),
  `legacy/scene/index.ts:9-19` (same, "scenes are momentary — `isOn: () => false`
  ensures scenes never appear stuck on"), `legacy/automation/index.ts:8-14`
  (`automation.trigger`); its FAQ (`docs-site/docs/faq.md:127-133`) records the
  earlier on-then-off flip that left Echo devices unresponsive (#423).
- This repo v0.4.4: `rows/button.yaml:5-21` (the shape: OnOffPlugInUnit, `OnOff`
  const false, `handle: "On"` with `method: press`, `raw: {const: PRESS}`),
  `docs/by_design.md` "A button is a plug-in unit, not a switch", `rows_gen.go`
  `Platform` is a plain string (no catalog lookup; `script` and `automation`
  are not MQTT platforms and need none), `rows_test.go` `TestButton`.

## §1 Rows

Three files, `rows/script.yaml`, `rows/scene.yaml`, `rows/automation.yaml`,
each one row `plug`, `when: {}`, `type: OnOffPlugInUnit`, `verdict: mappable`,
`clusters: OnOff` with `attributes: [{attribute: OnOff, const: false}]` and one
command `handle: "On"`:
- script: `method: turn_on`, `raw: {const: RUN}`;
- scene: `method: turn_on`, `raw: {const: ACTIVATE}` (no `transition`);
- automation: `method: trigger` (a new method constant `MethodTrigger`,
  generated), `raw: {const: TRIGGER}`.
The entity's state is not read: a script's `on` means running, a scene's state
is a timestamp, an automation's `on` means enabled — none is what a
controller's On means. No `Off` handler: the kit's server turns itself off
again, as for a button. Each row's `cites` carries the lines above.

## §2 by_design

`docs/by_design.md`, one entry "A script, scene or automation is a plug-in
unit that always reads off": matter-hub does the same since #423 (the three
`index.ts` lines); the reason in one sentence — a controller's On is a
trigger, the entity's own state means something else, so the plug stays off
and never flips.

## §3 Tests, generate, docs

`rows_test.go`: `TestScript`, `TestScene`, `TestAutomation` on the `TestButton`
pattern — type OnOffPlugInUnit, `OnOff` false after a state `on`/timestamp/`on`,
`On` invoke → command with method `turn_on`/`turn_on`/`trigger`, `r.noMore()`.
`make generate` (`rows_gen.go`, `docs/rows.md` table, the method constants).
`changelog.md` `### Added`: the three rows and the host they serve.
`CLAUDE.md`/`README.md` state lines move with the release commit.

## §4 Guards, commits, release

Commits, each green under `make ci`: (1) this brief; (2) rows + generate +
tests; (3) by_design + changelog. Guards in the background with one summary
log: `make ci`, `make race`, `make cover-check`, `make fuzz-smoke` (the rows
changed). Local work only — the owner's session pushes, opens the pull
request, folds in the v0.4.5 release commit and tags.

## §5 Report (hand back, fixed shape)

Per section the commits; guards with HEAD and exit; differences from this
brief; decisions; left out; questions; `by_design` candidates. Under 200
words.
