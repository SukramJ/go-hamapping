# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- go-fabric `v0.15.6` (was `v0.15.5`): `camera.ErrPayloadTooLarge` in
  `cluster/camera` and a port's `contract.StatusError` answering its own
  status instead of FAILURE (go-fabric #356, go-ha2fabric's two
  questions); test fixes.

## [0.4.0] - 2026-10-31

Brief R6 (go-ha2fabric `docs/roadmap.md` §R6): host ports in a row and the
camera.

### Added

- The row format's `ports: host` on a cluster: the kit's server runs on
  ports the consumer supplies; the generator refuses it on a cluster
  without a port materializer (WebRtcTransportProvider,
  CameraAvStreamManagement in this release).
- `Options.Ports func(model.Entity) any`: the host object whose
  interfaces those clusters run on, asked once per entity. Without it, or
  with a host that lacks a port, `Map` builds no endpoint and the verdict
  is Unmappable: "<row>: the consumer serves no host ports for <cluster>".
- camera: a Camera for a camera with the STREAM feature
  (`camera/camera`), CameraAvStreamManagement (VDO, ADO, SNP) and
  WebRtcTransportProvider on the host's ports, the capabilities nominal
  after connectedhomeip's camera app; a camera without STREAM is
  Unmappable (`camera/unmapped`).

### Changed

- go-fabric `v0.15.5`: the camera's signals to the controller's requestor
  (go-fabric #351); the notices name the pinned version again.

## [0.3.0] - 2026-10-31

The roadmap rows R1–R5 (go-ha2fabric `docs/roadmap.md`), brief R1–R5:
sensors, buttons, events, selects, climate, vacuum, valve, water heater,
media player, and the row format's `events`, `options` and `otherwise`.

### Added

- The row format's `events` on a cluster: events emitted through the
  cluster's typed server (`<cluster>def.Server.Emit<Event>`) when a value
  of the entity changes; `Patch.Events` reports them.
- The row format's `options` on a cluster: a list of the entity
  (`desc.options` or `extra.<key>`) a materializer turns into the cluster's
  attributes and command — ModeSelect's modes.
  `OptionList` is the runtime form, `Row.Clusters[].Options`.
- select: a ModeSelect from the entity's options (`select/mode_select`).
- sensor: carbon_dioxide, carbon_monoxide, pm1, pm25, pm10,
  volatile_organic_compounds(_parts), nitrogen_dioxide, ozone and radon
  as an AirQualitySensor with the class's concentration cluster; a class
  with ratio and mass units keeps the unit's family on the wire
  (`MeasurementUnit` by the entity's unit), no molar mass is assumed.
- sensor: apparent_power, reactive_power, frequency and power_factor as an
  ElectricalSensor (ApparentPower mVA, ReactivePower mVAR, Frequency mHz,
  PowerFactor in 1/100 %; a power factor without a unit is a ratio).
- sensor: volume_flow_rate as a FlowSensor (0.1 m³/h), moisture as a
  SoilSensor; binary_sensor: cold as a WaterFreezeDetector.
- button: an OnOffPlugInUnit whose On presses the button (`press`), its
  OnOff always off.
- event: a GenericSwitch (MomentarySwitch, MomentarySwitchRelease) whose
  every new event is a short press (InitialPress, ShortRelease); partial
  until long and multi presses are read from the event type.
- climate: a RoomAirConditioner for an entity with FAN_MODE, TURN_ON and
  TURN_OFF — OnOff (`turn_on`, `turn_off`), the thermostat's Thermostat,
  FanControl from `fan_mode` (writes of FanMode and of PercentSetting by
  bands go to `fan_mode`) and RelativeHumidityMeasurement from
  `current_humidity`.
- vacuum: a RoboticVacuumCleaner for an entity with START — RvcRunMode
  Idle/Cleaning (`run_mode`), RvcOperationalState from the activity
  (`pause`, `resume`, `go_home`), on the clusters' generated servers.
- valve: a WaterValve — CurrentState and TargetState from the state,
  Open and Close to `open_valve` and `close_valve`, on the generated
  server; a position is a caveat.
- water_heater: a heating-only Thermostat — SystemMode Off for `off`, Heat
  for every other operation mode (writes to `operation_mode` with `off` or
  `on`), the setpoint to `temperature`, the limits from min_temp and
  max_temp.
- media_player: a tv is a BasicVideoPlayer (OnOff, MediaPlayback with
  `media_play`, `media_pause`, `media_stop`, KeypadInput's Power and volume
  keys to `key`); every other player a Speaker — OnOff from the power
  (`turn_on`, `turn_off`) or, without it, from the mute (`volume_mute`),
  LevelControl from `volume_level` (writes to `volume_level`, 0 to 254).
- The row format's `otherwise` on a handled command: the handler's
  response when none of its commands' `if` holds (RvcRunMode's
  UnsupportedMode, KeypadInput's UnsupportedKey).
- Method constants for the new methods: `MethodPress`, `MethodSelectOption`,
  `MethodTurnOn`, `MethodTurnOff`, `MethodFanMode`, `MethodRunMode`,
  `MethodPause`, `MethodResume`, `MethodGoHome`, `MethodOpenValve`,
  `MethodCloseValve`, `MethodOperationMode`, `MethodVolumeMute`,
  `MethodVolumeLevel`, `MethodMediaPlay`, `MethodMediaPause`,
  `MethodMediaStop`, `MethodKey`.
- `docs/by_design.md`: a switch is a plug-in unit, a moisture sensor a
  SoilSensor, a button a plug-in unit, a select's mode its option's index.

### Changed

- go-fabric v0.15.4, go-hamqtt v0.39.0.
- The unmappable verdicts of `sensor/no_device_type` and
  `binary_sensor/no_device_type` no longer say "in H1" and list only the
  classes no row maps; `number/unmapped` no longer says "in H1".
- `climate/thermostat`'s caveats no longer say "in H1"; the preset caveat
  names go-fabric's setpoint requirement, the fan caveat the missing
  turn_on and turn_off.
- `TestSensorUnitsAreTheCatalogs` holds the catalog's null unit as the
  units key "" instead of dropping it.

### Note for consumers

- go-ha2fabric translates the new methods into Home Assistant service
  calls: `press`, `select_option` (the option), `turn_on`/`turn_off`,
  `fan_mode` (the fan mode's name), `run_mode` (`0` idle, `1` cleaning),
  `pause`, `resume`, `go_home`, `open_valve`, `close_valve`,
  `operation_mode` (`off` or `on`), `volume_mute` (`true`/`false`),
  `volume_level` (0 to 254), `media_play`, `media_pause`, `media_stop`,
  `key` (the CEC key code: 64 power, 65 volume up, 66 volume down, 67 mute).

## [0.2.1] - 2026-10-26

The verbatim name of go-ha2fabric's H3 report (Q3), brief F1.

### Added

- `Options.Name func(model.Entity) (name string, verbatim bool)`: when it
  returns a non-empty name with `verbatim` true, that name is the
  endpoint's NodeLabel as given — without the device name in front, cut to
  32 bytes the way `device.ComposeNodeLabel` cuts — because a name a user
  typed in a UI is final. Nil, an empty name or `verbatim` false keeps the
  composed label (device name and entity name).

### Changed

- go-fabric `v0.13.3` (was `v0.12.0`); the generated rows are unchanged.
  None of go-fabric's 0.13 breaks reaches this module: it calls no
  `MarkComplete` and implements no `thermo.Sink`.

## [0.2.0] - 2026-10-25

The follow-ups of go-ha2fabric's H2 report (Q3–Q5), brief H3.

### Added

- `hamapping.Options` and `Map(dev, entities, opts ...Options)`:
  `Options.StableKey func(model.Entity) string` supplies the identity of an
  entity that survives a change of its key (a registry id). The endpoint's
  stable key — the one go-fabric numbers the endpoint by — is the device
  UID, `/` and this identity; the entity's `Key()` stays the address of the
  verdict, the endpoint, `Plan.Apply` and `Plan.DeviceFor`. Nil or an empty
  result keeps v0.1.0's stable key (`uid/Key()`). go-hamqtt's model carries
  no entity identity besides `Key()`, so the source passes it.
- Method constants (`MethodDirection`, `MethodMode`, `MethodOscillation`,
  `MethodPercentage`, `MethodPresetMode`, `MethodSetPosition`,
  `MethodTemperature`, `MethodTemperatureHigh`, `MethodTemperatureLow`,
  `MethodTilt`), generated from the rows' `method` names by
  `internal/gen`; a test holds them equal to the rows.
- `docs/rows.md` §The keys a source fills: per platform, every `state`,
  `state.<key>` and `extra.<key>` the rows read, by which rows, as
  predicate or value, and whether the catalog states a default — generated
  from the rows and held current by `TestGeneratedRowsAreCurrent`.

### Changed

- go-fabric `v0.12.0` (was `v0.11.0`); the generated rows are unchanged.

### Note for consumers

- A consumer that starts passing `Options.StableKey` changes the stable
  key, and with it the endpoint number, of every entity once. go-ha2fabric
  moves in its v0.2.0 (the registry entry id), so a controller re-learns
  its bridged endpoints once at that upgrade.

## [0.1.0] - 2026-10-25

The first release (brief H1): the row format, its generated Go form, the
guards and the first rows, each with a round trip through go-fabric's
in-process controller.

### Added

- `hamapping.Map(dev, entities)`: a verdict per go-hamqtt entity
  (eligibility's Mappable, Partial or Unmappable, with a reason) and a
  `Plan` with one endpoint per mapped entity; `Plan.Build` constructs the
  go-fabric kit devices, `Plan.Apply` carries a `model.State` onto them as
  typed `Set` calls and `SetReachable`, `Plan.OnChange` hands a
  controller's command or write back as a `model.Command`.
- The table as data: `rows/<platform>.yaml`, read by `internal/rowfile`
  and generated into `rows_gen.go` and the row table of `docs/rows.md` by
  `internal/gen` (`make generate`); `docs/rows.md` describes the format.
- Rows: `light` (OnOffLight, DimmableLight, ColorTemperatureLight,
  ExtendedColorLight; an RGB-only light Partial), `switch`
  (OnOffPlugInUnit), `sensor` (temperature, humidity, pressure and
  atmospheric pressure, illuminance, power, voltage, current, energy;
  battery riding on the device's primary endpoint, Partial), `binary_sensor`
  (ContactSensor, OccupancySensor, WaterLeakDetector, SmokeCoAlarm for smoke
  and carbon monoxide, ContactSensor for a sensor without device class), `cover` (WindowCovering),
  `climate` (Thermostat), `lock` (DoorLock), `fan` (Fan); `number`,
  `select`, `button`, `event` and `text` Unmappable with their reasons.
- Guards: the generated rows are current; every row names go-fabric's
  device types and clusters; every sensor and binary_sensor device class of
  the catalog has exactly one row; a mapped sensor converts exactly the
  catalog's units; predicates name only the catalog's feature flags and
  keys; the import guard (`internal/arch`); coverage floors
  (`make cover-check`) and the fuzz target of the row loader
  (`make fuzz-smoke`), both now jobs of the CI workflow.
