# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Home Assistant component to Matter cluster mapping for go-fabric bridges. Twin of `go-ha2fabric`; both are built on
[go-fabric](https://github.com/thingfabric/go-fabric) and share its project
setup. **State: v0.7.0 (go-fabric v0.25.0, `thingfabric.dev/go-fabric`).** `CONCEPT.md` is the design,
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

Go 1.27.4, `CGO_ENABLED=0`. Dependencies: go-fabric, go-hamqtt,
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
go-fabric #456; re-cut by the owner on 2026-11-15, mirrored from go-fabric
#529 and openccu-loom #1108, in the owner's words: "Wichtig ist, dass die
Qualität hoch ist, aber Fable gezielt für die Dinge eingesetzt wird, die es
wirklich besser kann"), adapted. The bar is unchanged: a row is a statement
about Matter (go-fabric's generated packages, matter.js's model at the pin)
and about Home Assistant (go-ha-catalog's tables, `home-assistant/core` at
its tag) that a consumer cites later as fact, so it comes from the strongest
model. But the strongest model does only the part that needs it: reading the
reference and judging. The writing, the mechanics and the code review are
Opus.

| Work | Model |
| --- | --- |
| **Coordination** of a wave: starting agents, integrating, guards, pull requests, status | **Opus**, in a fresh session per wave (the state is in the notes and the memory, never in a long context) |
| **Planning and decisions:** the wave plan and its brief; the decisions put to the owner; *which* guard gets built and its bite line; a question to go-fabric; the verdict where go-fabric, matter.js, go-ha-catalog and `home-assistant/core` contradict each other | **Fable**, as a targeted agent |
| **Reading the reference and judging:** what go-fabric's generated packages, matter.js's model at the pin, go-ha-catalog's tables and `home-assistant/core` at its tag say, and whether a difference is a deliberate divergence or a gap — for every row line and its `cites`, `docs/by_design.md`, `CONCEPT.md`, `docs/rows.md`'s prose and every comment that cites a matter.js, go-fabric, go-ha-catalog or Home Assistant line | **Fable** gives the verdict with the lines as a short fact note; **Opus** writes the row, the entry or the comment in the file's form; **Fable** reads the finished text once against its note. Every reference statement comes from Fable |
| **Implementation**, also of the rows (`rows/*.yaml`) from Fable's fact note; the generated form (`make generate`); the round-trip and row tests; a dependency bump with its notices; a `changelog.md` entry, the wave report and docs; rebases and merge conflicts | **Opus** (also what fast mode runs on) — with the Fable read of the next row before it merges where it adds or changes a mapping row or mirrors a reference |
| **Review** of a wave's pull request | **Opus** does the code review — correctness, the generated form, tests, the fixed report shape. **Fable** reads on top, bounded: behaviour against matter.js and Home Assistant at their pins, every added or changed mapping row, and every finding of high or medium severity for confirmation before it reaches an implementer |
| **Citations** | The row loader (`internal/rowfile`) refuses a row without `cites`; **Opus** checks every range (the file, the lines, the construct the sentence names). **Fable** reads every citation that is the reason for behaviour (a row's `cites`, "as Home Assistant does"), every doubtful one and a sample of the rest |
| Work a test or generator checks immediately: `make generate`, `gofumpt`, regenerating the coverage floors, adding a test case to an existing pattern; search and first reading of a source tree (go-ha-catalog, go-fabric's device kit, matter.js's model, `home-assistant/core`, this repo) whose result a stronger model or the owner reads again | **Sonnet**, including as the model of a delegated search subagent |

Sonnet never writes a fact about Matter or Home Assistant into the repo.
Opus does not either: it writes what Fable's fact note states. When in doubt
whether a statement is a reference fact, it is one.

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
- **The strongest model for the part that needs it.** Set by the owner on
  2026-11-15. Fable reads references, judges and plans (the table above);
  it does not coordinate, format, write prose or run mechanics. A Fable
  agent's brief is bounded: the lines to read, the question to answer, a
  fact note back.
- **Few turns in a long session.** A coordinator waits for several agents
  at once and integrates them together; a status message carries new
  information or is not sent; a wave starts in a fresh session.
- **The assistant says when to clear the context.** Set by the owner on
  2026-11-15. It does not wait to be asked: it proposes a fresh session
  when a wave or a pull request is merged; when the context has been
  summarised once and the work that remains is of another kind; or when
  what it still needs fits in a short hand-over and the turns are mostly
  waiting for agents or pipelines. It does NOT propose one while agents of
  the session are running or a decision of the owner is open. The proposal
  comes with the hand-over already written — the state, the open points
  with where each goes, the next step, the lessons — into the memory file
  and the notes (never only into the chat), and with the exact way to go
  on: `/clear`, the model to switch to (`/model`), and the first message
  to send, which names the files to read.
- **The pipeline's guards before the pull request, not after.** The local
  guard set is what the jobs of `.github/workflows/ci.yml` run: `make ci`,
  `make race`, `make cover-check fuzz-smoke`, `make vuln licenses`, and
  `make generate` leaving `rows_gen.go` and `docs/rows.md` unchanged. A red
  pipeline costs a round of coordinator turns.
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
