# By design

Deliberate departures from what the brief or matter.js would suggest, one
paragraph each, with the citations.

## Generic binary_sensor is a ContactSensor

A Home Assistant binary_sensor without a device class maps to ContactSensor
(BooleanState; Home Assistant's `on` is StateValue false, as for `opening`),
not to OnOffSensor: OnOffSensor's OnOff is a client cluster (matter.js
`packages/model/src/standard/elements/on-off-sensor.element.ts:25` at
`862ad75f`), so a controller would never see the sensor's state, while
ContactSensor serves it in BooleanState (`contact-sensor.element.ts:19`;
`packages/model/src/standard/resources/boolean-state.resource.ts:26-30`:
FALSE = open or no contact, TRUE = closed or contact). Row
`binary_sensor/generic`; review decision on PR #4 (Q6).

## A switch is a plug-in unit

Every Home Assistant `switch` becomes an OnOffPlugInUnit, whatever its
device class (`outlet`, `switch` or none; `switch/const.py:19-23` at
`2026.10.3`), as home-assistant-matter-hub does: its `SwitchDevice` is an
`OnOffPlugInUnitDevice` and reads no device class
(`packages/backend/src/matter/endpoints/legacy/switch/index.ts:13-22,
34-68` at `664575a`). An OnOffLight for `device_class: switch` would claim
a lamp the entity does not say it drives, and the plug-in unit is what
controllers already show for a matter-hub switch. Row `switch/plug`; owner
decision of 2026-10-31.

## A moisture sensor is a SoilSensor

A Home Assistant `sensor` of device class `moisture` (unit `%`,
go-ha-catalog `data/sensor.json:521-523`) becomes a SoilSensor with
SoilMeasurement's `SoilMoistureMeasuredValue` in percent (matter.js
`packages/model/src/standard/elements/soil-sensor.element.ts:20`,
`packages/model/src/standard/resources/soil-measurement.resource.ts:36-37`
at `862ad75f`), not a HumiditySensor as home-assistant-matter-hub maps it
(`packages/backend/src/matter/endpoints/legacy/sensor/index.ts:102-111` at
`664575a`, its #273): relative humidity is water in air, and the device type
that measures water in soil exists. Row `sensor/moisture`.

## A button is a plug-in unit, not a switch

A Home Assistant `button` becomes an OnOffPlugInUnit whose On presses it
(the `press` service, `button/services.py:11-15` at `2026.10.3`) and whose
OnOff stays off, as home-assistant-matter-hub serves it
(`packages/backend/src/matter/endpoints/legacy/button/index.ts:22-49` at
`664575a`), not a GenericSwitch: a button is an action target a controller
triggers, while a GenericSwitch reports presses to the controller — the
other direction. Row `button/plug`.

## A select's mode is its option's index

A Home Assistant `select` becomes a ModeSelect whose modes are its options
in order, each mode the option's 0-based index, as home-assistant-matter-hub
numbers them (`packages/backend/src/matter/behaviors/mode-select-server.ts:
10-16` at `664575a`): the options carry no number of their own, and the same
numbering keeps a controller's scene stored against one bridge valid on the
other. Row `select/mode_select`.

## Camera: a nominal stream declaration

A Home Assistant camera with the STREAM feature becomes a Camera whose
CameraAvStreamManagement declares VDO, ADO and SNP with fixed capabilities,
although Home Assistant names no codec, resolution or frame rate: its
capabilities are the frontend stream types WEB_RTC and HLS only
(`homeassistant/components/camera/__init__.py:742-755` at `2026.10.3`).
The values are connectedhomeip's camera app's (`examples/camera-app/linux/
include/camera-device.h:42-67`, `src/camera-device.cpp:1515-1603, 1652-1657,
1986-1987` at `1dc3b69ed5e`): one H264 trade-off point at 640×480, the
mandatory codec (`data_model/1.7/clusters/CameraAVStreamManagement.xml:
173-185`), a 1920×1080 sensor at 120 fps, one JPEG snapshot capability,
LiveView only, and a microphone of one Opus channel at 48 kHz and 24 bit —
the app's mono stream, narrower than its 8-channel MicrophoneCapabilities.
The Camera type mandates all three features (go-fabric
`device/camera_gen.go:36-40` at `v0.15.5`), and the host relays the
controller's offer and answer without reading the stream, so a declaration
it cannot measure is nominal by necessity; the verdict says so. Row
`camera/camera`.

## Camera without STREAM is unmapped

A camera without the STREAM feature has no live stream
(`camera/__init__.py:745` at `2026.10.3` adds a stream type only under
STREAM), and the Camera type's mandatory WebRtcTransportProvider has nothing
to offer; a snapshot-only device type is a later wave. ON_OFF
(`camera/const.py:74` at `2026.10.3`) stays unmapped on a streaming camera
too: the Camera type lists no OnOff (`device/camera_gen.go:34-71` at
`v0.15.5`). Row `camera/unmapped`.
