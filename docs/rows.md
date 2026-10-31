# The rows

The table maps a Home Assistant entity, in go-hamqtt's vocabulary
(`model.Entity`, its `model.Description`, its `model.State`), onto a device
type of go-fabric's device kit with its cluster declarations, and a
controller's command or write back onto a `model.Command`. It is data:
one file per platform under [`rows/`](../rows), read by
[`internal/rowfile`](../internal/rowfile) and generated into
[`rows_gen.go`](../rows_gen.go) by [`internal/gen`](../internal/gen)
(`make generate`). `TestGeneratedRowsAreCurrent` fails when the YAML and
the Go form differ, and when the table at the end of this page is stale.

## How a row is chosen

`hamapping.Map(dev, entities)` tries the rows of an entity's platform in
file order; the first whose `when` holds decides. A platform without a row
file, or an entity no row matches, is Unmappable with a reason that names
the platform and device class. A row's verdict is `mappable`,
`partial: <reason>` or `unmappable: <reason>`; a mappable row's `caveats`
turn it Partial for the entities they match. `{device_class}` in a reason
is the entity's device class. A unit the row cannot convert makes the
entity Unmappable, naming the unit.

A row with `ride: primary` does not get an endpoint of its own when the
device has a primary endpoint (the first mapped entity whose row does not
ride): its `ride_measurement` (H1: the battery) is declared on the primary
as go-fabric's measurement source (`device.Measurement`). Without a primary
it is an endpoint of its own type.

## The format

```yaml
platform: sensor                 # the Home Assistant platform
cites: [...]                     # sources every row of the file reads
rows:
  - id: temperature              # the row id is <platform>/<id>
    when:                        # every key set must hold
      device_class: [temperature]   # "none": an entity without one
      unit: [...]
      state_class: [...]
      features: {any: [SET_POSITION]}   # the catalog's feature flags (features.json),
                                        # from Extra["supported_features"]: a bit mask or names
      extra:                     # Description.Extra under the catalog's key names
        supported_color_modes: {any: [hs, xy], none: [...], all: [...]}
        color_temp_kelvin: {true: true}     # a boolean key
        preset_modes: {present: true}       # the entity states the key
      options: true              # the entity carries options
    type: TemperatureSensor      # go-fabric's device.<Type>Type
    verdict: mappable            # | "partial: <reason>" | "unmappable: <reason>"
    caveats: [{when: {...}, reason: "..."}]
    cites: [...]                 # the matter.js element lines, the catalog lines
    parts:                       # component devices the type requires
      - type: PowerSource
        clusters: [...]          # features and constant initial values only
    clusters:
      - cluster: TemperatureMeasurement   # go-fabric's <cluster>def name
        features: [Name, {name: Name, when: {...}}]   # <cluster>def.Feature<Name>
        initial:                 # Declare.Initial: a const, a literal or a value
          - {attribute: Order, const: 0}  # of the description (device_class, extra.<key>)
        start_null: [MeasuredValue]       # attributes that start null (unknown)
        attributes:              # the state mapping Plan.Apply sets
          - attribute: MeasuredValue      # or Attribute.Field of a structure
            from: state                   # the value (below)
            units: {"°C": {scale: 100}}   # the conversion (below)
        commands:                # the command mapping Plan.OnChange hands out
          - handle: LockDoor     # | write: [Attr, ...] | change: [Attr, ...]
            if: {field: SystemMode, in: [SystemModeAuto]}   # or not_in
            when: {...}          # entities the command applies to
            method: set_position # the entity's named method; empty: its command binding
            raw: {...}           # the payload as one value, or
            json: {state: {...}, color.h: {...}}   # a JSON object by dotted key
            derive: rgb_from_xy  # keys computed from several fields
            otherwise: {Status: ModeChangeStatusUnsupportedMode}   # the response when no if holds
        events:                  # events Plan.Apply emits when a value changes
          - on: state            # state, state.<key> or extra.<key>
            emit:                # in order, through Server.Emit<Event>
              - {event: InitialPress, literal: {NewPosition: 1}}
        options:                 # a list a materializer serves (ModeSelect)
          list: desc.options     # or extra.<key>: a list of strings
          current: state         # the value whose entry is the current one
          method: select_option  # a choice's method; the payload is the string
```

**Values.** `from` reads the entity: `state` (the primary state value; for a
map state its `state` key), `state.<key>` (a key of a map state, dotted for
nested keys, else of `State.Extra`), `extra.<key>` (a key of
`Description.Extra`; with `catalog: true`, or in a predicate, the catalog's
default for the key when the entity states none), `device_class`, `unit`,
`desc.min`, `desc.max`, `desc.step`. A command value reads `field` (the
handled request's field, or the cluster's state field for `write` and
`change`) or a `const`. A number converts as
`y = func(x / per + offset) * scale + add`, times `times`, clamped to
`[min, max]`; `func` is `log10`, `pow10` or `reciprocal`; `units` replaces
offset, scale and add by the entity's unit (`default_unit` for an entity
without one); `digits` rounds a payload number. `enum` maps the entity's
string (lower case; a boolean state reads `on`/`off`) to a constant of the
cluster package, a bool or an integer for an attribute, and a constant (or
`true`/`false`) to the entity's string for a command value. An attribute
whose value is no number (`unknown`, `unavailable`, null) is set null where
it is nullable.

**Commands.** `handle` installs the cluster's typed command handler
(`Declare.Handle<Command>`): the kit's server does not run the command, the
entity does, and its state comes back through `Apply`. `write` installs the
write veto (`Declare.OnWrite`), which accepts the write and hands it out.
`change` hands out a change the kit's server made running a command (the
light's On, MoveToLevel, MoveToColor), from the server's state at the
change; `unless: [Attr]` skips a change that also touches one of these
attributes (the light's colour-mode switch, which converts the current
colour before the controller's target is set). Every command reaches
`Plan.OnChange`'s function inline, in the controller's request. A command with an empty `method` goes to the entity's command
binding (`model.Command.Role` `command`, its `Slot`); one with a method to
the named method (the MQTT `<method>_command_topic` of the platform).
Every method a row names is an exported constant of the package
(`hamapping.MethodPresetMode` for `preset_mode`), generated with the rows,
so a consumer's command handler compares against the constant.
`otherwise` (a `handle` with an `if` only) is the handler's response, by
field name of its response structure, when the `if` of none of the
handler's commands holds: the command is answered with the cluster's own
status and nothing is handed out. Without it a handler answers the zero
response, the cluster's success.

**Events.** `events` emit events of the cluster through its typed server
(`<cluster>def.Server.Emit<Event>`, on the device's endpoint number) when
the value `on` names changes from a known value to a different non-empty
one: Home Assistant's event entity, whose state is the timestamp of its
last event, becomes a press. The first value after the plan is built, or
after the entity was unavailable or its value `unknown`, only sets the
baseline and emits nothing. `literal` gives the event's data by field name
of the event structure in the `<cluster>def` package; the generator refuses
an event or a field the package does not have. `Plan.Apply` reports the
events it emitted in `Patch.Events`. The table names them as
`[on <value>: <events>]` after the cluster.

**Options.** `options` hand a list of the entity to a cluster-specific
materializer of the root package (`options.go`), which turns it into the
cluster's attributes and its command; the generator refuses `options` on a
cluster without one. `list` is `desc.options` (`Description.Options`' codes,
in order) or `extra.<key>` (a list of strings in `Description.Extra`);
`current` is the value whose entry is the current one; a controller's
choice is a `model.Command` of `method` with the chosen string as the
payload. One materializer in this wave: **ModeSelect** — one
`ModeOptionStruct` per entry (the label the entry, the mode its 0-based
index), `Description` the entity's name (`Select` without one),
`StandardNamespace` null, `CurrentMode` the current entry's index (0 when
the state names none); `ChangeToMode` for a known mode delivers the entry,
for another answers INVALID_COMMAND; served on the generated server
(`Declare.HandleChangeToMode`). A select without options is Unmappable.
The table names the
options as `[options <list> at <current> to <method>]`.

## The rows

<!-- rows:begin (generated by internal/gen; do not edit) -->

| Row | Predicate | Device type | Clusters | Verdict | Cites |
| --- | --- | --- | --- | --- | --- |
| `binary_sensor/contact` | device_class door, window, opening, garage_door | ContactSensor | BooleanState | mappable | matter.js packages/model/src/standard/elements/contact-sensor.element.ts:19 @862ad75f (BooleanState mandatory); matter.js packages/model/src/standard/resources/boolean-state.resource.ts:26-30 @862ad75f (Contact Sensor - FALSE=open or no contact, TRUE=closed or contact) |
| `binary_sensor/motion` | device_class motion | OccupancySensor | OccupancySensing (PassiveInfrared) | mappable | matter.js packages/model/src/standard/elements/occupancy-sensor.element.ts:20 @862ad75f (OccupancySensing mandatory); matter.js packages/model/src/standard/resources/occupancy-sensing.resource.ts:45 @862ad75f (Occupancy, bit Occupied); matter.js packages/node/src/behaviors/occupancy-sensing/OccupancySensingServer.ts:27-66 @862ad75f (OccupancySensorType and its bitmap follow the PIR, ultrasonic and physical-contact features) |
| `binary_sensor/occupancy` | device_class occupancy, presence | OccupancySensor | OccupancySensing (Other) | mappable | matter.js packages/model/src/standard/elements/occupancy-sensor.element.ts:20 @862ad75f (OccupancySensing mandatory); matter.js packages/model/src/standard/resources/occupancy-sensing.resource.ts:45 @862ad75f (Occupancy, bit Occupied); matter.js packages/node/src/behaviors/occupancy-sensing/OccupancySensingServer.ts:27-66 @862ad75f (OccupancySensorType and its bitmap follow the PIR, ultrasonic and physical-contact features) |
| `binary_sensor/water_leak` | device_class moisture | WaterLeakDetector | BooleanState | mappable | matter.js packages/model/src/standard/elements/water-leak-detector.element.ts:20-22 @862ad75f (BooleanState mandatory, CHGEVENT) |
| `binary_sensor/water_freeze` | device_class cold | WaterFreezeDetector | BooleanState | mappable | matter.js packages/model/src/standard/elements/water-freeze-detector.element.ts:19-23 @862ad75f (BooleanState mandatory, CHGEVENT); home-assistant/core homeassistant/components/binary_sensor/const.py:23-24 @2026.10.3 (cold, on means cold) |
| `binary_sensor/smoke` | device_class smoke | SmokeCoAlarm | SmokeCoAlarm (SmokeAlarm); part PowerSource | mappable | matter.js packages/model/src/standard/elements/smoke-co-alarm-device.element.ts:24,28 @862ad75f (SmokeCoAlarm mandatory, PowerSource device min 1); matter.js packages/model/src/standard/resources/smoke-co-alarm-cluster.resource.ts:42 @862ad75f (SmokeState) |
| `binary_sensor/carbon_monoxide` | device_class carbon_monoxide | SmokeCoAlarm | SmokeCoAlarm (CoAlarm); part PowerSource | mappable | matter.js packages/model/src/standard/elements/smoke-co-alarm-device.element.ts:24,28 @862ad75f (SmokeCoAlarm mandatory, PowerSource device min 1) |
| `binary_sensor/generic` | device_class none | ContactSensor | BooleanState | mappable | matter.js packages/model/src/standard/elements/contact-sensor.element.ts:19 @862ad75f (BooleanState mandatory); matter.js packages/model/src/standard/resources/boolean-state.resource.ts:26-30 @862ad75f (Contact Sensor - FALSE=open or no contact, TRUE=closed or contact); matter.js packages/model/src/standard/elements/on-off-sensor.element.ts:25 @862ad75f (OnOffSensor's OnOff is a client cluster) |
| `binary_sensor/no_device_type` | device_class battery, battery_charging, connectivity, gas, glass_break, heat, light, lock, moving, plug, power, problem, running, safety, sound, tamper, update, vibration |  |  | unmappable: no Matter device type for a binary_sensor of device_class {device_class} |  |
| `button/plug` | any | OnOffPlugInUnit | OnOff | mappable | matter.js packages/model/src/standard/elements/on-off-plug-in-unit.element.ts:27-31 @862ad75f (OnOff mandatory, feature LT); home-assistant/core homeassistant/components/button/__init__.py:101-106 @2026.10.3 (the state is the last press's timestamp); home-assistant/core homeassistant/components/button/services.py:11-15 @2026.10.3 (press, no fields); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/button/index.ts:22-49 @664575a (OnOffPlugInUnit; On presses, OnOff returns to off) |
| `camera/camera` | features any of STREAM | Camera | CameraAvStreamManagement (Video, Audio, Snapshot) [ports: host]; WebRtcTransportProvider [ports: host] | mappable | home-assistant/core homeassistant/components/camera/__init__.py:742-755 @2026.10.3 (camera_capabilities: the stream types WEB_RTC and HLS under STREAM; no codec, resolution or frame rate); go-fabric device/camera_gen.go:34-71 @v0.15.5 (Camera: CameraAvStreamManagement with VDO, ADO, SNP and WebRtcTransportProvider mandatory; no OnOff); go-fabric device/device.go:431-439 @v0.15.5 (a selection that leaves out a mandated feature is ErrConfiguration); go-fabric device/defaults_camera.go:110-125,147-157 @v0.15.5 (the kit's AvStream and Provider servers run on the Declare's ports); go-fabric cluster/camera/avstream.go:194-198 @v0.15.5 (VDO needs a VideoStreamPort, ADO an AudioStreamPort, SNP a SnapshotStreamPort and a SnapshotCapturePort); go-fabric cluster/camera/avstream.go:262-306 @v0.15.5 (chipDefaults: the scalars only; the capability structures and lists start unset); go-fabric cluster/spec/cameraavstreammanagement/definition_gen.go:396,545 @v0.15.5 (RequiresHardwareEncoder only with RequiresEncodedPixels; MaxHdrfps only with HDR, which the row does not select); connectedhomeip data_model/1.7/clusters/CameraAVStreamManagement.xml:173-185 @1dc3b69ed5e (H264 is the mandatory video codec); connectedhomeip examples/camera-app/linux/include/camera-device.h:42-44,48,52-53,58-59,66-67 @1dc3b69ed5e (the camera app's MaxContentBufferSize 4096, MaxNetworkBandwidth 128000000, MaxConcurrentEncoders 1, MaxEncodedPixelRate 248832000, min resolution 640x480, snapshot 30 fps, video 120 fps, sensor 1920x1080); connectedhomeip examples/camera-app/linux/src/camera-device.cpp:1515-1520,1557-1568,1596-1603,1652-1657 @1dc3b69ed5e (VideoSensorParams, MinViewport, one H264 RateDistortionTradeOff point, the first JPEG SnapshotCapabilities entry, LiveView among the supported stream usages); connectedhomeip examples/camera-app/linux/src/camera-device.cpp:1986-1987 @1dc3b69ed5e (the app's mono LiveView audio stream: Opus, one channel, 48000 Hz, 24 bit; its MicrophoneCapabilities at :1576-1584 offer 8 channels, Opus and AAC-LC) |
| `camera/unmapped` | any |  |  | unmappable: a camera without the STREAM feature has no live stream; a SnapshotCamera row is a later wave |  |
| `climate/air_conditioner` | features all of FAN_MODE, TURN_ON, TURN_OFF; modes any of heat, cool, heat_cool | RoomAirConditioner | OnOff (DeadFrontBehavior); Thermostat (Heating, Cooling, AutoMode); FanControl (Auto); RelativeHumidityMeasurement | mappable | go-fabric cluster/thermo/presets.go:246-257 @v0.15.4 (every preset needs the setpoints of the selected Heating and Cooling features); go-fabric device/room_air_conditioner_gen.go:39-86 @v0.15.4 (OnOff mandatory with DF, Thermostat mandatory, FanControl and RelativeHumidityMeasurement optional); matter.js packages/model/src/standard/resources/fan-control.resource.ts:44,80,95 @862ad75f (FanMode, PercentSetting, PercentCurrent); matter.js packages/model/src/standard/resources/relative-humidity-measurement.resource.ts:20-26 @862ad75f (MeasuredValue = 100 x water content); home-assistant/core homeassistant/components/climate/const.py:71-81,194-205 @2026.10.3 (fan modes; FAN_MODE, TURN_OFF, TURN_ON); home-assistant/core homeassistant/components/climate/__init__.py:318-325 @2026.10.3 (current_humidity and fan_mode state attributes); home-assistant/core homeassistant/components/climate/services.py:170-180,214-218 @2026.10.3 (turn_on, turn_off, set_fan_mode); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/climate/index.ts:83-102,172-186 @664575a (FAN_MODE makes a RoomAirConditioner; OnOff with TURN_ON and TURN_OFF; humidity from current_humidity); go-ha-catalog data/features.json:14-24 (climate feature flags) |
| `climate/thermostat` | modes any of heat, cool, heat_cool | Thermostat | Thermostat (Heating, Cooling, AutoMode) | mappable | matter.js packages/model/src/standard/elements/thermostat-device.element.ts:25 @862ad75f (Thermostat mandatory); matter.js packages/model/src/standard/resources/thermostat-cluster.resource.ts:50,163,322 @862ad75f (LocalTemperature, OccupiedHeatingSetpoint, SystemMode); matter.js packages/model/src/standard/elements/temperature.element.ts:12 @862ad75f (the temperature datatype, int16 in 0.01 degC); go-fabric cluster/thermo/presets.go:246-257 @v0.15.4 (every preset needs the setpoints of the selected Heating and Cooling features) |
| `climate/no_heating_or_cooling` | any |  |  | unmappable: a climate entity without the hvac modes heat, cool or heat_cool has no Thermostat feature (Heating or Cooling) |  |
| `cover/window_covering` | any | WindowCovering | WindowCovering (Lift, PositionAwareLift, Tilt, PositionAwareTilt) | mappable | matter.js packages/model/src/standard/elements/window-covering-device.element.ts:24 @862ad75f (WindowCovering mandatory); matter.js packages/model/src/standard/resources/window-covering-cluster.resource.ts:39,98,116-118,180-199 @862ad75f (Type, EndProductType, CurrentPositionLiftPercent100ths; DownOrClose ends at 100.00%) |
| `event/generic_switch` | any | GenericSwitch | Switch (MomentarySwitch, MomentarySwitchRelease) [on state: InitialPress, ShortRelease] | mappable | matter.js packages/model/src/standard/elements/generic-switch.element.ts:19 @862ad75f (Switch mandatory); go-fabric cluster/spec/switchcluster/definition_gen.go:30-31,38,40,54-55 @v0.15.4 (NumberOfPositions, CurrentPosition; InitialPress (MS), ShortRelease (MSR)); go-fabric cluster/spec/switchcluster/server_gen.go:156-171 @v0.15.4 (EmitInitialPress, EmitShortRelease); home-assistant/core homeassistant/components/event/__init__.py:187-194 @2026.10.3 (the state is the last event's timestamp); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/event/index.ts:12-24,191-195 @664575a (GenericSwitch; a short press is InitialPress then ShortRelease at position 1) |
| `fan/fan` | any | Fan | FanControl (Rocking, AirflowDirection, Auto) | mappable | matter.js packages/model/src/standard/elements/fan.element.ts:25 @862ad75f (FanControl mandatory); matter.js packages/model/src/standard/resources/fan-control.resource.ts:44,80,95 @862ad75f (FanMode, PercentSetting, PercentCurrent) |
| `light/extended_color` | supported_color_modes any of hs, xy | ExtendedColorLight | OnOff; LevelControl; ColorControl (Xy, ColorTemperature, HueSaturation) | mappable | matter.js packages/model/src/standard/elements/extended-color-light.element.ts:28-53 @862ad75f (OnOff LT, LevelControl OO+LT, ColorControl XY+CT mandatory, HS optional); matter.js packages/model/src/standard/resources/color-control.resource.ts:41-47,58-64,97-104,139-154 @862ad75f (CurrentHue, CurrentSaturation, CurrentX, ColorTemperatureMireds); go-fabric cluster/spec/levelcontrol/values.go:28-44 @v0.11.0 (LevelFromUnit, UnitFromLevel) |
| `light/rgb` | supported_color_modes any of rgb, rgbw, rgbww | ExtendedColorLight | OnOff; LevelControl; ColorControl (Xy, ColorTemperature, HueSaturation) | partial: an RGB light without hs or xy: a controller's CIE x/y or hue and saturation reach it as color r, g, b at full brightness (go-fabric's XYToRGB, HSVToRGB), so colours outside the sRGB gamut are clipped and the w and ww channels are not driven | matter.js packages/model/src/standard/elements/extended-color-light.element.ts:28-53 @862ad75f (ColorControl XY+CT mandatory, HS optional); go-fabric cluster/spec/colorcontrol/colorconversion.go:108-150 @v0.11.0 (RGBToXY, XYToRGB, matter.js ColorConversionUtils.ts) |
| `light/color_temperature` | supported_color_modes any of color_temp | ColorTemperatureLight | OnOff; LevelControl; ColorControl (ColorTemperature) | mappable | matter.js packages/model/src/standard/elements/color-temperature-light.element.ts:28-48 @862ad75f (OnOff LT, LevelControl OO+LT, ColorControl CT); matter.js packages/model/src/standard/resources/color-control.resource.ts:139-154 @862ad75f (ColorTemperatureMireds, mireds = 1,000,000 / K) |
| `light/dimmable` | supported_color_modes any of brightness, white | DimmableLight | OnOff; LevelControl | mappable | matter.js packages/model/src/standard/elements/dimmable-light.element.ts:28-39 @862ad75f (OnOff LT, LevelControl LT+OO, CurrentLevel 1 to 254); go-fabric cluster/spec/levelcontrol/values.go:28-44 @v0.11.0 (LevelFromUnit, UnitFromLevel) |
| `light/onoff` | any | OnOffLight | OnOff | mappable | matter.js packages/model/src/standard/elements/on-off-light.element.ts:28-30 @862ad75f (OnOff mandatory, feature LT) |
| `lock/door_lock` | any | DoorLock | DoorLock | mappable | matter.js packages/model/src/standard/elements/door-lock-device.element.ts:30 @862ad75f (DoorLock mandatory); matter.js packages/model/src/standard/resources/door-lock-cluster.resource.ts:151 @862ad75f (LockState) |
| `media_player/video_player` | device_class tv | BasicVideoPlayer | OnOff; MediaPlayback; KeypadInput | mappable | matter.js packages/model/src/standard/elements/basic-video-player.element.ts:22,27,30 @862ad75f (OnOff, MediaPlayback and KeypadInput mandatory); go-fabric cluster/spec/mediaplayback/definition_gen.go:45-48,169-172,192-197,425-428 @v0.15.4 (Play, Pause, Stop; PlaybackStateEnum; StatusEnum; PlaybackResponse); go-fabric cluster/spec/keypadinput/definition_gen.go:95-98,238-240,253-255,272-274 @v0.15.4 (Power, VolumeUp, VolumeDown, Mute; StatusEnum; SendKey); go-fabric device/defaults_media.go:63-72 @v0.15.4 (the kit's media servers need their ports; without them the generated servers); home-assistant/core homeassistant/components/media_player/services.py:125-163 @2026.10.3 (turn_on, turn_off, toggle, volume_up, volume_down, media_play, media_pause, media_stop); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/index.ts:24-39 @664575a (device_class tv is a video player); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-media-playback-server.ts:17-63 @664575a (the playback states and commands); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-keypad-input-server.ts:42-130 @664575a (keys onto services) |
| `media_player/speaker` | features all of TURN_ON, TURN_OFF | Speaker | OnOff; LevelControl | mappable | matter.js packages/model/src/standard/elements/speaker.element.ts:18-19 @862ad75f (OnOff and LevelControl mandatory); go-fabric cluster/spec/levelcontrol/values.go:19-44 @v0.15.4 (levels 0 to 254, a unit value times 254); home-assistant/core homeassistant/components/media_player/__init__.py:334,362-364,379-381 @2026.10.3 (volume_level 0 to 1, is_volume_muted); home-assistant/core homeassistant/components/media_player/services.py:125-130,182-203 @2026.10.3 (turn_on, turn_off, volume_set, volume_mute); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/index.ts:41-91 @664575a (a Speaker; power with TURN_ON and TURN_OFF, else mute as OnOff; VolumeSet as LevelControl); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-power-on-off-server.ts:8-16 @664575a (off and standby are off); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-on-off-server.ts:8-17 @664575a (OnOff is not muted) |
| `media_player/speaker_mute` | features any of VOLUME_MUTE | Speaker | OnOff; LevelControl | mappable | matter.js packages/model/src/standard/elements/speaker.element.ts:18-19 @862ad75f (OnOff and LevelControl mandatory); go-fabric cluster/spec/levelcontrol/values.go:19-44 @v0.15.4 (levels 0 to 254, a unit value times 254); home-assistant/core homeassistant/components/media_player/__init__.py:334,362-364,379-381 @2026.10.3 (volume_level 0 to 1, is_volume_muted); home-assistant/core homeassistant/components/media_player/services.py:125-130,182-203 @2026.10.3 (turn_on, turn_off, volume_set, volume_mute); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/index.ts:41-91 @664575a (a Speaker; power with TURN_ON and TURN_OFF, else mute as OnOff; VolumeSet as LevelControl); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-power-on-off-server.ts:8-16 @664575a (off and standby are off); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-on-off-server.ts:8-17 @664575a (OnOff is not muted) |
| `media_player/speaker_volume` | features any of VOLUME_SET | Speaker | OnOff; LevelControl | mappable | matter.js packages/model/src/standard/elements/speaker.element.ts:18-19 @862ad75f (OnOff and LevelControl mandatory); go-fabric cluster/spec/levelcontrol/values.go:19-44 @v0.15.4 (levels 0 to 254, a unit value times 254); home-assistant/core homeassistant/components/media_player/__init__.py:334,362-364,379-381 @2026.10.3 (volume_level 0 to 1, is_volume_muted); home-assistant/core homeassistant/components/media_player/services.py:125-130,182-203 @2026.10.3 (turn_on, turn_off, volume_set, volume_mute); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/index.ts:41-91 @664575a (a Speaker; power with TURN_ON and TURN_OFF, else mute as OnOff; VolumeSet as LevelControl); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-power-on-off-server.ts:8-16 @664575a (off and standby are off); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/media-player/behaviors/media-player-on-off-server.ts:8-17 @664575a (OnOff is not muted) |
| `media_player/unmapped` | any |  |  | unmappable: a media player without turn_on and turn_off, volume_mute or volume_set has nothing a Speaker serves |  |
| `number/unmapped` | any |  |  | unmappable: no generic Matter counterpart for a number entity |  |
| `select/mode_select` | any | ModeSelect | ModeSelect [options desc.options at state to select_option] | mappable | matter.js packages/model/src/standard/elements/mode-select-device.element.ts:18 @862ad75f (ModeSelect mandatory); go-fabric cluster/spec/modeselect/definition_gen.go:30-40,54-58 @v0.15.4 (SupportedModes, CurrentMode, ChangeToMode; ModeOptionStruct); home-assistant/core homeassistant/components/select/__init__.py:108-122 @2026.10.3 (options capability; state is the current option); home-assistant/core homeassistant/components/select/services.py:42-46 @2026.10.3 (select_option with option); home-assistant-matter-hub packages/backend/src/matter/behaviors/mode-select-server.ts:10-16 @664575a (the option's 0-based index is the mode) |
| `sensor/temperature` | device_class temperature | TemperatureSensor | TemperatureMeasurement | mappable | matter.js packages/model/src/standard/elements/temperature-sensor.element.ts:18 @862ad75f (TemperatureMeasurement mandatory); matter.js packages/model/src/standard/elements/temperature.element.ts:12 @862ad75f (MeasuredValue is the temperature datatype, int16 in 0.01 degC); go-ha-catalog data/sensor.json:633-637 (temperature units K, degC, degF) |
| `sensor/humidity` | device_class humidity | HumiditySensor | RelativeHumidityMeasurement | mappable | matter.js packages/model/src/standard/elements/humidity-sensor.element.ts:19 @862ad75f (RelativeHumidityMeasurement mandatory); matter.js packages/model/src/standard/resources/relative-humidity-measurement.resource.ts:20-26 @862ad75f (MeasuredValue = 100 x water content); go-ha-catalog data/sensor.json:511-513 (humidity unit %) |
| `sensor/pressure` | device_class pressure, atmospheric_pressure | PressureSensor | PressureMeasurement | mappable | matter.js packages/model/src/standard/elements/pressure-sensor.element.ts:19 @862ad75f (PressureMeasurement mandatory); matter.js packages/model/src/standard/resources/pressure-measurement.resource.ts:24-27 @862ad75f (MeasuredValue = 10 x Pressure [kPa]); go-ha-catalog data/sensor.json:362-375,579-592 (atmospheric_pressure and pressure units) |
| `sensor/illuminance` | device_class illuminance | LightSensor | IlluminanceMeasurement | mappable | matter.js packages/model/src/standard/elements/light-sensor.element.ts:19 @862ad75f (IlluminanceMeasurement mandatory); matter.js packages/model/src/standard/resources/illuminance-measurement.resource.ts:19-30 @862ad75f (MeasuredValue = 10,000 x log10(illuminance) + 1, 1 to 0xFFFE; 0 too low to measure); go-ha-catalog data/sensor.json:514-516 (illuminance unit lx) |
| `sensor/carbon_dioxide` | device_class carbon_dioxide | AirQualitySensor | AirQuality; CarbonDioxideConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); go-ha-catalog data/sensor.json:383-385 (carbon_dioxide unit ppm) |
| `sensor/carbon_monoxide` | device_class carbon_monoxide | AirQualitySensor | AirQuality; CarbonMonoxideConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); home-assistant/core homeassistant/util/unit_conversion.py:243-262 @2026.10.3 (CarbonMonoxideConcentrationConverter: ppm 1e6, ppb 1e9, mg/m³ and μg/m³ through the molar mass, 1e3 apart); go-ha-catalog data/sensor.json:386-391 (carbon_monoxide units) |
| `sensor/pm1` | device_class pm1 | AirQualitySensor | AirQuality; Pm1ConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); go-ha-catalog data/sensor.json:544-546 (pm1 unit μg/m³) |
| `sensor/pm25` | device_class pm25 | AirQualitySensor | AirQuality; Pm25ConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); go-ha-catalog data/sensor.json:550-552 (pm25 unit μg/m³) |
| `sensor/pm10` | device_class pm10 | AirQualitySensor | AirQuality; Pm10ConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); go-ha-catalog data/sensor.json:547-549 (pm10 unit μg/m³) |
| `sensor/volatile_organic_compounds` | device_class volatile_organic_compounds | AirQualitySensor | AirQuality; TotalVolatileOrganicCompoundsConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); home-assistant/core homeassistant/util/unit_conversion.py:495-503 @2026.10.3 (MassVolumeConcentrationConverter: 1000 μg/m³ = 1 mg/m³); go-ha-catalog data/sensor.json:643-646 (volatile_organic_compounds units) |
| `sensor/volatile_organic_compounds_parts` | device_class volatile_organic_compounds_parts | AirQualitySensor | AirQuality; TotalVolatileOrganicCompoundsConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); home-assistant/core homeassistant/util/unit_conversion.py:941-956 @2026.10.3 (UnitlessRatioConverter: ppm 1e6, ppb 1e9); go-ha-catalog data/sensor.json:647-650 (volatile_organic_compounds_parts units) |
| `sensor/nitrogen_dioxide` | device_class nitrogen_dioxide | AirQualitySensor | AirQuality; NitrogenDioxideConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); home-assistant/core homeassistant/util/unit_conversion.py:511-526 @2026.10.3 (NitrogenDioxideConcentrationConverter: ppm 1e6, ppb 1e9, μg/m³ through the molar mass); go-ha-catalog data/sensor.json:524-528 (nitrogen_dioxide units) |
| `sensor/ozone` | device_class ozone | AirQualitySensor | AirQuality; OzoneConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); home-assistant/core homeassistant/util/unit_conversion.py:545-560 @2026.10.3 (OzoneConcentrationConverter: ppm 1e6, ppb 1e9, μg/m³ through the molar mass); go-ha-catalog data/sensor.json:536-540 (ozone units) |
| `sensor/radon` | device_class radon | AirQualitySensor | AirQuality; RadonConcentrationMeasurement (NumericMeasurement) | mappable | matter.js packages/model/src/standard/elements/air-quality-sensor.element.ts:19,22-34 @862ad75f (AirQuality mandatory, the concentration clusters optional); go-fabric cluster/spec/carbondioxideconcentrationmeasurement/definition_gen.go:47,87-89,108-115 @v0.15.4 (NumericMeasurement; MeasurementMedium; MeasurementUnit, the same names in every concentration package); go-fabric cluster/measurement/concentration.go:136-139 @v0.15.4 (AirQuality reads Unknown beside a concentration); home-assistant/core homeassistant/util/unit_conversion.py:111,623-633 @2026.10.3 (RadiationConcentrationConverter: 1 pCi/L = 37 Bq/m³); go-ha-catalog data/sensor.json:593-596 (radon units) |
| `sensor/power` | device_class power | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement and ElectricalEnergyMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:80,97,158,385-389 @862ad75f (Voltage mV, ActiveCurrent mA, ActivePower mW); go-fabric docs/guides/embed-in-a-daemon.md section 3 @v0.11.0 (the ElectricalSensor declaration of the cookbook); go-ha-catalog data/sensor.json:397-401,457-473,556-563,651-657 (current, energy, power, voltage units) |
| `sensor/voltage` | device_class voltage | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement and ElectricalEnergyMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:80,97,158,385-389 @862ad75f (Voltage mV, ActiveCurrent mA, ActivePower mW); go-fabric docs/guides/embed-in-a-daemon.md section 3 @v0.11.0 (the ElectricalSensor declaration of the cookbook); go-ha-catalog data/sensor.json:397-401,457-473,556-563,651-657 (current, energy, power, voltage units) |
| `sensor/current` | device_class current | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement and ElectricalEnergyMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:80,97,158,385-389 @862ad75f (Voltage mV, ActiveCurrent mA, ActivePower mW); go-fabric docs/guides/embed-in-a-daemon.md section 3 @v0.11.0 (the ElectricalSensor declaration of the cookbook); go-ha-catalog data/sensor.json:397-401,457-473,556-563,651-657 (current, energy, power, voltage units) |
| `sensor/apparent_power` | device_class apparent_power | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:201-203 @862ad75f (ApparentPower in millivolt-amps, mVA); home-assistant/core homeassistant/util/unit_conversion.py:195-209 @2026.10.3 (ApparentPowerConverter: mVA 1000, VA 1, kVA 1/1000); go-ha-catalog data/sensor.json:342-346 (apparent_power units) |
| `sensor/reactive_power` | device_class reactive_power | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:179-181 @862ad75f (ReactivePower in millivolt-amps reactive, mVAR); home-assistant/core homeassistant/util/unit_conversion.py:647-661 @2026.10.3 (ReactivePowerConverter: mvar 1000, var 1, kvar 1/1000); go-ha-catalog data/sensor.json:601-605 (reactive_power units) |
| `sensor/frequency` | device_class frequency | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:278 @862ad75f (Frequency in millihertz, mHz); go-fabric cluster/spec/electricalpowermeasurement/definition_gen.go:515-521 @v0.15.4 (Frequency 0 to 1000000); home-assistant/core homeassistant/util/unit_conversion.py:365-375 @2026.10.3 (FrequencyConverter: mHz 1e3, Hz 1, kHz 1e-3, MHz 1e-6, GHz 1e-9); go-ha-catalog data/sensor.json:497-503 (frequency units) |
| `sensor/power_factor` | device_class power_factor | ElectricalSensor | PowerTopology (NodeTopology); ElectricalPowerMeasurement (AlternatingCurrent) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:328 @862ad75f (PowerFactor in +/- 1/100ths of a percent); go-fabric cluster/spec/electricalpowermeasurement/definition_gen.go:542-548 @v0.15.4 (PowerFactor -10000 to 10000); home-assistant/core homeassistant/util/unit_conversion.py:941-956 @2026.10.3 (UnitlessRatioConverter: no unit 1, % 100); go-ha-catalog data/sensor.json:564-567 (power_factor units %, none: the key "") |
| `sensor/energy` | device_class energy | ElectricalSensor | PowerTopology (NodeTopology); ElectricalEnergyMeasurement (ImportedEnergy, CumulativeEnergy) | mappable | matter.js packages/model/src/standard/elements/electrical-sensor.element.ts:18-20 @862ad75f (PowerTopology mandatory, ElectricalPowerMeasurement and ElectricalEnergyMeasurement O.a+); matter.js packages/model/src/standard/resources/electrical-power-measurement.resource.ts:80,97,158,385-389 @862ad75f (Voltage mV, ActiveCurrent mA, ActivePower mW); go-fabric docs/guides/embed-in-a-daemon.md section 3 @v0.11.0 (the ElectricalSensor declaration of the cookbook); go-ha-catalog data/sensor.json:397-401,457-473,556-563,651-657 (current, energy, power, voltage units) |
| `sensor/volume_flow_rate` | device_class volume_flow_rate | FlowSensor | FlowMeasurement | mappable | matter.js packages/model/src/standard/elements/flow-sensor.element.ts:19 @862ad75f (FlowMeasurement mandatory); matter.js packages/model/src/standard/resources/flow-measurement.resource.ts:19-24 @862ad75f (MeasuredValue = 10 x Flow in m³/h); go-fabric cluster/measurement/flow.go:34-38 @v0.15.4 (uint16, 0.1 m³/h, clamped to 0..65534); home-assistant/core homeassistant/util/unit_conversion.py:44-45,66-70,88-92,986-1004 @2026.10.3 (VolumeFlowRateConverter: ratios to m³/h); go-ha-catalog data/sensor.json:668-680 (volume_flow_rate units) |
| `sensor/moisture` | device_class moisture | SoilSensor | SoilMeasurement | mappable | matter.js packages/model/src/standard/elements/soil-sensor.element.ts:20 @862ad75f (SoilMeasurement mandatory); matter.js packages/model/src/standard/resources/soil-measurement.resource.ts:20-32,36-37 @862ad75f (the limits in percent, one accuracy range with PercentMax at most 10.00 %; the measured value in percent); go-fabric cluster/spec/soilmeasurement/definition_gen.go:40-46,105,133-142 @v0.15.4 (MeasurementAccuracyStruct, MeasurementTypeSoilMoisture, the range's PercentMax in 1/100 %); go-ha-catalog data/sensor.json:521-523 (moisture unit %) |
| `sensor/battery` | device_class battery | PowerSource (rides on the primary) | PowerSource (Battery); rides as the battery measurement | partial: the battery rides on the device's primary endpoint as go-fabric's battery measurement (PowerSource, BatPercentRemaining), not as a declared PowerSource, until the kit carries PowerSourceConfiguration for a bridged PowerSource | matter.js packages/model/src/standard/elements/power-source-device.element.ts:18 @862ad75f (PowerSource mandatory); matter.js packages/model/src/standard/elements/bridged-node.element.ts:23 @862ad75f (PowerSource under BridgedPowerSourceInfo on a bridged node); matter.js packages/model/src/standard/resources/power-source-cluster.resource.ts:120-125 @862ad75f (BatPercentRemaining in half percent, 0 to 200); go-fabric docs/guides/embed-in-a-daemon.md section 3 @v0.11.0 (powersourcedef.Declare on the primary endpoint); go-ha-catalog data/sensor.json:376-378 (battery unit %) |
| `sensor/no_device_type` | device_class absolute_humidity, aqi, area, blood_glucose_concentration, conductivity, data_rate, data_size, date, distance, duration, energy_distance, energy_storage, enum, gas, irradiance, monetary, nitrogen_monoxide, nitrous_oxide, ph, pm4, precipitation, precipitation_intensity, reactive_energy, signal_strength, sound_pressure, speed, sulphur_dioxide, temperature_delta, timestamp, uptime, volume, volume_storage, water, weight, wind_direction, wind_speed |  |  | unmappable: no Matter device type for a sensor of device_class {device_class} |  |
| `sensor/no_device_class` | device_class none |  |  | unmappable: a sensor without device_class has no Matter device type |  |
| `switch/plug` | device_class none, outlet, switch | OnOffPlugInUnit | OnOff | mappable | matter.js packages/model/src/standard/elements/on-off-plug-in-unit.element.ts:27-31 @862ad75f (OnOff mandatory, feature LT) |
| `text/unmapped` | any |  |  | unmappable: no Matter counterpart for a text entity |  |
| `vacuum/robotic_vacuum_cleaner` | features any of START | RoboticVacuumCleaner | RvcRunMode; RvcOperationalState | mappable | matter.js packages/model/src/standard/elements/robotic-vacuum-cleaner.element.ts:19-25 @862ad75f (RvcRunMode and RvcOperationalState mandatory); go-fabric cluster/spec/rvcrunmode/definition_gen.go:56-71,95-99,129-132,165-177,202,221-224 @v0.15.4 (ModeChangeStatus, ModeOptionStruct, ModeTagStruct, ModeTag Idle and Cleaning, ChangeToMode); go-fabric cluster/spec/rvcoperationalstate/definition_gen.go:40-45,66-84,155-165,188-191 @v0.15.4 (Pause, Resume, GoHome; ErrorStateEnum; OperationalStateEnum; OperationalStateStruct); go-fabric device/defaults_ports.go:30-33 @v0.15.4 (without its port a cluster keeps the generated server); home-assistant/core homeassistant/components/vacuum/const.py:30-38,53-69 @2026.10.3 (activities; PAUSE, RETURN_HOME, START); home-assistant/core homeassistant/components/vacuum/services.py:104-119 @2026.10.3 (start, pause, return_to_base); home-assistant-matter-hub packages/backend/src/matter/behaviors/rvc-run-mode-server.ts:20-23 @664575a (Idle 0, Cleaning 1); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/vacuum/behaviors/vacuum-rvc-operational-state-server.ts:78-161 @664575a (activities onto operational states; pause, resume as start, go home as return_to_base) |
| `vacuum/unmapped` | any |  |  | unmappable: a vacuum without the START feature cannot be started from a run mode |  |
| `valve/water_valve` | any | WaterValve | ValveConfigurationAndControl | mappable | matter.js packages/model/src/standard/elements/water-valve.element.ts:19 @862ad75f (ValveConfigurationAndControl mandatory); go-fabric cluster/spec/valveconfigurationandcontrol/definition_gen.go:30-46,117-119 @v0.15.4 (OpenDuration, RemainingDuration, CurrentState, TargetState; Open, Close; ValveStateEnum); go-fabric device/defaults_declared.go:137-149 @v0.15.4 (the kit's valve server needs a StateSource and serves no feature; without it the generated server); home-assistant/core homeassistant/components/valve/const.py:36-51 @2026.10.3 (OPEN, CLOSE, SET_POSITION; the states); home-assistant/core homeassistant/components/valve/services.py:22-31 @2026.10.3 (open_valve, close_valve); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/valve/index.ts:11-31 @664575a (WaterValve; opening and closing are Transitioning) |
| `water_heater/thermostat` | any | Thermostat | Thermostat (Heating) | mappable | matter.js packages/model/src/standard/elements/thermostat-device.element.ts:25 @862ad75f (Thermostat mandatory); matter.js packages/model/src/standard/elements/temperature.element.ts:12 @862ad75f (the temperature datatype, int16 in 0.01 degC); home-assistant/core homeassistant/components/water_heater/__init__.py:146-151,164-182,187-219 @2026.10.3 (the state is the operation mode; min_temp, max_temp; current_temperature, temperature); home-assistant/core homeassistant/components/water_heater/const.py:53-58 @2026.10.3 (the operation states eco, electric, performance, high_demand, heat_pump, gas); home-assistant/core homeassistant/const.py:299-300 @2026.10.3 (on, off); home-assistant/core homeassistant/components/water_heater/services.py:78-100 @2026.10.3 (turn_on, turn_off, set_temperature, set_operation_mode); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/water-heater/index.ts:14-20,34-45 @664575a (a heating-only Thermostat with the entity's limits); home-assistant-matter-hub packages/backend/src/matter/endpoints/legacy/water-heater/behaviors/water-heater-thermostat-server.ts:46-81 @664575a (off is Off, else Heat; off to set_operation_mode, else turn_on; the setpoint to set_temperature) |

<!-- rows:end -->

## The keys a source fills

A source that is not MQTT (go-ha2fabric's registry adapter, a daemon's own
entities) fills `model.State` and `Description.Extra` under the names the
rows read. `state` is `model.State.Value` (or its `state` key when the
value is a map); `state.<key>` a key of a map value, else of
`model.State.Extra`; `extra.<key>` a key of `Description.Extra`, which a
predicate, a feature condition or a value mapping reads. A key with a
catalog default (go-ha-catalog `data/mqtt.json`, the platform's keys and
its variants' keys) may be left out: the rows read the default, except a
predicate's `present`, which asks the entity. `extra.supported_features`
is the catalog's flag bits or names. The table is generated from the rows.

<!-- keys:begin (generated by internal/gen; do not edit) -->

| Platform | Key | Read by | Read as | Catalog default |
| --- | --- | --- | --- | --- |
| `binary_sensor` | `state` | `binary_sensor/contact`, `binary_sensor/motion`, `binary_sensor/occupancy`, `binary_sensor/water_leak`, `binary_sensor/water_freeze`, `binary_sensor/smoke`, `binary_sensor/carbon_monoxide`, `binary_sensor/generic` | value | — |
| `camera` | `extra.supported_features` | `camera/camera` | predicate | no |
| `climate` | `extra.fan_modes` | `climate/air_conditioner`, `climate/thermostat` | predicate | yes |
| `climate` | `extra.max_temp` | `climate/air_conditioner`, `climate/thermostat` | value | no |
| `climate` | `extra.min_temp` | `climate/air_conditioner`, `climate/thermostat` | value | no |
| `climate` | `extra.modes` | `climate/air_conditioner`, `climate/thermostat` | predicate | yes |
| `climate` | `extra.preset_modes` | `climate/air_conditioner`, `climate/thermostat` | predicate | yes |
| `climate` | `extra.supported_features` | `climate/air_conditioner` | predicate | no |
| `climate` | `extra.swing_modes` | `climate/air_conditioner`, `climate/thermostat` | predicate | yes |
| `climate` | `state` | `climate/air_conditioner`, `climate/thermostat` | value | — |
| `climate` | `state.current_humidity` | `climate/air_conditioner` | value | — |
| `climate` | `state.current_temperature` | `climate/air_conditioner`, `climate/thermostat` | value | — |
| `climate` | `state.fan_mode` | `climate/air_conditioner` | value | — |
| `climate` | `state.target_temp_high` | `climate/air_conditioner`, `climate/thermostat` | value | — |
| `climate` | `state.target_temp_low` | `climate/air_conditioner`, `climate/thermostat` | value | — |
| `climate` | `state.temperature` | `climate/air_conditioner`, `climate/thermostat` | value | — |
| `cover` | `extra.payload_close` | `cover/window_covering` | value | yes |
| `cover` | `extra.payload_open` | `cover/window_covering` | value | yes |
| `cover` | `extra.payload_stop` | `cover/window_covering` | value | yes |
| `cover` | `extra.supported_features` | `cover/window_covering` | predicate | no |
| `cover` | `state.current_position` | `cover/window_covering` | value | — |
| `cover` | `state.current_tilt_position` | `cover/window_covering` | value | — |
| `event` | `state` | `event/generic_switch` | value | — |
| `fan` | `extra.payload_off` | `fan/fan` | value | yes |
| `fan` | `extra.payload_on` | `fan/fan` | value | yes |
| `fan` | `extra.payload_oscillation_off` | `fan/fan` | value | yes |
| `fan` | `extra.payload_oscillation_on` | `fan/fan` | value | yes |
| `fan` | `extra.preset_modes` | `fan/fan` | predicate | yes |
| `fan` | `extra.supported_features` | `fan/fan` | predicate | no |
| `fan` | `state` | `fan/fan` | value | — |
| `fan` | `state.direction` | `fan/fan` | value | — |
| `fan` | `state.oscillating` | `fan/fan` | value | — |
| `fan` | `state.percentage` | `fan/fan` | value | — |
| `fan` | `state.preset_mode` | `fan/fan` | value | — |
| `light` | `extra.brightness_scale` | `light/extended_color`, `light/rgb`, `light/color_temperature`, `light/dimmable` | value | yes |
| `light` | `extra.color_temp_kelvin` | `light/extended_color`, `light/color_temperature` | predicate | yes |
| `light` | `extra.max_kelvin` | `light/extended_color`, `light/rgb`, `light/color_temperature` | value | no |
| `light` | `extra.max_mireds` | `light/extended_color`, `light/rgb`, `light/color_temperature` | value | no |
| `light` | `extra.min_kelvin` | `light/extended_color`, `light/rgb`, `light/color_temperature` | value | no |
| `light` | `extra.min_mireds` | `light/extended_color`, `light/rgb`, `light/color_temperature` | value | no |
| `light` | `extra.supported_color_modes` | `light/extended_color`, `light/rgb`, `light/color_temperature`, `light/dimmable` | predicate | no |
| `light` | `state` | `light/extended_color`, `light/rgb`, `light/color_temperature`, `light/dimmable`, `light/onoff` | value | — |
| `light` | `state.brightness` | `light/extended_color`, `light/rgb`, `light/color_temperature`, `light/dimmable` | value | — |
| `light` | `state.color.h` | `light/extended_color`, `light/rgb` | value | — |
| `light` | `state.color.s` | `light/extended_color`, `light/rgb` | value | — |
| `light` | `state.color.x` | `light/extended_color`, `light/rgb` | value | — |
| `light` | `state.color.y` | `light/extended_color`, `light/rgb` | value | — |
| `light` | `state.color_mode` | `light/extended_color`, `light/rgb` | value | — |
| `light` | `state.color_temp` | `light/extended_color`, `light/color_temperature` | value | — |
| `lock` | `extra.payload_lock` | `lock/door_lock` | value | yes |
| `lock` | `extra.payload_open` | `lock/door_lock` | predicate | no |
| `lock` | `extra.payload_unlock` | `lock/door_lock` | value | yes |
| `lock` | `extra.supported_features` | `lock/door_lock` | predicate | no |
| `lock` | `state` | `lock/door_lock` | value | — |
| `media_player` | `extra.supported_features` | `media_player/video_player`, `media_player/speaker`, `media_player/speaker_mute`, `media_player/speaker_volume` | predicate | no |
| `media_player` | `state` | `media_player/video_player`, `media_player/speaker`, `media_player/speaker_volume` | value | — |
| `media_player` | `state.is_volume_muted` | `media_player/speaker_mute` | value | — |
| `media_player` | `state.volume_level` | `media_player/speaker`, `media_player/speaker_mute`, `media_player/speaker_volume` | value | — |
| `select` | `state` | `select/mode_select` | value | — |
| `sensor` | `state` | `sensor/temperature`, `sensor/humidity`, `sensor/pressure`, `sensor/illuminance`, `sensor/carbon_dioxide`, `sensor/carbon_monoxide`, `sensor/pm1`, `sensor/pm25`, `sensor/pm10`, `sensor/volatile_organic_compounds`, `sensor/volatile_organic_compounds_parts`, `sensor/nitrogen_dioxide`, `sensor/ozone`, `sensor/radon`, `sensor/power`, `sensor/voltage`, `sensor/current`, `sensor/apparent_power`, `sensor/reactive_power`, `sensor/frequency`, `sensor/power_factor`, `sensor/energy`, `sensor/volume_flow_rate`, `sensor/moisture`, `sensor/battery` | value | — |
| `switch` | `extra.payload_off` | `switch/plug` | value | yes |
| `switch` | `extra.payload_on` | `switch/plug` | value | yes |
| `switch` | `state` | `switch/plug` | value | — |
| `vacuum` | `extra.supported_features` | `vacuum/robotic_vacuum_cleaner` | predicate | yes |
| `vacuum` | `state` | `vacuum/robotic_vacuum_cleaner` | value | — |
| `valve` | `extra.supported_features` | `valve/water_valve` | predicate | no |
| `valve` | `state` | `valve/water_valve` | value | — |
| `water_heater` | `extra.max_temp` | `water_heater/thermostat` | value | no |
| `water_heater` | `extra.min_temp` | `water_heater/thermostat` | value | no |
| `water_heater` | `state` | `water_heater/thermostat` | value | — |
| `water_heater` | `state.current_temperature` | `water_heater/thermostat` | value | — |
| `water_heater` | `state.temperature` | `water_heater/thermostat` | value | — |

<!-- keys:end -->
