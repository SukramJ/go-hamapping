# Read map R6 — go-hamapping plans, declarations and ports; go-hamqtt's model for a camera (Sonnet read, 2026-10-31; a map: file:line for every fact, no judgement; verify before acting)

Provenance. go-hamapping: /Users/markus/Documents/GitHub/go-hamapping, HEAD d4624d1 ("chore(deps): go-fabric v0.15.5"); go.mod:6-8 pins go-fabric v0.15.5, go-ha-catalog v0.3.2-0.20261024093528-d9582e022b8f, go-hamqtt v0.39.0. Line numbers are of that checkout (they differ by a few lines from `read-r0-rowformat.md`, which was read on another HEAD). go-hamqtt: `~/go/pkg/mod/github.com/!sukram!j/go-hamqtt@v0.39.0/` (below `$HQ`). go-ha-catalog: `.../go-ha-catalog@v0.3.2-0.20261024093528-d9582e022b8f/` (below `$HC`). go-fabric v0.15.5 (`$GF`, `~/go/pkg/mod/github.com/!sukram!j/go-fabric@v0.15.5/`) was read only where a port is named (marked "outside the assigned sources"). go-ha2fabric: /Users/markus/Documents/GitHub/go-ha2fabric, branch chore/go-fabric-0.15.5, HEAD 24bad8d. Of `notes/` only `read-r0-rowformat.md` was read (headings and sections 1-3); one incidental `grep -rn -i camera notes` hit list was seen and is not used.

## Q1 Row schema (internal/rowfile/rowfile.go)

- Parser: `dec.KnownFields(true)` rowfile.go:289, so a key outside the structs below is an error.
- `File` rowfile.go:23-31: `platform`, `cites`, `rows`. `Row` :34-53: `id`, `when`, `type`, `verdict`, `ride`, `caveats`, `cites`, `clusters`, `ride_measurement`, `parts`. `Part` :68 = `type`, `clusters`. `When` :81-88: `device_class`, `unit`, `state_class`, `features`, `extra` (map of Set), `options`.
- The cluster entry, verbatim (rowfile.go:100-116 region; tags as written):
```go
type Cluster struct {
	Cluster  string    `yaml:"cluster"`
	Features []Feature `yaml:"features"`
	Initial  []Attr    `yaml:"initial"`
	// StartNull are the attributes that start null (unknown).
	StartNull  []string  `yaml:"start_null"`
	Attributes []Attr    `yaml:"attributes"`
	Commands   []Command `yaml:"commands"`
	// Events emit the cluster's events when a value of the entity
	// changes.
	Events []Event `yaml:"events"`
	// Options materialize a list of the entity into the cluster's
	// attributes and its command (a cluster with a materializer).
	Options *Options `yaml:"options"`
}
```
  So the keys of a cluster entry are exactly: `cluster`, `features`, `initial`, `start_null`, `attributes`, `commands`, `events`, `options`. There is no key `handlers` (the word "handlers" appears only as the generator function `handlers`, gen.go:1093, which emits the `handle` commands) and no key for a port, a hook or a name of a host interface: not found in rowfile.go (searched by reading the structs :23-210).
- `Options` rowfile.go:118-: `List` ("desc.options" or "extra.<key>"), `Current` ("state", "state.<key>", "extra.<key>"), `Method` (yaml `list`, `current`, `method`; the payload is the chosen string).
- `Event` rowfile.go:132-: `On` ("state", "state.<key>", "extra.<key>"), `Emit` (events, in order).
- `Command` rowfile.go:206-: exactly one of `handle` (command name) | `write` []attr | `change` []attr; plus `unless`, `if`, `when`, `method`, `raw` | `json`, `derive`. `Attr`: `attribute`, inline `Value`, `when`, `state_in`, `literal`. `Value` rowfile.go:174-: `from`, `field`, `const`, `units`, `default_unit`, `offset`, `scale`, `add`, `func`, `min`, `max`, `digits`, `per`, `times`, `enum`, `catalog`.
- Row-level fields a camera row would meet: `ride: primary` and `ride_measurement` (a go-fabric measurement source, class from `MeasurementClasses` = `{"battery": "MeasurementBattery"}` rowfile.go ~:64 per read-r0 :64), `parts` (component devices; a part's clusters carry features and constant `initial` only; read-r0-rowformat.md:section 1 citing rowfile.go:347, :351 — not re-read).
- Rows existing for a camera platform: none. `ls rows/` lists binary_sensor, button, climate, cover, event, fan, light, lock, media_player, number, select, sensor, switch, text, vacuum, valve, water_heater (no camera.yaml). `grep -rn -i camera` over *.go, *.yaml, *.md outside notes/: one hit, changelog.md:11 ("go-fabric `v0.15.5`: the camera's signals to the controller's requestor").
- Consequence in code: `find` returns nil when no row of the platform matches (plan.go:183-190), and `verdict` then says `no row for platform %s` (plan.go:194-196).

## Q2 Generated form and materializers

- The gate of `<cluster>def` packages: `defs` map internal/gen/defs.go (the `newDef[Declare,Patch,State,Server](Name, ClusterID, pkg)` list, defs.go:~64-102); comment "A row naming another cluster fails generation until its package is listed here" (defs.go, above `var defs`). 36 entries. The list holds: airquality; carbondioxide/carbonmonoxide/pm1/pm25/pm10/tvoc/nitrogendioxide/ozone/radon concentration measurements; flowmeasurement; soilmeasurement; switchcluster; rvcrunmode; rvcoperationalstate; valveconfigurationandcontrol; mediaplayback; keypadinput; booleanstate; colorcontrol; doorlock; electricalenergymeasurement; electricalpowermeasurement; fancontrol; illuminancemeasurement; levelcontrol; modeselect; occupancysensing; onoff; powersource; powertopology; pressuremeasurement; relativehumiditymeasurement; smokecoalarm; temperaturemeasurement; thermostat; windowcovering. Not in it: cameraavstreammanagement, webrtctransportprovider, webrtctransportrequestor, zonemanagement, chime, pushavstreamtransport, tlscertificatemanagement, tlsclientmanagement (grep of the `*def` imports defs.go:1-45; these packages exist in `$GF/cluster/spec/`).
- `def` struct (defs.go:~50-60): `name`, `id`, `path`, `alias`, `declare`, `patch`, `state`, `server` reflect types; `newDef` takes `D, P, S, V` type parameters. The generator reads field names from `Declare`/`Patch`/`State` by reflection (`field` gen.go:666).
- Device types: no registry in hamapping; `deviceType(name)` searches `device.Types()` (gen.go:237) and an unknown name errors `device type %q: go-fabric's device package has none` (gen.go:285). go-fabric has `device.CameraType` (`$GF/device/camera_gen.go:26`, outside the assigned sources).
- Entry point: `GenerateFiles` gen.go:168; `gen.row` gen.go:257; `gen.clusters` :324; `gen.cluster` gen.go:679-723 emits, per cluster entry, a `Cluster{...}` literal in this order: `Name`, `ID`, `Features: []Feature{...}`, `Attributes`, `Initial`, `Commands` (gen.go:679-696), then `declare:` (g.declare, :964), then `Options: &OptionList{List, Current, Method}` (gen.go:701-703), then `apply:` when attributes or options (gen.go:704-708), `watch:` when a command has `change` (:709-713), `events`/`fire` when events (:714-718).
- Shape of the generated `Declare` literal: not a literal but a closure. `g.declare` (gen.go:964-1020) writes
  `declare: func(b *binding, ci int) device.Declaration { var d <alias>.Declare; <body> return d },` (gen.go:~1018 `fmt.Fprintf(w, "declare: func(b *binding, ci int) ...")`). The body sets `d.Features |= ...` (gen.go:~975-985), `d.Initial.<Attr> = ...` (:986-1004), `d.Initial.Null = append(...)` for `start_null` (:1005-1008 region), the handlers (`g.handlers` :1093), `OnWrite` (`g.onWrite` :1193), and, for `options`, `b.options<Name>(&d, ci)` (gen.go:1009-1015, the `Fprintf` at :1014).
- Generated example (the only materializer row), rows_gen.go:2917-2940 (`select/mode_select`): `declare: func(b *binding, ci int) device.Declaration { var d modeselectdef.Declare; b.optionsModeSelect(&d, ci); return d }`, `Options: &OptionList{List: "desc.options", Current: "state", Method: "select_option"}`, and `apply:` calling `n += b.optionsModeSelectApply(&p, &st, ci)` then `modeselectdef.Wrap(b.server(modeselectdef.ClusterID)).Set(p)` (rows_gen.go:2929-2938). The generator side of the apply call: gen.go:1240-1241 (`n += b.options%sApply(&p, &st, ci)`), the `Wrap(...).Set(p)` tail gen.go:~1244.
- The materializer table: `var materializers = map[string]string{"ModeSelect": "ModeSelect"}` gen.go:1297-1299 (comment gen.go:1293-1296: "clusters whose options a Go materializer of the root package turns into attributes and a command handler"). A cluster with `options` but no entry fails with `options: %s has no materializer` (gen.go:1011-1013 region; test gen_test.go:72 `"no materializer"`).
- The materializer's signature and body, options.go:124-148: `func (b *binding) optionsModeSelect(d *modeselectdef.Declare, ci int)`. It fills `d.Initial.SupportedModes`, `.Description`, `.CurrentMode`, `.Null`, and sets the func field `d.HandleChangeToMode = func(_ context.Context, req modeselectdef.ChangeToModeRequest) (struct{}, error) {...}` (options.go:~137-145), which calls `b.send(o.Method, l[req.NewMode])`. Apply side: `func (b *binding) optionsModeSelectApply(p *modeselectdef.Patch, st *model.State, ci int) int` options.go:151-.
- Delivery inside a materializer: `b.send(method, payload string)` options.go:59-61 = `b.plan.deliver(model.Command{Entity: b.entity, Method: method, Payload: []byte(payload)})`. Verdict-time check: `unmaterializable` options.go:~85 and `checkOptions` (modeselect: needs >0 and <=256 options).
- A host port in the generated code today: none. options.go:115-122 states the nearest thing, verbatim: "The kit's server needs a live ModeSource (device/defaults_declared.go:152-160 at v0.15.4); without one the declaration keeps the generated server, which the handler and the Initial serve." That is, hamapping deliberately sets no `ModeSource`; it uses func fields (`HandleChangeToMode`) of the `Declare`.
- Generated command handlers are func fields of the `Declare` (same mechanism): `g.handlers` gen.go:1093-1172, `g.onWrite` gen.go:1193-1232, `b.emit(ci, k, f)` binding.go:228-247 builds `model.Command{Entity, Method, Payload}` at binding.go:237 and calls `b.plan.deliver`.

Outside the assigned sources (go-fabric v0.15.5, shown because it is the port mechanism the question asks about; not verified beyond these lines):
- `<cluster>def.Declare` carries typed port fields next to the func fields. modeselect: `ModeSource ModeSource` (`$GF/cluster/spec/modeselect/server_gen.go:17-18`, interface `ModeSource` :187; Declare struct :205). cameraavstreammanagement `Declare` (`$GF/cluster/spec/cameraavstreammanagement/server_gen.go:634`): `VideoStreamPort`, `AudioStreamPort`, `SnapshotStreamPort`, `SnapshotCapturePort` (:677-684 region; interfaces `VideoStreamPort` :588, `AudioStreamPort` :600); webrtctransportprovider `Declare` (`.../webrtctransportprovider/server_gen.go:232`): `ProviderPort ProviderPort` (:260-261, interface :219); webrtctransportrequestor `Declare` (`.../webrtctransportrequestor/server_gen.go:234`): `RequestorPort RequestorPort` (:257-258, interface :213).
- The kit builds those servers from the ports; a missing port is `notServed(... errNoPort)` (`$GF/device/defaults_declared.go:119-120, 142, 155-156`; camera ones in `$GF/device/defaults_camera.go:61, 71-72, 114, 132-133, 149-150`).
- `device.Declaration` is an interface with `ClusterID() uint32` and `ClusterDeclaration() spec.ClusterDeclaration` (`$GF/device/options.go:76-83`); `<cluster>def.Declare` values are passed as `device.Declaration`.

## Q3 The Plan API and the consumer's hooks (plan.go, row.go, binding.go)

Structs, quoted:
```go
// plan.go:39-60
type Endpoint struct {
	Key string
	Row string
	Type *device.Type
	Options device.Options
	Declarations []device.Declaration
	Features []string
	Bindings []model.Binding
	Riders []string
}
// plan.go:82-91
type Plan struct {
	Device    *model.Device
	Endpoints []Endpoint

	opts     Options
	bindings map[string]*binding
	devices  map[string]*device.Device

	mu    sync.Mutex
	fn    func(model.Command)
	unsub []func()
	built bool
}
// plan.go:97-
type Options struct {
	StableKey func(model.Entity) string
	Name func(model.Entity) (name string, verbatim bool)
}
```
(field comments omitted; `Declarations` comment plan.go:49-50: "the cluster declarations of the row, and of the rows that ride on this endpoint".)
- API: `func Map(dev *model.Device, entities []model.Entity, opts ...Options) (*Plan, []Verdict)` plan.go:127; `(*Plan).Build(_ context.Context) ([]*device.Device, error)` :318; `DeviceFor(key)` :390; `Apply(ctx, key, st model.State) []Patch` :404; `OnChange(fn func(model.Command))` :464; `Close()` :480; `Rows() []Row` plan.go:~491. `Patch` plan.go:63-80 {Key, Cluster, Value any, Reachable *bool, Events []string, Err error}.
- What a plan hands the consumer: `Plan.Endpoints[i].Declarations` is `[]device.Declaration`, i.e. values (the generated `declare` closure is called during `Map`: `endpoint()` plan.go:254-264 sets `Declarations: declarations(b)`; `declarations(b)` plan.go:268-276 calls each `Clusters[ci].declare(b, ci)`). The closures stay unexported in `Row.Clusters[].declare/apply/watch/fire` (row.go:95-98). The `Declare` values hold func fields that close over the `*binding` (e.g. `HandleChangeToMode`, options.go:~137) — closures inside values.
- `Build` reads the plan's exported slice, not a copy: `ep := &p.Endpoints[i]` then `device.New(ep.Type, ep.Options, ep.Declarations...)` plan.go:~326-327. `Endpoints` and `Declarations` are exported and `Map` returns `*Plan`; nothing in plan.go copies or seals them between `Map` and `Build` (`built` flag :89 guards a second `Build` only, plan.go:~321). Whether a consumer is meant to modify them: not stated in plan.go or doc.go (doc.go 26 lines, not read); not found.
- Rider declarations are appended to the primary's `Declarations` in `Map` (plan.go:170 `device.Measurement(r.b.measure)`; :172 `declarations(r.b)...`), i.e. the plan itself already appends declarations to an endpoint after `endpoint()` (the same slice a consumer sees).
- Parts: `Plan.parts` builds `device.New(part.Type, opts, decls...)` plan.go:365 and `d.AddPart(pd)`.
- The consumer's hooks, complete list found in the root package: (1) `Options.StableKey`, `Options.Name` (plan.go:97-, the only `func(...)` fields of `Options`); (2) `Plan.OnChange(fn func(model.Command))` (plan.go:464; doc says fn runs inline in the controller's request and must not block); (3) `Plan.Apply` for state in; (4) `Plan.Build` returns the `*device.Device` values; (5) exported `Plan.Endpoints[i].Declarations`. No `Options` field takes an interface, a port, a factory or a per-entity declaration hook: not found (`grep -rn 'Source\b'` over non-test, non-generated root files: only options.go:121 comment and measure.go:14, :28).
- Nearest existing "host object" mechanism inside hamapping: `measure` (measure.go:14-28), `var _ contract.MeasurementSource = (*measure)(nil)` measure.go:28, attached by `primary.Declarations = append(primary.Declarations, device.Measurement(r.b.measure))` (plan.go:170): a hamapping-owned implementation of a go-fabric source interface, not host-supplied.
- Command path (`model.Command{Method, Payload}`): `Command` struct `$HQ/model/entity.go:76-93` region: fields `Entity Entity`, `Role string`, `Slot Slot`, `Method string`, `Payload []byte` (the type's doc comment: Method is "the named action ... Empty otherwise"). hamapping builds it at binding.go:237 (`Method: cmd.Method`; if `cmd.Method == ""` it sets `Role = model.RoleCommand` and `Slot` from `model.Bind(b.entity, model.RoleCommand)`, binding.go:238-243) and at options.go:60 (Method only). Delivery: `deliver` plan.go:470-477 calls `p.fn` when non-nil. Method constants live in the generated rows_gen.go (`Method...` consts begin rows_gen.go:58-; e.g. `MethodCloseValve = "close_valve"` :58); `methodsSource` gen/keys.go:68-92; test keys_internal_test.go (per read-r0 :17-45).
- Events: `Cluster.Events []Event{On, Emit}` row.go:117-128, `fire func(b *binding, ei int, endpoint uint16) error` row.go:98; `(*binding).event` events.go:30-.. emits through the typed server, endpoint number via `b.target.EndpointNumber()` events.go:57.

## Q4 go-hamqtt v0.39.0 model fields for a camera (`$HQ/model/`)

- `Entity` interface, entity.go:27-40, four methods: `Key() string`, `Platform() hacatalog.Platform`, `Desc() *Description`, `Bindings() []Binding`. `Basic` struct entity.go:45-50: `EntityKey string`, `EntityPlatform hacatalog.Platform`, `Description Description`, `Binds []Binding`; methods `Key/Platform/Desc/Bindings` entity.go:53-62. The doc above `Entity` (entity.go:~20-26) says capabilities are opt-in interfaces: `Commander` (entity.go:~51-57 region: `Command(ctx, cmd Command) error`), `Invoker` (`Methods() []string`), `Suppressor`, `Deriver`, `EntitySource`, `Enricher` (entity.go lines relative: `Command` struct :76-93; the capability interfaces follow).
- `Binding` slot.go:217-226: `Role string`, `Slot Slot`, `Mode BindMode`; `Valid()` slot.go:229.
- `Description` description.go:180- (struct fields, in order): `Name Localized`, `NameKey string`, `NameArgs map[string]string`, `NameNull bool`, `DeviceClass DeviceClass`, `StateClass hacatalog.StateClass`, `Unit Unit`, `Icon string`, `Category hacatalog.EntityCategory`, `Enabled *bool`, `Options *Enum`, `Precision *int`, `Min, Max, Step *float64`, `Availability Availability`, `Optimistic *bool`, `JSONAttributesTopic`, `JSONAttributesTemplate string`, `ValueTemplate string`, `CommandTemplate string`, `Multiplier *float64`, and the last field `Extra map[string]any` (description.go:366; doc :364-365: "Extra carries platform keys the model does not model. Applied last in the discovery pipeline, so it wins over everything — which makes it both the escape hatch and the footgun."). `Options *Enum` = `Enum{Codes []string; Labels map[string]Localized}` device.go:239-242. `DeviceClass` is `type DeviceClass string` description.go:25; `Unit` description.go:34.
- Fields a consumer may fill with arbitrary data: `Description.Extra map[string]any` (description.go:366), `State.Extra map[string]any` (state.go:39), `State.Value any` (state.go:27), `Device.Extra map[string]any` (device.go:187); `Description.JSONAttributesTopic/Template` are strings (carry a topic, not data).
- `State` state.go:23-40: `Value any`, `Available bool`, `Origin Origin`, `ModifiedAt time.Time`, `RefreshedAt time.Time`, `Extra map[string]any` (doc: "Extra carries additional information alongside the value"). Attributes: the model has no attribute type; attributes ride in `State.Extra` (as go-ha2fabric does, Q6) and `Description.Extra`.
- `Device` device.go:169-188: `Identity Identity`, `Name Localized`, `Manufacturer`, `Model`, `ModelID`, `SWVersion`, `HWVersion`, `SerialNumber`, `SuggestedArea`, `ConfigURL string`, `Via *Identity`, `Extra map[string]any`; `UID()` device.go:191.
- Camera hits, `grep -rn -i 'camera\|stream\|image' $HQ/model $HQ/discovery | grep -v _test`: model/ has none relevant (only the word "downstream" at description.go:204, :231); discovery/ has `CameraFields` {`ImageEncoding string json:"image_encoding,omitempty"`} discovery/gen_fields.go:43-46, `"camera": CameraFields{}` gen_fields.go:512, `ImageFields` {ImageEncoding, ImageTopic} gen_fields.go:226-231, `"image": ImageFields{}` :521 (MQTT discovery keys only). No model field for a stream, a snapshot, an image or a video: not found in model/.
- `hacatalog.PlatformCamera Platform = "camera"` `$HC/gen_vocab.go:17` (also :63, :99).

## Q5 go-ha-catalog camera facts (`$HC/data/`)

- `features.json:10-13`: `"camera": { "ON_OFF": 1, "STREAM": 2 }` — names and bits read from the file (ON_OFF = 1, STREAM = 2).
- `mqtt.json:1075` starts `"camera": {` (MQTT platform schema; keys at 1077-1198 in this order: availability, availability_mode, availability_template, availability_topic, default_entity_id, device, enabled_by_default, encoding (default "utf-8"), entity_category, entity_picture, group, icon, image_encoding, json_attributes_template, json_attributes_topic, message_expiry_interval, name, origin, payload_available, payload_not_available, qos, topic, unique_id, visible_by_default); no `supported_features` key under camera (the file's `supported_features` hits are mqtt.json:751 and :6458, outside the camera block); `"camera"` also in `supported_components` mqtt.json:6928. Command, stream and state keys: none in the camera block.
- `device_classes.json`, `snapshot.json`: `grep -n -i camera` — no hit (not found).
- `icons.json:243-249`: camera icon `mdi:video`, state off `mdi:video-off`.
- How go-hamapping decodes the mask: `features(e)` binding.go:101-127 reads `Description.Extra["supported_features"]` (`keyFeature` value.go:31), a list of names or an int mask, decoded with `hacatalog.LoadFeatures()` flags for `string(e.Platform())`, sorted, upper-case names.

## Q6 go-ha2fabric's plan application

- `Map` is called twice in internal/bridge/bridge.go: dry (`Dry`, bridge.go:~613 `hamapping.Map(g.device, g.entities, mapOptions(g.regs, g.names))`) and in `(*Bridge).build` bridge.go:939-941: `plan, mv := hamapping.Map(p.device, p.entities, mapOptions(p.regs, p.names))`. `mapOptions` bridge.go:593-607 sets `StableKey` (registry id) and `Name` (web UI override, verbatim).
- `build`: `devs, err := plan.Build(ctx)` bridge.go:945; on error every non-unmappable verdict becomes unmappable "go-fabric refused the device: ..." (bridge.go:946-955); else `g.devices = devs`, `g.keys` from `plan.Endpoints[i].Options.StableKey` (bridge.go:957-961), then `plan.OnChange(b.onCommand)` bridge.go:962. `group` struct bridge.go:122-134 holds `plan *hamapping.Plan`, `verdicts`, `devices []*device.Device`, `keys []device.SourceKey`.
- Between `Map` and `Build` the bridge does not touch `plan.Endpoints` or `Declarations`: `grep -n 'Declarations' internal/bridge/*.go` (non-test) — no hit. The only `Endpoints` read is bridge.go:959-960.
- Devices reach the node by `b.n.Aggregator().Batch(ctx, func(bt *node.Batch) error {...})` bridge.go:715, with `bt.Add(d)`, `bt.Replace(k, d)`, `drop`, `bt.Forget` (bridge.go:728-751). Not `device.New` directly (that is hamapping's, plan.go:327).
- State: `(*Bridge).apply` bridge.go:965-982: `g.plan.Apply(ctx, id, adapt.State(st))`; errors logged at debug. `stateChanged` bridge.go:~985-1004 re-syncs when the description string changes.
- Commands: `onCommand(c model.Command)` internal/bridge/command.go:25-33 only queues on `b.cmds` (non-blocking; drops with an error log when full); `commandLoop` command.go:36-46; `command` command.go:48-: `adapt.Service(c, cur)` → `b.up.CallService(cctx, sc)` with `commandTimeout = 30s` (command.go:~18); on failure re-applies the state (command.go:~66-71). `Upstream` interface bridge.go:52-62 (the only HA calls: `ListEntities`, `ListDevices`, `ListAreas`, `ListLabels`, `GetStates`, `GetConfig`, `ListExposed`, `CallService`); no method for a camera stream, snapshot or WebRTC: not found.
- Services table: internal/adapt/command.go:28-34 comment, `payloads` (domain → {payload key, service}) command.go:36-50 for switch, fan, cover, lock; `services` (domain → method → service, no data) command.go:52-75 for button, climate, vacuum, valve, media_player; `Service(cmd, cur)` command.go:78-: resolves `services[domain][cmd.Method]` (command.go:92-94), then a `switch` by domain and method (command.go:95-). There is no `camera` domain entry in either table (`grep -n -i camera internal/adapt/*.go internal/bridge/*.go internal/expose/*.go internal/ha/*.go`, non-test: no hit).
- Entity adaptation: `adapt.Entity(in Input) *model.Basic` adapt.go:~107-170: `EntityKey` = entity id, `EntityPlatform` = domain, `Description.Extra = description(domain, attrs)` (adapt.go:124), a state and a command binding (adapt.go:149 `Binds`). `description()` adapt.go:172-228: always `Extra["supported_features"]` from the attribute (adapt.go:176-180); `switch domain` cases light, climate, fan, water_heater only (case lines 182, 197, 213, 218); no camera case; `return ex` adapt.go:228. `adapt.State(st)` adapt.go:232-266: `model.State{Value: st.State, Available: ..., Extra: maps.Clone(st.Attributes)}` — all HA state attributes ride in `State.Extra`.
- The points where a host-implemented port could attach (positions only, no judgement): (a) after `hamapping.Map` and before `plan.Build` in `build` (bridge.go:940-945), where `plan.Endpoints[i].Declarations` is exported and mutable (plan.go:49-51, :326-327); (b) the `mapOptions`/`hamapping.Options` struct (bridge.go:593), which today takes only `StableKey` and `Name` (plan.go:97-); (c) `Upstream` (bridge.go:52) for what the host can ask of HA; (d) `plan.OnChange` (bridge.go:962) for commands. Whether any of these is the intended one: not stated in any source read.

## Summary table of unverified / not found

| Item | Status |
| --- | --- |
| A camera row, a camera `<cluster>def` in `defs`, a port field or interface in hamapping's rows/generator | not found (rows/ has no camera.yaml; defs.go lists no camera/WebRTC package) |
| A `handlers` or port key in the row YAML cluster entry | not found (keys: cluster, features, initial, start_null, attributes, commands, events, options; rowfile.go:100-116) |
| A hook in `hamapping.Options` for a host interface, factory or declaration | not found (only StableKey, Name; plan.go:97-) |
| Documented intent that a consumer may edit `Plan.Endpoints[i].Declarations` before `Build` | not found (exported + read at plan.go:326-327; doc.go not read, docs/ not read) |
| `Declare` port fields of go-fabric's camera clusters (names, line numbers) | read in `$GF` only as listed (outside the assigned sources); the semantics of each port interface's methods were NOT read |
| Which go-fabric device type/cluster set a camera needs, and whether `device.New` accepts a `CameraType` with the declarations | not read (only `device.CameraType` existence, camera_gen.go:26) |
| `Declare` literal line numbers marked `~` (gen.go:~975-1008, ~1018, ~1244; plan.go:~321, ~491; options.go:~85, ~137; defs.go:~50-102; entity.go capability-interface lines; bridge.go ~613, ~985-1004; command.go ~18, ~66) | approximate (read in slices, not each line counted) |
| defs.go entry count "36" | counted by eye from the printed list; not machine-counted |
| Home Assistant's camera attributes and services (stream source, snapshot, `camera.turn_on`, etc.) | not read (home-assistant/core not in the sources); no camera service in go-ha2fabric's tables |
| go-ha-catalog `device_classes.json`/`snapshot.json` camera content | grep for "camera": no hit |
| doc.go, docs/by_design.md, CONCEPT.md, tests | not read |
