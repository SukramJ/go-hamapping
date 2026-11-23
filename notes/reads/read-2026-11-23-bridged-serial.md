# Read — bridged SerialNumber shared by several endpoints of one HA device (issue #37; Fable read, 2026-11-23; verify before acting)
Pins: matter-hub 664575a (`$MH` = packages/backend/src), go-fabric v0.25.0 module cache (`$GF`), HA core 2026.10.3, go-hamapping working tree (go.mod:9 go-fabric v0.25.0, :7 go-hamqtt v0.40.0).

## Verdict

State the device's serial on **every** endpoint of that device, cut to 32 bytes; state nothing when it is empty. Do not put it on one endpoint only, do not derive or decorate it.

- Nothing asks for a serial that is unique per endpoint: uniqueness is UniqueId's job, which go-fabric's assembler owns and the host may not state (`$GF/device/bridged_basic_information.go:21-25`). The only cross-check in matter.js is "serial != uniqueId" (below).
- It is a fact of the physical device, and every endpoint here is a part of that device. HA itself treats it so: one serial fans out over several device entries (core device_registry.py:787-789).
- matter-hub does exactly this when its registry-serial flag is on (below). "One only" would need an arbitrary choice of entity that moves when entities are added, removed or filtered, on an attribute of fixed quality.
- A controller that groups endpoints by serial would group the parts of one device — the true relation.

## 1 matter-hub 664575a

- States `serialNumber` on every bridged endpoint, always: `$MH/matter/behaviors/basic-information-server.ts:96` (the class extends matter.js BridgedDeviceBasicInformationServer, :5).
- Source order, :58-64: `mapping.customSerialNumber` > `device.serial_number` only with feature flag `useHaRegistrySerial` > `hash(maxRawLen, anchor)`. `anchor` = identityAnchor or entity_id (:40). `hash` (:128-136) is the anchor text itself, cut to the byte bound with a 4-hex md5 tail when too long — not a digest. So the default serial is per entity and unique.
- Flag `useHaRegistrySerial`: default false, "because changing serialNumber after commissioning can confuse controllers" (packages/common/src/bridge-data.ts:102-110, schemas/bridge-config-schema.ts:292-300). The stated reason is the change, not duplicates.
- Several entities of one device, flag on: each entity endpoint reads the same `entity.deviceRegistry.serial_number` (:36, :59); no dedup, no per-entity suffix. `serialNumberSuffix` (:53-57, :65-68) is one value per bridge, appended to all. Server mode: same order (`$MH/matter/endpoints/server-mode-server-node.ts:136-145`).
- One exception: synthetic vacuum area switches drop both serial sources, "The vacuum's serial belongs to the vacuum alone" (`$MH/matter/endpoints/legacy/vacuum/vacuum-area-switch.ts:41-54`). They are invented siblings, not entities of the device; go-hamapping has no such case in this scope.
- Bound: 32 UTF-8 bytes, cut on a character with "..." (`$MH/utils/trim-to-length.ts:11-16`). Registry changes are re-applied live although the attribute is fixed (`$MH/matter/endpoints/update-entity-state.ts:68-73`).

## 2 Matter facts via go-fabric v0.25.0

- BridgedDeviceBasicInformation SerialNumber 0x000F: string, conformance O, R V, quality Fixed, constraint "max 32" (`$GF/cluster/spec/bridgeddevicebasicinformation/definition_gen.go:45`, :556-562). The bound counts bytes: `n = len(x)` (`$GF/cluster/spec/write.go:405, 833-835`); the test names "32 bytes" (`$GF/device/bridged_info_constraint_test.go:35-39`).
- No uniqueness rule in the definition or the assembler; no check across endpoints found (grep serial over `$GF/endpoint`, `$GF/device`, `$GF/cluster/core`). UniqueId 0x0012 is the separate, mandatory-here identifier (`$GF/cluster/core/bridged_device_basic_information.go:230-231`).
- Served only when stated, nothing derived (`$GF/endpoint/spec.go:62-70`, `$GF/endpoint/materialize.go:146-150`, :591-592). Reason noted there: matter.js warns "uniqueId and serialNumber shall not be the same" (`$GF/cluster/core/bridged_device_basic_information.go:224-228`; matter.js checkout v0.18.0-90, packages/node/src/behaviors/basic-information/basic-information-validators.ts:42-44 — a log warning, not a refusal).
- Over-long is refused, not cut: `checkBridgedInfoValue` → ErrConfiguration (`$GF/device/bridged_basic_information.go:68-73`; test :25).
- `device.Options` has **no** serial field: StableKey, Name, Vendor, DeviceAddress, ChannelAddress, Logger, Conditions, DeviceTypes, Tags, Surface (`$GF/device/options.go:29-71`). The only way is the typed declaration, Initial only (`$GF/device/bridged_basic_information.go:27-58`; Declare `…/server_gen.go:455-458`, Patch.SerialNumber `*string` :74):

```go
import bdbidef "thingfabric.dev/go-fabric/cluster/spec/bridgeddevicebasicinformation"

sn := cut(p.Device.SerialNumber, 32)
device.New(ep.Type, ep.Options, append(ep.Declarations, bdbidef.Declare{Initial: bdbidef.Patch{SerialNumber: &sn}})...)
```

- A cluster declared twice on one device is refused (`$GF/device/options.go:233`, :245); Options.Vendor beside a declared VendorName too (`bridged_basic_information.go:93-95`) — so keep Vendor in Options and declare SerialNumber alone.

## 3 HA core 2026.10.3, homeassistant/helpers/device_registry.py

- `serial_number: str | None`, default None, on DeviceEntry (:537; TypedDicts :145, :288); added 2023.11, older stores migrate to None (:1004-1008). Free text of the integration: no length bound, no uniqueness check seen.
- "serial_number … describes the physical device and is consistent across a composite's splits, so it fans out like sw_version" (:787-789).

## 4 go-hamapping fit

- `model.Device.SerialNumber string` exists (go-hamqtt v0.40.0 model/device.go:178); no go-hamapping .go file reads it yet (grep SerialNumber: none).
- `options` (plan.go:328-364) returns device.Options only — the serial cannot go there. `Endpoint.Declarations` (plan.go:51) comes from `declarations(b)` per row cluster (:312-320), set in `endpoint` (:300-306); riders append theirs to the primary's list (:208, :216).
- So: append the one `bdbidef.Declare` in `Plan.endpoint` (once per endpoint), not in `declarations(b)` — a rider would declare it a second time and Build (:377) would fail with "declared twice". Bound constant beside `maxVendor` (:326), same `cut` (options.go:67-75).
- Not checked: whether an Initial with an empty or whitespace-only string is refused; the issue's "empty → nothing stated" avoids it (test `strings.TrimSpace` if HA can deliver blanks; matter-hub treats blank as absent, trim-to-length.ts:8).
- Caution from matter-hub's flag text: for bridges already commissioned, a fixed attribute appears where none was served. It is an addition, not a change of value; changelog-worthy.
