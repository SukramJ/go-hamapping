# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Home Assistant component to Matter cluster mapping for go-fabric bridges. Twin of `go-ha2fabric`; both are built on
[go-fabric](https://github.com/SukramJ/go-fabric) and share its project
setup. **State: v0.4.1 (brief R6, the go-fabric v0.15.6 bump).** `CONCEPT.md` is the design,
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
- **Which model does which work** (as go-fabric's CLAUDE.md): analysis,
  planning, anything that becomes a fact about Matter or Home Assistant
  in the repository is the strongest model's; implementation of a
  finished plan with guards behind it may use a cheaper one; a
  generated or formatted artefact a test checks may use the cheapest.
- **Official documents in English**: issues, pull requests, docs, notes.
- **One pull request per wave**, squash-merged; CI is the gate.
- **No attribution lines** in commits or pull request bodies.
- A user-visible change gets a `changelog.md` entry under `[Unreleased]`.
