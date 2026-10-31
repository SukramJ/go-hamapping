# go-hamapping

[![ci](https://github.com/SukramJ/go-hamapping/actions/workflows/ci.yml/badge.svg)](https://github.com/SukramJ/go-hamapping/actions/workflows/ci.yml)
[![codeql](https://github.com/SukramJ/go-hamapping/actions/workflows/codeql.yml/badge.svg)](https://github.com/SukramJ/go-hamapping/actions/workflows/codeql.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)

The table from Home Assistant's component vocabulary (`light`, `switch`,
`cover`, `climate`, `lock`, `fan`, `sensor`, `binary_sensor`, `number`,
`select`, their attributes and capabilities such as `supported_color_modes`,
`hvac_modes` and `device_class`) to a Matter device type with its cluster
declaration, and back from a Matter command to a Home Assistant service
call, as a Go module on [go-fabric](https://github.com/SukramJ/go-fabric)'s
device kit. Two consumers: [`go-ha2fabric`](https://github.com/SukramJ/go-ha2fabric)
and the `go-*2mqtt` daemons that embed go-fabric in process.

Twin of [`go-ha2fabric`](https://github.com/SukramJ/go-ha2fabric); both are built on
[go-fabric](https://github.com/SukramJ/go-fabric) and share its project
setup (pure Go, `CGO_ENABLED=0`, Apache-2.0 with a contributor licence
agreement, the same guards in CI).

> **Status: v0.4.0 (brief R6).** The row format, its generated Go form,
> the guards and the first rows, each with a round trip through go-fabric's
> in-process controller (v0.1.0, brief H1); a stable identity a source
> supplies, the method constants and the table of keys a source fills
> (v0.2.0); a verbatim endpoint name a source supplies (v0.2.1); the
> roadmap rows — air quality, electrical, flow, soil, button, event,
> select, climate fan modes, vacuum, valve, water heater, media player —
> and the format's events, options and otherwise (v0.3.0); host ports in a
> row and the camera (v0.4.0). The design is in
> [`CONCEPT.md`](CONCEPT.md), the rows in [`docs/rows.md`](docs/rows.md).

## Why "Matter" is not in the name

"Matter" and the Matter logo are trademarks of the Connectivity Standards
Alliance. Their use in a product, project or repository name, as on a
device, is reserved to Alliance members and certified products under the
*Connectivity Standards Alliance Trademark and Logo Usage Guidelines and
Terms* (<https://csa-iot.org/resources/governing-documents/>; the terms are
restated in go-fabric's
[`licenses/NOTICE-matter.js.txt`](https://github.com/SukramJ/go-fabric/blob/main/licenses/NOTICE-matter.js.txt):
"Only the Alliance and its members may use Alliance trademarks and logos,
including, without limitation, the Matter trademarks and logos"). This
project is neither, so the mark is not part of its name; the word appears
only descriptively, as in "a bridge for Matter controllers", and never to
suggest certification or endorsement. Nothing built with this project is
certified by the Alliance, and using it does not convey any right to
describe a device, product or service as Matter compliant or certified.
See go-fabric's README, section "Not certified".

## Features

One row per Home Assistant capability ([`docs/rows.md`](docs/rows.md) lists
each with its predicate, clusters, verdict and citations):

- **light**: `onoff` → OnOffLight, `brightness`/`white` → DimmableLight,
  `color_temp` → ColorTemperatureLight (mireds or Kelvin), `hs`/`xy` →
  ExtendedColorLight; an RGB-only light → ExtendedColorLight, Partial.
  On/Off/Toggle, MoveToLevel, MoveToColorTemperature, MoveToHueAndSaturation
  and MoveToColor become the light's JSON command.
- **switch** → OnOffPlugInUnit.
- **sensor**: temperature, humidity, pressure, illuminance, power, voltage,
  current and energy, each in every unit the catalog allows; a battery rides
  on the device's primary endpoint (PowerSource); every other device class
  is Unmappable with its reason.
- **binary_sensor**: door, window, opening, garage door → ContactSensor;
  motion, occupancy, presence → OccupancySensor; moisture →
  WaterLeakDetector; smoke and carbon monoxide → SmokeCoAlarm; no class →
  ContactSensor ([by design](docs/by_design.md)).
- **cover** → WindowCovering (lift, tilt, position-aware by the entity's
  feature flags; the device class sets Type and EndProductType).
- **climate** → Thermostat (heating, cooling, auto from the hvac modes).
- **lock** → DoorLock. **fan** → Fan (percentage, presets, oscillation,
  direction).
- **number, select, button, event, text**: Unmappable in v0.1.0, with the
  reason.

## Quickstart

A daemon that holds go-hamqtt entities maps them, builds the devices and
adds them to its go-fabric node; the node, its configuration and its
lifecycle are go-fabric's
([embed in a daemon](https://github.com/SukramJ/go-fabric/blob/main/docs/guides/embed-in-a-daemon.md)):

```go
plan, verdicts := hamapping.Map(dev, []model.Entity{lamp}) // lamp: a go-hamqtt light entity
devices, err := plan.Build(ctx)
if err != nil {
	return err
}
for _, d := range devices {
	if err := n.Aggregator().Add(ctx, d); err != nil { // n: the daemon's *node.Node
		return err
	}
}
plan.OnChange(func(c model.Command) { commands <- c }) // to the daemon's model.Commander
plan.Apply(ctx, lamp.Key(), state)                    // each state the daemon reads
```

`verdicts` says for each entity what Matter makes of it, with a reason an
operator can read for a Partial or Unmappable one. A source whose entity
key can change (a Home Assistant entity id the user renames) passes the
identity that survives the rename, so the endpoint keeps its number:

```go
plan, _ := hamapping.Map(dev, entities, hamapping.Options{
	StableKey: func(e model.Entity) string { return registryID[e.Key()] },
})
```

A command handler compares `model.Command.Method` with the generated
constants (`hamapping.MethodSetPosition`, …), not with literals.

A cluster a row marks `ports: host` (the camera's stream and WebRTC
clusters) runs on ports the consumer implements: `Options.Ports` returns
the host object for an entity, which implements go-fabric's port
interfaces of those clusters. Without it, or with a host that lacks a
port, the entity gets no endpoint and a verdict that says so:

```go
plan, _ := hamapping.Map(dev, entities, hamapping.Options{
	Ports: func(e model.Entity) any { return relays[e.Key()] }, // nil: no ports
})
```

## Configuration

None: the table is the module's. A daemon states what its entities
support in `model.Description.Extra` under the catalog's key names
([`docs/rows.md`](docs/rows.md#the-format)); which `state.<key>` and
`extra.<key>` names each platform's rows read, and which have a catalog
default, is the generated table in
[`docs/rows.md`](docs/rows.md#the-keys-a-source-fills).

## Development

```sh
make build     # go build ./...
make test      # go test -count=1 ./...
make race      # go test -race (CGO_ENABLED=1 for this run only)
make lint      # gofumpt + golangci-lint + go mod tidy -diff
make generate  # rows/*.yaml -> rows_gen.go and the table of docs/rows.md
make cover-check fuzz-smoke   # coverage floors, the row loader's fuzz target
make ci        # build vet test lint
```

The guards, the tool versions and the workflow files follow the sibling
repositories.

## Acknowledgements

Built on [go-fabric](https://github.com/SukramJ/go-fabric), a semantic
port of [matter.js](https://github.com/matter-js/matter.js) (Apache-2.0),
and on the [Home Assistant](https://www.home-assistant.io/) APIs. The
pattern of a thin client between Home Assistant and a Matter process is
the one Home Assistant uses for its own Matter support
([matterjs-server](https://github.com/matter-js/matterjs-server)).

## License

Apache-2.0, see [LICENSE](LICENSE). Contributions are accepted under the
[contributor licence agreement](CLA.md); the use of AI tools is governed by
[`AI_POLICY.md`](AI_POLICY.md).
