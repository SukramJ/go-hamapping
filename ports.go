// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"fmt"
	"time"

	cameraavstreammanagementdef "thingfabric.dev/go-fabric/cluster/spec/cameraavstreammanagement"
	webrtctransportproviderdef "thingfabric.dev/go-fabric/cluster/spec/webrtctransportprovider"
	"thingfabric.dev/go-fabric/eligibility"
)

// The port materializers of a row's `ports: host` clusters: the kit's
// server of such a cluster runs on ports of the consumer's host object
// ([Options.Ports]). The generated declaration calls b.ports<Cluster>,
// which sets the Declare's port fields and reports whether the host
// serves every port the cluster needs; b.unserved records the first
// cluster it does not, and [Map] then gives the entity no endpoint.
//
// A camera must never be built without its ports: a Declare that carries
// some port but lacks a needed one fails the device's construction
// (go-fabric device/device.go:396-400 at v0.15.5), and one without any
// keeps the generated server, whose commands are unimplemented
// (device/device.go:402-410).

// host is the consumer's host object for the entity, asked once per
// binding; nil without [Options.Ports].
func (b *binding) host() any {
	if !b.hostAsked {
		b.hostAsked = true
		if f := b.plan.opts.Ports; f != nil {
			b.hostObj = f(b.entity)
		}
	}
	return b.hostObj
}

// unserved records that the host serves no ports for the row's cluster
// ci; the first such cluster names the verdict.
func (b *binding) unserved(ci int) {
	if b.noPorts == "" {
		b.noPorts = b.row.Clusters[ci].Name
	}
}

// unservedVerdict is the verdict of an entity whose host serves no ports
// for a cluster of its row.
func unservedVerdict(b *binding) Verdict {
	return Verdict{
		Key: b.key, Row: b.row.ID, State: eligibility.StateUnmappable,
		Reason: fmt.Sprintf("%s: the consumer serves no host ports for %s", b.row.ID, b.noPorts),
	}
}

// WebRTCSessionBounds is an optional interface of the consumer's host
// object ([Options.Ports]) for a camera: the session bounds of its
// WebRtcTransportProvider (go-fabric
// cluster/spec/webrtctransportprovider/server_gen.go:316-341 at v0.28.0).
// total bounds every session of the camera, perFabric the sessions of one
// fabric; 0 is no bound. They are copied unchanged into the declaration,
// and only together with the host's ProviderPort: a bound without the port
// fails the build (device/options.go:288-315 counts a non-zero bound as a
// port, device/device.go:492-496 refuses it, at v0.28.0). A negative bound
// fails [Plan.Build] with a [*BuildError] wrapping go-fabric's
// camera.ErrNegativeMaxSessions (device/defaults_camera.go:149-153 at
// v0.28.0); no relation between the two is checked, as go-fabric checks
// none.
type WebRTCSessionBounds interface {
	WebRTCSessionBounds() (total, perFabric int)
}

// WebRTCSessionEviction is an optional interface of the consumer's host
// object ([Options.Ports]) for a camera, beside [WebRTCSessionBounds]: the
// eviction at the session bounds of its WebRtcTransportProvider (go-fabric
// cluster/spec/webrtctransportprovider/server_gen.go:342-366 at v0.28.0).
// idle is Declare.EvictIdle, acrossFabrics Declare.EvictAcrossFabrics;
// 0 and false evict nothing. Both are copied unchanged into the
// declaration, and only together with the host's ProviderPort, as the
// bounds are. No relation to the bounds is checked, as go-fabric checks
// none: idle without a bound, and acrossFabrics without idle and a total
// bound, are inert in the kit (cluster/camera/provider_cap.go:53-55,
// cluster/camera/provider.go:142 at v0.28.0). A negative idle fails
// [Plan.Build] with a [*BuildError] wrapping go-fabric's
// camera.ErrNegativeEvictIdle (device/defaults_camera.go:154-156 at
// v0.28.0). Which idle time and whether fabrics cross is the host's.
type WebRTCSessionEviction interface {
	WebRTCSessionEviction() (idle time.Duration, acrossFabrics bool)
}

// portsWebRTCProvider sets WebRtcTransportProvider's ProviderPort from
// the host, its session bounds when the host also implements
// [WebRTCSessionBounds] and its eviction when it implements
// [WebRTCSessionEviction]; the kit's server needs the port (go-fabric
// device/defaults_camera.go:149-168 at v0.28.0).
func (b *binding) portsWebRTCProvider(d *webrtctransportproviderdef.Declare) bool {
	h := b.host()
	p, ok := h.(webrtctransportproviderdef.ProviderPort)
	if !ok {
		return false
	}
	d.ProviderPort = p
	if sb, ok := h.(WebRTCSessionBounds); ok {
		d.MaxSessions, d.MaxSessionsPerFabric = sb.WebRTCSessionBounds()
	}
	if se, ok := h.(WebRTCSessionEviction); ok {
		d.EvictIdle, d.EvictAcrossFabrics = se.WebRTCSessionEviction()
	}
	return true
}

// portsAvStream sets CameraAvStreamManagement's four stream ports from the
// host and reports false unless it serves all four: the camera row selects
// VDO, ADO and SNP, which need a VideoStreamPort, an AudioStreamPort, a
// SnapshotStreamPort and a SnapshotCapturePort (go-fabric
// cluster/camera/avstream.go:194-198 at v0.15.5).
func (b *binding) portsAvStream(d *cameraavstreammanagementdef.Declare) bool {
	h := b.host()
	video, okV := h.(cameraavstreammanagementdef.VideoStreamPort)
	audio, okA := h.(cameraavstreammanagementdef.AudioStreamPort)
	snapshot, okS := h.(cameraavstreammanagementdef.SnapshotStreamPort)
	capture, okC := h.(cameraavstreammanagementdef.SnapshotCapturePort)
	if !okV || !okA || !okS || !okC {
		return false
	}
	d.VideoStreamPort, d.AudioStreamPort, d.SnapshotStreamPort, d.SnapshotCapturePort = video, audio, snapshot, capture
	return true
}
