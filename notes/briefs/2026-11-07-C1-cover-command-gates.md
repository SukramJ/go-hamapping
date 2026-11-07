# Brief C1 (go-hamapping) — cover command rows gated on the entity's features (v0.4.4)

Owner decision (2026-11-07): start issue #18 now as a small wave; one pull
request on `fix/cover-command-gates`, squash-merged, auto-merge; release
v0.4.4 at the end (the tag is the release). Reported in go-ha2fabric #40
(comment 6711638340); host follow-up go-ha2fabric #42.

Sources, verified 2026-11-07 (cite them, do not re-derive):
- Home Assistant 2026.10.3: `cover.set_cover_position` requires
  `CoverEntityFeature.SET_POSITION` (`homeassistant/components/cover/services.py:49-58`),
  `cover.stop_cover` requires `STOP` (`:60-62`); flags OPEN 1, CLOSE 2,
  SET_POSITION 4, STOP 8 (`cover/const.py:44-54`); a call on a referenced
  entity without the feature raises `ServiceNotSupported`
  (`helpers/service.py:750-760`). Read from raw files, never run.
- go-fabric v0.18.0: WindowCovering serves `GoToLiftPercentage` on Lift
  without PositionAwareLift as an optional command
  (`cluster/cover/windowcovering_server.go:144-158`); without position
  awareness its own movement model maps the percentage to open or close
  (`:624-634`, `openOrClose`). `LiftPercent100thsValue` is an integer field
  (`cluster/spec/windowcovering/definition_gen.go:339-345`).
- matter.js db021da0 `packages/node/src/behaviors/window-covering/WindowCoveringServer.ts:576-590`:
  without `positionAwareLift`, `liftPercent100thsValue === 0` → `upOrOpen()`,
  else `downOrClose()`.
- This repo v0.4.3: `rows/cover.yaml:21-25` (features gated), `:34-35`
  (position attributes gated), `:38-51` (commands: StopMotion `:43-44` and
  GoToLiftPercentage `:45-47` ungated, GoToTiltPercentage `:48-51` gated);
  `docs/rows.md:68` (`when` on a command row), `:121-123` (a command without
  a handler row answers the cluster's success and hands nothing out);
  `row.go:236-244` (`Set{All, Any, None}`); the generator renders a numeric
  `if` literal for an integer field (`internal/gen/gen.go:1083-1096`,
  `isInteger` branch) — so `if: {field: LiftPercent100thsValue, in: [0]}`
  is valid; several `handle` entries for one command exist today
  (`rows/climate.yaml:83-111`).

## §1 Rows (`rows/cover.yaml`, the `commands` of `window_covering`)

```yaml
- handle: StopMotion
  when: {features: {any: [STOP]}}
  raw: {from: extra.payload_stop, catalog: true}
- handle: GoToLiftPercentage
  when: {features: {any: [SET_POSITION]}}
  method: set_position
  raw: {field: LiftPercent100thsValue, scale: -0.01, add: 100}
- handle: GoToLiftPercentage           # LF without PA_LF, as matter.js: 0 opens, else closes
  when: {features: {none: [SET_POSITION]}}
  if: {field: LiftPercent100thsValue, in: [0]}
  raw: {from: extra.payload_open, catalog: true}
- handle: GoToLiftPercentage
  when: {features: {none: [SET_POSITION]}}
  if: {field: LiftPercent100thsValue, not_in: [0]}
  raw: {from: extra.payload_close, catalog: true}
```

Each new line cites its source in the row's `cites` (the three sources
above, `file:line`). `GoToTiltPercentage` unchanged. If the generator or the
binding does not accept two `handle` entries of one command with different
`when`s (check `internal/gen/gen.go` `handlers`, `byName`), stop and report
with the lines — do not invent a new row form.

## §2 Tests (`rows_test.go`, the `TestCover` pattern `:359-383`)

- `TestCoverOpenCloseOnly`: `supported_features: 3`; `GoToLiftPercentage 0`
  → command binding payload `OPEN`; `10000` → `CLOSE`; `StopMotion` → no
  command (`r.noMore()`); `UpOrOpen`/`DownOrClose` as before.
- `TestCoverPositionNoStop`: `supported_features: 7`; `GoToLiftPercentage
  2500` → `set_position 75`; `StopMotion` → no command.
- `TestCover` (15) unchanged in its assertions.
- The round trip through go-fabric's in-process controller if the existing
  cover test has one (`roundtrip_test.go`): the OPEN|CLOSE cover answers
  `GoToLiftPercentage` with success.

## §3 Generate, docs, changelog

`make generate` (`rows_gen.go`, the table in `docs/rows.md`; never
hand-edit). `changelog.md` `### Fixed` under `[Unreleased]`: one entry naming
the symptom (HA's `ServiceNotSupported` on `set_cover_position`/`stop_cover`
for a cover without SET_POSITION/STOP) and the mapping (matter.js's open/close
fallback), with go-ha2fabric #40. `docs/by_design.md` if the repo has one and
the open/close fallback is a difference from matter-hub (check; otherwise no
entry).

## §4 Guards, commits, release

Commits, each green under `make ci`: (1) this brief into `notes/briefs/`;
(2) rows + generate + tests; (3) changelog. Guards in the background with a
summary log: `make ci`, `make race`, `make cover-check`, `make fuzz-smoke`
(the row loader is the fuzz target and the rows changed). PR on
`fix/cover-command-gates`, body: symptom, sources, what changed; auto-merge.
Then the release commit as `eea8ace`/`b9592a3` did (`changelog.md`
`## [0.4.4] - 2026-11-07`, state lines in `CLAUDE.md`/`README.md`), same PR
or a second — one PR is fine for a wave this small; tag `v0.4.4` annotated
like `v0.4.3` on the squash commit after CI.

## §5 Report (hand back, fixed shape)

Per section the commits; guards with HEAD and exit; differences from this
brief; decisions; left out; questions; `by_design` candidates. Under 200
words.
