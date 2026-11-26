# go-hamapping: concept

Status: F1 delivered (v0.2.1, 2026-10-26): `Options.Name`; H3 delivered (v0.2.0, 2026-10-25): `Options.StableKey`, the method
constants and the generated key table; H1 (v0.1.0, 2026-10-25): the row
format, the generated Go form, the guards and the rows of Sequence step 2 with their round
trips; see [`docs/rows.md`](docs/rows.md) and
[`docs/reports/H1.md`](docs/reports/H1.md). Written 2026-10-24 from go-fabric's concept
v6 part C (2026-10-20) and the owner's decisions of 2026-10-24 (go-fabric
concept v8: `go-fabric-server`). Changes to this document follow those
decisions; the go-fabric roadmap names the milestone in which work starts.

## Purpose

One table and its runtime forms: a Home Assistant component (`light`,
`switch`, `cover`, `climate`, `lock`, `fan`, `sensor`, `binary_sensor`,
`number`, `select`, …) with its attributes and capabilities
(`supported_color_modes`, `hvac_modes`, `device_class`, `supported_features`)
to a device type of go-fabric's device kit with its cluster declaration,
and back: a Matter command or write to a Home Assistant service call or
an MQTT command. The vocabulary is the one `go-hamqtt` already carries for
the `go-*2mqtt` daemons (`model.Entity`, `model.Device`, the platform
catalogue of `go-ha-catalog`), so one table serves every consumer.

## Consumers

- The `go-*2mqtt` daemons (daikin, homeconnect, mtec, unifi, zendure)
  embed go-fabric in process: the table's **Go form** turns an entity
  into a typed declaration (`device.<Type>` with `<cluster>def.Declare`)
  and a command into a handler call.
- `go-ha2fabric` embeds go-fabric in process too (owner decision
  2026-10-24, evening) and maps a Home Assistant registry entity onto
  go-hamqtt's vocabulary with a small adapter, then applies the same Go
  form. There is no JSON form of the table: no consumer runs outside Go.

## Boundary

The Go form imports `node`, `device`, `contract`, `schema`, `eligibility`
and the `<cluster>def` packages of go-fabric and nothing below them: the
module is the first consumer at which go-fabric's facade measurement
holds without legacy. Which device types an ecosystem accepts is
`eligibility`'s knowledge; the table chooses the device type by it, the
consumer does not.

## Form of a row

A row names the component, the capability predicate (which attributes or
feature flags must be present), the device type, the clusters with the
attribute mapping (Home Assistant attribute, unit and scale to Matter
attribute), and the command mapping (Matter command and fields to service
and data). A row is data; the Go form is generated from it, as go-fabric
generates its cluster definitions from the matter.js snapshot, and a
guard holds the two forms equal.

## Tests

Every row has a round trip in go-fabric's `fabrictest`: an entity state
becomes an endpoint whose attributes a controller reads back, a
controller's command becomes the expected service call. The device-type
validator of go-fabric judges every generated declaration.

## Sequence

1. go-fabric closes milestone 4 (`v0.11.0`); its cookbook wave has shown
   one device type end to end on the kit from a `go-*2mqtt` daemon.
2. The row format and the first rows (`light`, `switch`, `sensor`,
   `binary_sensor`, `cover`, `climate`, `lock`, `fan`), each with its
   round trip.
3. The rows go-ha2fabric needs first, with the registry adapter's
   expectations written into the row format.

## Open decisions

- Decided in H1: the row format is YAML in `rows/`, the Go form is
  generated (`internal/gen`), and `TestGeneratedRowsAreCurrent` holds the
  two equal.
- Decided in H1: a daemon states what its entity supports in
  `Description.Extra` under the catalog's key names
  (`supported_color_modes`, `supported_features` as the catalog's flag bits
  or names, `modes`, `preset_modes`); a row's predicate and its feature
  conditions read them, so a cover without tilt features declares no Tilt
  and a light without colour is a DimmableLight. Every cover declares
  PositionAwareLift, as home-assistant-matter-hub does, and every tilting
  cover PositionAwareTilt; without SET_POSITION its lift position is the
  state's (`open`, `closed`) unless the state carries `current_position`,
  and a GoToLiftPercentage or GoToTiltPercentage between fully open and
  fully closed of an axis without a position service is refused (#44).
- Decided in R1–R5: a cluster may carry `events` — events emitted through
  the typed server's `Emit<Event>` when a value of the entity changes
  (Home Assistant's event entity as a GenericSwitch press). The first
  value only sets the baseline.
- Decided in R1–R5: a cluster may carry `options` — a list of the entity
  (a select's options) that a cluster-specific Go materializer
  (`options.go`) turns into the cluster's attributes and command. A
  materializer is code, not data, because the list's length is the
  entity's; ModeSelect has one. Thermostat presets were planned as the
  second and wait for go-fabric: its Thermostat server requires setpoints
  on every preset (`cluster/thermo/presets.go:246-257` at v0.15.4), which a
  Home Assistant preset mode does not carry.
- Decided in R1–R5: a handled command may carry `otherwise`, the
  handler's response when none of its commands' `if` holds (RvcRunMode's
  UnsupportedMode, KeypadInput's UnsupportedKey).
- Open: the import boundary. H1 imports `device`, `contract`, `schema`,
  `eligibility` and the `<cluster>def` packages, not `node` (brief H1 hard
  rule 1); a daemon adds the built devices to its own node.

## References

- go-fabric: [README](https://github.com/thingfabric/go-fabric#readme),
  [roadmap](https://github.com/thingfabric/go-fabric/blob/main/docs/roadmap.md),
  [concept v6](https://github.com/thingfabric/go-fabric/blob/main/docs/history/concept-v6-embedding-layer.md)
  part C (the origin of this document),
  [concept v7](https://github.com/thingfabric/go-fabric/blob/main/docs/concept-v7-operation-and-packaging.md)
  decisions 0.4, 0.6, 0.9.
- go-hamqtt and go-ha-catalog: the Home Assistant vocabulary the table
  reads.

## Name

`go-hamapping` names what the module is: the mapping of Home Assistant's
vocabulary onto go-fabric's device types and clusters, and back. It sits
with `go-hamqtt` and `go-ha-catalog` (go-ha + noun). The repository was
created as `go-hafabric` on 2026-10-24 and renamed the same day because
that name sat too close to `go-ha2fabric`, the bridge; GitHub redirects the
old name.
