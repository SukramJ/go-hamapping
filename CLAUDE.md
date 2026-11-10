# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Home Assistant component to Matter cluster mapping for go-fabric bridges. Twin of `go-ha2fabric`; both are built on
[go-fabric](https://github.com/SukramJ/go-fabric) and share its project
setup. **State: v0.4.10 (brief C7, go-fabric v0.21.0).** `CONCEPT.md` is the design,
`docs/rows.md` the row format. The table is `rows/*.yaml`; `rows_gen.go`
is generated from it (`make generate`) and never hand-edited.

## Commands

```sh
make ci        # build vet test lint — what the pipeline runs
make generate  # rows/*.yaml -> rows_gen.go and the table of docs/rows.md
make cover-check fuzz-smoke   # the coverage floors and the row loader's fuzz target
make race      # go test -race (CGO_ENABLED=1 for this run only)
make fmt       # gofumpt -w .
```

Go 1.27.3, `CGO_ENABLED=0`. Dependencies: go-fabric, go-hamqtt,
go-ha-catalog, gopkg.in/yaml.v3 (`THIRD-PARTY-NOTICES.md`). Every further
dependency is a decision: Apache-2.0 / MIT / BSD only, and it updates the
notices.

## Rules

- **Naming.** "Matter" is a trademark of the Connectivity Standards
  Alliance and is not part of this project's name, its packages, its
  binaries or its images; it appears only descriptively. See the README.
- **go-fabric is the gold standard** for everything on the Matter side:
  device types, clusters, the facade (`node`, `device`, `contract`, the
  `<cluster>def` packages). This repository never reimplements a
  cluster or reaches below go-fabric's facade: non-test code imports
  `device`, `contract`, `schema`, `eligibility` and the `<cluster>def`
  packages only (`internal/arch`). No Matter or Home Assistant fact of
  its own: device types and clusters from go-fabric's generated packages,
  Home Assistant from go-ha-catalog's tables, each row citing its lines.
- **Which model does which work**: the section below, as go-fabric's
  `CLAUDE.md`.
- **Official documents in English**: issues, pull requests, docs, notes.
- **One pull request per wave**, squash-merged; CI is the gate.
- **No attribution lines** in commits or pull request bodies.
- A user-visible change gets a `changelog.md` entry under `[Unreleased]`.

## Which model does which work

As go-fabric's `CLAUDE.md` (owner, 2026-10-10; the review split 2026-11-08,
go-fabric #456), adapted: a row is a statement about Matter (go-fabric's
generated packages, matter.js's model at the pin) and about Home Assistant
(go-ha-catalog's tables, `home-assistant/core` at its tag) that a consumer
cites later as fact, so the work that produces such statements gets the
strongest model, and the work a guard checks may use a cheaper one.

| Work | Model |
| --- | --- |
| Analysis, planning, briefs, the wave report; every row line and its `cites`; anything written into `docs/by_design.md`, `CONCEPT.md`, `docs/rows.md`'s prose or a comment that cites a matter.js, go-fabric, go-ha-catalog or Home Assistant line; rebases and merge conflicts where upstream may have overtaken the change; a question to go-fabric | **Fable** (the default for this repo) |
| Implementation of a finished brief with guards behind it: a row whose sources the brief has verified, the generated form (`make generate`), the round-trip and row tests in the existing pattern, a dependency bump with its notices, a `changelog.md` entry for a finished diff | **Opus** (also what fast mode runs on) |
| **Review of a wave's pull request, split (owner decision 2026-11-08, go-fabric #456).** The code review — correctness, the generated form, tests, the fixed report shape — is **Opus**. On top of it **Fable** reads, bounded: every new `cites` line against the pinned source, every line that enters `docs/by_design.md` or `CONCEPT.md`, and the brief's decisions against matter.js and Home Assistant — the behavioural gaps a code review does not see. The load-bearing claims of the Opus review are verified before they reach the implementer. | **Opus**, then **Fable** |
| Work a test or generator checks immediately: `make generate`, `gofumpt`, regenerating the coverage floors, adding a test case to an existing pattern; search and first reading of a source tree (go-ha-catalog, go-fabric's device kit, matter.js's model, `home-assistant/core`, this repo) whose result a stronger model or the owner reads again | **Sonnet**, including as the model of a delegated search subagent |

Sonnet never writes a fact about Matter or Home Assistant into the repo.
When in doubt about which row a task is in, it is in the first.

## Token use

As go-fabric's `CLAUDE.md` (owner, 2026-10-14: "Wichtig ist mir eine
effiziente Nutzung der Token"). A wave costs on the order of half a million
to a million tokens per implementing agent before the owner's review; the
rules below are what keeps that from doubling.

- **Read slices, not files.** `sed -n a,bp`, `grep -n` with a tight
  pattern and `head`; a whole file only when the task is the file. Never
  re-read what is already in context; cite the earlier read.
- **Delegate first reads.** A source tree's inventory (go-ha-catalog,
  go-fabric's `device/*_gen.go` and `cluster/spec/*`, matter.js's model at
  the pin, `home-assistant/core` at its tag, this repo) is a Sonnet read
  written to `notes/reads/read-*.md` with `file:line` for every fact and
  no judgement; the owner spot-checks the load-bearing lines and verifies
  every fact before it enters a brief, a row's `cites` or
  `docs/by_design.md` — Sonnet's result is a map, never a citation.
- **Logs by their summary.** Guards (`make ci`, `make race`,
  `make cover-check`, `make fuzz-smoke`, `make generate`) run in the
  background with a summary log (HEAD before and after, one `exit` line
  per guard); read the summary, and open a guard's log only on failure,
  with `grep` for `FAIL`, `DATA RACE`, `_test.go:` and the error strings —
  never `cat`. The same for CI: the job log is fetched once to a file and
  grepped.
- **One turn, every independent call.** Reads, checks and background
  starts that do not depend on each other go out together; a wait is a
  `Monitor` or an `until` loop in the background, not polling by hand.
- **Short artefacts.** A brief (`notes/briefs/<date>-<wave>-<topic>.md`)
  names decisions and sections, not prose; a report
  (`docs/reports/<wave>.md`) follows the fixed shape — per section with
  the commits, guards with HEAD, differences from the brief, decisions,
  left out, questions, `by_design` candidates; a `by_design` entry is one
  paragraph with the citations and one sentence of reasoning; status
  messages to the owner are short, outcome first, in German.
- **Skip what cannot have changed.** No `make fuzz-smoke` when the row
  loader's inputs did not move; no `make generate` for a notes-only
  change; no second guard run for a notes-only commit — say what was
  skipped and why.
- **Edit by script.** Several replacements in several files are one
  scripted call with asserted anchors, not a chain of interactive
  edits; a generated file (`rows_gen.go`, the table in `docs/rows.md`) is
  regenerated, never patched.
- **Agents read what the brief names.** An implementing agent gets the
  file list and the pinned sources in the brief, reads those, and reports
  in the fixed shape; it does not survey the tree. The brief is copied to
  `notes/briefs/` in the wave's first commit. One agent per brief; two
  agents only on disjoint paths.
- **Push, pull request and tag are the session's own calls.** An
  implementing agent works locally — branch, commits, guards — and
  reports; the owner's session reads the diff, pushes, opens the pull
  request with auto-merge and tags after CI. A delegated push, pull
  request or tag is refused by the auto-mode classifier ("Create Public
  Surface", 2026-11-07), and the review above sits between the agent's
  report and the push anyway.
