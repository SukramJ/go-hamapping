// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping

import (
	"fmt"

	cameraavstreammanagementdef "github.com/SukramJ/go-fabric/cluster/spec/cameraavstreammanagement"
	webrtctransportproviderdef "github.com/SukramJ/go-fabric/cluster/spec/webrtctransportprovider"
	"github.com/SukramJ/go-fabric/eligibility"
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

// portsWebRTCProvider sets WebRtcTransportProvider's ProviderPort from
// the host; the kit's server needs it (go-fabric
// device/defaults_camera.go:147-151 at v0.15.5).
func (b *binding) portsWebRTCProvider(d *webrtctransportproviderdef.Declare) bool {
	p, ok := b.host().(webrtctransportproviderdef.ProviderPort)
	if !ok {
		return false
	}
	d.ProviderPort = p
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
