# Brief R6 (go-hamapping) — host ports in a row; the camera row (v0.4.0)

Owner decisions (2026-10-31, evening): build R6 now; one pull request per
repository on `roadmap/r6-camera`, squash-merged, auto-merge; tag `v0.4.0`
when the pull request is merged (the tag is this repository's release).
go-ha2fabric's roadmap: `../go-ha2fabric/docs/roadmap.md` §R6.

Reads (maps, `file:line` for every fact; verify a load-bearing line before
it enters code or docs): `notes/reads/read-r6-ports-model.md` (this repo's
plan API, go-hamqtt's model, go-ha2fabric's plan application),
`../go-ha2fabric/notes/reads/read-r6-gofabric-camera-kit.md` (go-fabric
v0.15.5: the Camera type, the Declare structs, the ports, the AvStream
server), `../go-ha2fabric/notes/reads/read-r6-ha-webrtc.md` and
`read-r6-camera.md` (Home Assistant 2026.10.3). go-fabric is checked out at
`v0.15.5` in `../go-fabric`; go.mod already pins `v0.15.5`.

Facts verified by the owner for this brief:

- A Camera (`device/camera_gen.go:36-45`) mandates CameraAvStreamManagement
  with `Features: 0x7` (VDO, ADO, SNP) and WebRtcTransportProvider. A host
  selection that leaves a mandated feature out is `ErrConfiguration`
  (`device/device.go:431-439`, `selection`). So the row declares all three
  and the host must serve four stream ports plus the provider port.
- A Declare with any port but a missing needed one fails `device.New`; a
  Declare without any port keeps the generated server
  (`device/device.go:390-404`, `device/defaults_camera.go:110-125, 147-157`,
  `cluster/camera/avstream.go:194-198`). A camera must therefore never be
  built without its host ports: a row that needs them is skipped when the
  consumer serves none.
- The row format's cluster entry has the keys cluster, features, initial,
  start_null, attributes, commands, events, options (`internal/rowfile/
  rowfile.go:100-116`; unknown keys are errors, `:289`). `Options` has
  `StableKey` and `Name` (`plan.go:97-`). The generated declaration is a
  closure returning a `<x>def.Declare` value (`internal/gen/gen.go:964-1020`);
  the one materializer is `b.optionsModeSelect(&d, ci)` (`gen.go:1009-1015`,
  table `materializers` `gen.go:1297`, `options.go:124-148`).
- Home Assistant names no codec, resolution or frame rate for a camera; its
  capabilities are the stream types WEB_RTC and HLS
  (`homeassistant/components/camera/__init__.py:742-755` @2026.10.3).
  `CameraEntityFeature`: `ON_OFF = 1`, `STREAM = 2` (`camera/const.py:48-51`);
  go-ha-catalog `features.json:10-13` has the same.

## §1 Row format: `ports: host`

- New cluster-entry key `ports` with the single value `host` (rowfile: a
  string field, any other value is an error with the row id and cluster).
  Meaning: the kit's server of this cluster runs on ports the consumer
  supplies through `Options.Ports` (§2); the row is skipped for an entity
  whose host does not implement every port the cluster needs.
- `internal/gen`: a `ports: host` cluster emits `b.ports<Cluster>(&d)` in
  the declare closure, from a table `portMaterializers` (cluster package →
  method name), beside `materializers`. A `ports: host` on a cluster without
  an entry is a generator error naming the cluster. Entries for this wave:
  `webrtctransportprovider` → `portsWebRTCProvider`,
  `cameraavstreammanagement` → `portsAvStream`.
- `internal/gen/defs.go`: add `cameraavstreammanagement` and
  `webrtctransportprovider` (the `cluster/spec/<x>` packages go-fabric's
  camera kit declares on, `device/defaults_camera.go:13-17`).
- Regenerate (`make generate`), never patch `rows_gen.go`.

## §2 `Options.Ports` and the materializers (root package, `ports.go`)

```go
// Ports returns the host object whose interfaces the clusters a row marks
// `ports: host` run on — go-fabric's port interfaces of those clusters
// (cluster/spec/webrtctransportprovider.ProviderPort;
// cluster/spec/cameraavstreammanagement.VideoStreamPort, AudioStreamPort,
// SnapshotStreamPort, SnapshotCapturePort). nil, or an object that lacks
// a port the cluster needs, skips the row for that entity with a verdict.
Ports func(e model.Entity) any
```

- `portsWebRTCProvider(d *wpdef.Declare) bool`: `d.ProviderPort`, from a
  type assertion on the host object; false when missing.
- `portsAvStream(d *csdef.Declare) bool`: the four stream ports; false
  unless all four are present (the row's features mandate VDO, ADO, SNP;
  `avstream.go:194-198`).
- The host object is asked once per binding (`Options.Ports(b.entity)`),
  cached on the binding.
- Skipping: when a materializer returns false the row does not apply to the
  entity; `Map` reports a `Verdict` of the shape the existing unmappable
  verdicts have, text: `"<row id>: the consumer serves no host ports for
  <cluster>"`. No endpoint is built. Where this check lives (in `endpoint()`
  after `declarations(b)`, or as an implicit `when`) is the implementer's
  choice; the observable behaviour and the test are fixed.

## §3 The camera row (`rows/camera.yaml`)

- Row `camera`, `when: {features: {any: [STREAM]}}`, `type: Camera`,
  verdict mappable. Clusters:
  - `cameraavstreammanagement`: `features: [VDO, ADO, SNP]`,
    `ports: host`, `initial` with the attributes the kit's server does not
    default (`avstream.go:262-306` sets only scalars; the read's §Q2):
    `SupportedStreamUsages: [LiveView]`, `MaxConcurrentEncoders`,
    `MaxEncodedPixelRate`, `MaxNetworkBandwidth`, `MaxContentBufferSize`,
    `VideoSensorParams`, `MinViewportResolution`,
    `RateDistortionTradeOffPoints` (one H.264 point), `MicrophoneCapabilities`
    (one channel, OPUS), `SnapshotCapabilities` (one JPEG entry). The
    values: connectedhomeip's camera app is the existing caller — read
    `examples/camera-app/linux/src/camera-device.cpp` (or the file that
    builds the AvStream init arguments; `grep -rn 'kMaxEncodedPixelRate\|
    VideoSensorParams\|RateDistortion' examples/camera-app/linux/src` in
    `../connectedhomeip`, checkout 1dc3b69ed5e) and take its nominal
    values, cited `file:line` in the row's `cites`. If a value is not in
    chip's app, state it as a nominal 1920×1080 at 30 fps and say so in
    the caveat. H.264 is the mandatory codec (data model 1.7,
    `CameraAVStreamManagement.xml:173-185`, already cited in
    `../go-ha2fabric/docs/roadmap.md` §R6).
  - `webrtctransportprovider`: `ports: host`.
  - No OnOff: the Camera type lists none (`camera_gen.go:34-71`); the
    ON_OFF feature stays unmapped (caveat).
- Caveat on the row: the declaration is nominal — Home Assistant reports no
  codec, resolution or frame rate (`camera/__init__.py:742-755`); the host
  relays the offer and answer and does not read the stream.
- Second row `unmapped`, `when: {}`, verdict `"unmappable: a camera without
  the STREAM feature has no live stream; a SnapshotCamera row is a later
  wave"`.
- The row-count guard (`TestDeviceClassesHaveOneRow` or its kin) must keep
  passing; camera has no device class.

## §4 Tests

- rowfile: `ports: host` accepted; `ports: other` rejected with row id and
  cluster; `ports` on a cluster without a materializer rejected by the
  generator (an `internal/gen` test on a synthetic row file).
- Root package: a fake host type implementing all five interfaces (zero
  methods bodies); `Map` of a camera entity with `supported_features: 2`
  and `Options{Ports: ...}` yields one endpoint of type Camera whose
  Declarations carry the ports (assert through `device.New` building a
  `*device.Camera` whose `Server(wpdef.ClusterID)` is a `*camera.Provider`
  and `Server(csdef.ClusterID)` a `*camera.AvStream`; the test may import
  `cluster/camera`). Without `Options.Ports`, and with a host lacking
  `AudioStreamPort`: no endpoint, one verdict with the text of §2. A camera
  with `supported_features: 1` (ON_OFF only): the `unmapped` verdict.
- Golden/keys tests regenerate as the repo's pattern prescribes.

## §5 Docs and release

- `docs/by_design.md`: one paragraph "Camera: a nominal stream declaration"
  (the HA citation, chip's app as the source of the values, one sentence of
  reasoning), one paragraph "Camera without STREAM is unmapped".
- `README.md` (row format: the `ports` key; `Options.Ports`), `CLAUDE.md`
  state line → v0.4.0, `changelog.md` `## [0.4.0] - 2026-10-31` with Added
  (`ports: host`, `Options.Ports`, the camera row) and the `[Unreleased]`
  entry of the go-fabric bump moved under it.
- Commit the brief as the first commit (`notes/briefs/`), then small
  commits per section. Guards: `make generate`, `make ci` (background,
  summary log; read the log only on failure with grep).
- Do not tag; the owner tags `v0.4.0` after the merge.

## §6 Report (hand back, fixed shape)

Per section the commits; guards with HEAD; differences from the brief;
decisions taken; left out; questions; `by_design` candidates. Name the
exact exported API (`Options.Ports` signature, the materializer names, the
verdict text) so the consumer brief can rely on it.
