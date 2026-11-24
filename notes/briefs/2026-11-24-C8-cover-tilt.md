# Brief C8: covers with tilt build and work; select bounds (v0.7.0)

Source: go-ha2fabric's review phase 2 (`go-ha2fabric/notes/review/2026-11-23/phase-2-X2.md`,
rows HR-P2-adpt-33, -adpt-34, -adpt-36, -data-35; decisions 35, 40 and 47
in its `decisions.md`). Fact note: `notes/reads/fact-2026-11-24-cover-tilt.md`
(the verdict; by Opus under the owner's rule of 2026-11-24 "Opus instead of
Fable until further notice"). Pins: go-fabric v0.25.0
(`/home/markus/go/pkg/mod/thingfabric.dev/go-fabric@v0.25.0`),
home-assistant/core 2026.10.3, matter-hub `664575a`, matter.js as in
`CLAUDE.md`.

Branch `fix/cover-tilt` from `origin/main` (`d570fdd`). First commit: this
brief and the fact note.

## Work

| # | What | Source |
| --- | --- | --- |
| 1 | `rows/cover.yaml`: Type and EndProductType by device class **and** tilt (LF+TL → TiltBlindLift with InteriorBlind / SheerShade / ExteriorVenetianBlind / VerticalBlindStripCurtain / Unknown per the note's table; LF unchanged; garage, gate, door, damper, window unchanged). FeatureMap unchanged (Lift always) | note Q1, Q2 |
| 2 | Tilt-only covers (no OPEN/CLOSE, no SET_POSITION; e.g. features 240): UpOrOpen, DownOrClose, GoToLiftPercentage go to the tilt axis, as matter-hub does | note "surprising"; decision 47 |
| 3 | GoToTiltPercentage without SET_TILT_POSITION: open_tilt for fully open, close_tilt otherwise (as the lift row without SET_POSITION); new methods `open_tilt`, `close_tilt` (a host translates them to `open_cover_tilt` / `close_cover_tilt`) | note "surprising"; decision 47 |
| 4 | `maxModes = 255` | note Q3, HR-P2-adpt-34 |
| 5 | ModeSelect labels that repeat after the 64-byte cut are numbered " (n)", cut to fit | note Q4, HR-P2-adpt-36 |
| 6 | A typed build error: `BuildError{Key, Row, …, Err}`, `Error()` keeps today's text, `Unwrap()`; returned where `Build` fails today | note Q5, HR-P2-data-35 |
| 7 | A standing guard: a row test over every cover device class × feature set (and a 255/256-option select) that what `Map` calls mappable **builds** against go-fabric | HR-P2-adpt-33 |

`make generate`; `docs/rows.md` regenerated; `docs/by_design.md` rows for
the divergences from matter-hub the note lists (one paragraph each, its
citations); `changelog.md` `[Unreleased]` (Added: methods `open_tilt`,
`close_tilt`, `BuildError`; Fixed: tilt covers, 255 modes, repeated
labels). The new methods are a contract change for hosts: say so in the
changelog.

## Rules

No sub-agents; no push, pull request or tag (the coordinator's); never
`git stash`; `PATH=$HOME/go/bin:$PATH`; red test first per row; the local
guard set of `CLAUDE.md` (`make ci`, `make race`, `make cover-check
fuzz-smoke`, `make vuln licenses`, `make generate` leaving the tree
unchanged) green at the end. Report `notes/briefs/2026-11-24-C8-report.md`
(commits, guards with HEAD, per row done / not done, the host's to-do for
go-ha2fabric's `internal/adapt`, questions), committed last. Stop when it
is written.
