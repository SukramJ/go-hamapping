// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	hacatalog "sukramj.dev/go-ha-catalog"
	"sukramj.dev/go-hamqtt/model"
	"thingfabric.dev/go-fabric/cluster/camera"
	csdef "thingfabric.dev/go-fabric/cluster/spec/cameraavstreammanagement"
	wpdef "thingfabric.dev/go-fabric/cluster/spec/webrtctransportprovider"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/eligibility"

	"sukramj.dev/go-hamapping"
)

// The host ports of a camera, each its own type so a host can leave one
// out.
type (
	videoPort    struct{}
	audioPort    struct{}
	snapshotPort struct{}
	providerPort struct{}
)

func (videoPort) VideoStreamAllocate(context.Context, csdef.VideoStreamStruct) (uint16, error) {
	return 1, nil
}

func (videoPort) VideoStreamAllocated(context.Context, csdef.VideoStreamStruct, csdef.StreamAllocationAction) {
}

func (videoPort) VideoStreamModify(context.Context, uint16, *bool, *bool) error { return nil }

func (videoPort) VideoStreamDeallocate(context.Context, uint16) error { return nil }

func (audioPort) AudioStreamAllocate(context.Context, csdef.AudioStreamStruct) (uint16, error) {
	return 1, nil
}
func (audioPort) AudioStreamDeallocate(context.Context, uint16) error { return nil }

func (snapshotPort) SnapshotStreamAllocate(context.Context, csdef.SnapshotStreamStruct) (uint16, error) {
	return 1, nil
}
func (snapshotPort) SnapshotStreamModify(context.Context, uint16, *bool, *bool) error { return nil }
func (snapshotPort) SnapshotStreamDeallocate(context.Context, uint16) error           { return nil }
func (snapshotPort) CaptureSnapshot(context.Context, *uint16, csdef.VideoResolutionStruct) (csdef.CaptureSnapshotResponse, error) {
	return csdef.CaptureSnapshotResponse{}, nil
}

func (providerPort) SolicitOffer(context.Context, wpdef.OfferArgs) (wpdef.WebRTCSessionStruct, bool, error) {
	return wpdef.WebRTCSessionStruct{}, false, nil
}

func (providerPort) ProvideOffer(context.Context, wpdef.OfferArgs) (wpdef.WebRTCSessionStruct, error) {
	return wpdef.WebRTCSessionStruct{}, nil
}
func (providerPort) ProvideAnswer(context.Context, uint16, string) error { return nil }
func (providerPort) ProvideIceCandidates(context.Context, uint16, []wpdef.ICECandidateStruct) error {
	return nil
}
func (providerPort) EndSession(context.Context, uint16, uint8) error { return nil }

// cameraHost serves all five ports; noAudioHost lacks the AudioStreamPort.
type (
	cameraHost struct {
		videoPort
		audioPort
		snapshotPort
		providerPort
	}
	noAudioHost struct {
		videoPort
		snapshotPort
		providerPort
	}
)

func cameraEntity(features int) model.Entity {
	return entity("camera.door", hacatalog.PlatformCamera, model.Description{
		Name: model.L("Door"), Extra: map[string]any{"supported_features": features},
	})
}

// TestCamera: a camera with STREAM and a host serving its ports is a
// Camera whose stream clusters are the kit's servers on those ports, the
// host asked once.
func TestCamera(t *testing.T) {
	asked := 0
	ports := hamapping.Options{Ports: func(model.Entity) any { asked++; return cameraHost{} }}
	p, v := hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, ports)
	if len(v) != 1 || v[0].Row != "camera/camera" || v[0].State != eligibility.StatePartial || v[0].DeviceType != uint16(device.CameraType().ID) {
		t.Fatalf("verdicts %+v", v)
	}
	if asked != 1 {
		t.Errorf("the host was asked %d times, want 1", asked)
	}
	if len(p.Endpoints) != 1 || p.Endpoints[0].Type.ID != device.CameraType().ID || len(p.Endpoints[0].Declarations) != 2 {
		t.Fatalf("endpoints %+v", p.Endpoints)
	}
	devs, err := p.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	d := devs[0]
	if _, ok := d.Server(wpdef.ClusterID).(*camera.Provider); !ok {
		t.Errorf("WebRtcTransportProvider is %T, want the kit's *camera.Provider", d.Server(wpdef.ClusterID))
	}
	srv, ok := d.Server(csdef.ClusterID).(*camera.AvStream)
	if !ok {
		t.Fatalf("CameraAvStreamManagement is %T, want the kit's *camera.AvStream", d.Server(csdef.ClusterID))
	}
	st := csdef.Wrap(srv).Get()
	if st.MaxEncodedPixelRate != 248832000 || len(st.SupportedStreamUsages) != 1 || st.SupportedStreamUsages[0] != csdef.StreamUsageLiveView ||
		len(st.RateDistortionTradeOffPoints) != 1 || st.RateDistortionTradeOffPoints[0].Codec != csdef.VideoCodecH264 ||
		len(st.SnapshotCapabilities) != 1 || st.SnapshotCapabilities[0].ImageCodec != csdef.ImageCodecJpeg ||
		st.MicrophoneCapabilities.MaxNumberOfChannels != 1 || st.VideoSensorParams.SensorWidth != 1920 {
		t.Fatalf("AvStream state %+v", st)
	}
}

// TestCameraWithoutHostPorts: without Options.Ports, or with a host that
// lacks a port, the camera gets no endpoint and the verdict names the
// cluster; a camera without STREAM is the unmapped row.
func TestCameraWithoutHostPorts(t *testing.T) {
	const reason = "camera/camera: the consumer serves no host ports for CameraAvStreamManagement"
	for name, opts := range map[string][]hamapping.Options{
		"no Ports":       nil,
		"nil host":       {{Ports: func(model.Entity) any { return nil }}},
		"no audio port":  {{Ports: func(model.Entity) any { return noAudioHost{} }}},
		"no stream port": {{Ports: func(model.Entity) any { return providerPort{} }}},
	} {
		t.Run(name, func(t *testing.T) {
			p, v := hamapping.Map(testDevice, []model.Entity{cameraEntity(3)}, opts...)
			if len(p.Endpoints) != 0 {
				t.Fatalf("endpoints %+v", p.Endpoints)
			}
			if len(v) != 1 || v[0].State != eligibility.StateUnmappable || v[0].Reason != reason || v[0].Row != "camera/camera" {
				t.Fatalf("verdicts %+v", v)
			}
			if p.DeviceFor("camera.door") != nil {
				t.Error("a device for the skipped camera")
			}
		})
	}
	_, v := hamapping.Map(testDevice, []model.Entity{cameraEntity(1)}, hamapping.Options{Ports: func(model.Entity) any { return cameraHost{} }})
	if len(v) != 1 || v[0].Row != "camera/unmapped" || v[0].State != eligibility.StateUnmappable ||
		v[0].Reason != "a camera without the STREAM feature has no live stream; a SnapshotCamera row is a later wave" {
		t.Fatalf("ON_OFF only: %+v", v)
	}
}

// boundedHost serves every camera port and the optional session bounds;
// boundsOnlyHost states the bounds but serves no ProviderPort.
type (
	boundedHost struct {
		cameraHost
		total, perFabric int
	}
	boundsOnlyHost struct {
		videoPort
		audioPort
		snapshotPort
	}
)

func (h boundedHost) WebRTCSessionBounds() (total, perFabric int) { return h.total, h.perFabric }

func (boundsOnlyHost) WebRTCSessionBounds() (total, perFabric int) { return 5, 2 }

var _ hamapping.WebRTCSessionBounds = boundedHost{}

// evictingHost serves every camera port, the session bounds and the
// eviction; evictionOnlyHost serves every camera port and the eviction
// without bounds; evictionNoPortHost states the eviction but serves no
// ProviderPort.
type (
	evictingHost struct {
		boundedHost
		idle   time.Duration
		across bool
	}
	evictionOnlyHost struct {
		cameraHost
		idle   time.Duration
		across bool
	}
	evictionNoPortHost struct {
		videoPort
		audioPort
		snapshotPort
	}
)

func (h evictingHost) WebRTCSessionEviction() (time.Duration, bool) { return h.idle, h.across }

func (h evictionOnlyHost) WebRTCSessionEviction() (time.Duration, bool) { return h.idle, h.across }

func (evictionNoPortHost) WebRTCSessionEviction() (time.Duration, bool) { return time.Minute, true }

var (
	_ hamapping.WebRTCSessionEviction = evictingHost{}
	_ hamapping.WebRTCSessionBounds   = evictingHost{}
)

// providerDeclare is the camera endpoint's WebRtcTransportProvider
// declaration.
func providerDeclare(t *testing.T, p *hamapping.Plan) wpdef.Declare {
	t.Helper()
	if len(p.Endpoints) != 1 {
		t.Fatalf("endpoints %+v", p.Endpoints)
	}
	for _, d := range p.Endpoints[0].Declarations {
		if wd, ok := d.(wpdef.Declare); ok {
			return wd
		}
	}
	t.Fatal("no WebRtcTransportProvider declaration")
	return wpdef.Declare{}
}

// TestCameraSessionBounds: a host implementing WebRTCSessionBounds has its
// bounds copied unchanged into the WebRtcTransportProvider declaration and
// the kit's server builds on them; without the interface both stay 0 (no
// bound); a negative bound fails the build with go-fabric's
// ErrNegativeMaxSessions; bounds without a ProviderPort leave the camera
// unserved, with no build error.
func TestCameraSessionBounds(t *testing.T) {
	opts := func(h any) hamapping.Options {
		return hamapping.Options{Ports: func(model.Entity) any { return h }}
	}
	p, _ := hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(boundedHost{total: 5, perFabric: 2}))
	if d := providerDeclare(t, p); d.MaxSessions != 5 || d.MaxSessionsPerFabric != 2 || d.ProviderPort == nil {
		t.Fatalf("with bounds: MaxSessions %d, MaxSessionsPerFabric %d, port %v", d.MaxSessions, d.MaxSessionsPerFabric, d.ProviderPort)
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatalf("Build with bounds: %v", err)
	}
	p.Close()

	p, _ = hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(cameraHost{}))
	if d := providerDeclare(t, p); d.MaxSessions != 0 || d.MaxSessionsPerFabric != 0 {
		t.Fatalf("without bounds: MaxSessions %d, MaxSessionsPerFabric %d", d.MaxSessions, d.MaxSessionsPerFabric)
	}

	p, _ = hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(boundedHost{total: -1}))
	_, err := p.Build(context.Background())
	var be *hamapping.BuildError
	if !errors.As(err, &be) || !errors.Is(err, camera.ErrNegativeMaxSessions) {
		t.Fatalf("Build with a negative bound: %v, want a *BuildError wrapping camera.ErrNegativeMaxSessions", err)
	}

	p, v := hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(boundsOnlyHost{}))
	if len(p.Endpoints) != 0 || len(v) != 1 || v[0].State != eligibility.StateUnmappable ||
		v[0].Reason != "camera/camera: the consumer serves no host ports for WebRtcTransportProvider" {
		t.Fatalf("bounds without a ProviderPort: endpoints %+v, verdicts %+v", p.Endpoints, v)
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatalf("Build of the unserved camera: %v", err)
	}
}

// TestCameraSessionEviction: a host implementing WebRTCSessionEviction has
// its idle time and its across-fabrics choice copied unchanged into the
// WebRtcTransportProvider declaration, beside the bounds of
// WebRTCSessionBounds or without them, and the kit's server builds on
// them; a negative idle time fails the build with go-fabric's
// ErrNegativeEvictIdle; the eviction without a ProviderPort leaves the
// camera unserved, with no build error.
func TestCameraSessionEviction(t *testing.T) {
	opts := func(h any) hamapping.Options {
		return hamapping.Options{Ports: func(model.Entity) any { return h }}
	}
	host := evictingHost{boundedHost: boundedHost{total: 5, perFabric: 2}, idle: 30 * time.Second, across: true}
	p, _ := hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(host))
	if d := providerDeclare(t, p); d.MaxSessions != 5 || d.MaxSessionsPerFabric != 2 ||
		d.EvictIdle != 30*time.Second || !d.EvictAcrossFabrics || d.ProviderPort == nil {
		t.Fatalf("with bounds and eviction: MaxSessions %d, MaxSessionsPerFabric %d, EvictIdle %v, EvictAcrossFabrics %v, port %v",
			d.MaxSessions, d.MaxSessionsPerFabric, d.EvictIdle, d.EvictAcrossFabrics, d.ProviderPort)
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatalf("Build with bounds and eviction: %v", err)
	}
	p.Close()

	p, _ = hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(evictionOnlyHost{idle: time.Minute, across: true}))
	if d := providerDeclare(t, p); d.MaxSessions != 0 || d.MaxSessionsPerFabric != 0 ||
		d.EvictIdle != time.Minute || !d.EvictAcrossFabrics || d.ProviderPort == nil {
		t.Fatalf("eviction without bounds: MaxSessions %d, MaxSessionsPerFabric %d, EvictIdle %v, EvictAcrossFabrics %v, port %v",
			d.MaxSessions, d.MaxSessionsPerFabric, d.EvictIdle, d.EvictAcrossFabrics, d.ProviderPort)
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatalf("Build with eviction without bounds: %v", err)
	}
	p.Close()

	p, _ = hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(boundedHost{total: 5, perFabric: 2}))
	if d := providerDeclare(t, p); d.EvictIdle != 0 || d.EvictAcrossFabrics {
		t.Fatalf("without eviction: EvictIdle %v, EvictAcrossFabrics %v", d.EvictIdle, d.EvictAcrossFabrics)
	}

	p, _ = hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(evictingHost{boundedHost: boundedHost{total: 5}, idle: -time.Second}))
	_, err := p.Build(context.Background())
	var be *hamapping.BuildError
	if !errors.As(err, &be) || !errors.Is(err, camera.ErrNegativeEvictIdle) {
		t.Fatalf("Build with a negative idle time: %v, want a *BuildError wrapping camera.ErrNegativeEvictIdle", err)
	}

	p, v := hamapping.Map(testDevice, []model.Entity{cameraEntity(2)}, opts(evictionNoPortHost{}))
	if len(p.Endpoints) != 0 || len(v) != 1 || v[0].State != eligibility.StateUnmappable ||
		v[0].Reason != "camera/camera: the consumer serves no host ports for WebRtcTransportProvider" {
		t.Fatalf("eviction without a ProviderPort: endpoints %+v, verdicts %+v", p.Endpoints, v)
	}
	if _, err := p.Build(context.Background()); err != nil {
		t.Fatalf("Build of the unserved camera: %v", err)
	}
}
