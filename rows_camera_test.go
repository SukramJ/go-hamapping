// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"context"
	"testing"

	"github.com/SukramJ/go-fabric/cluster/camera"
	csdef "github.com/SukramJ/go-fabric/cluster/spec/cameraavstreammanagement"
	wpdef "github.com/SukramJ/go-fabric/cluster/spec/webrtctransportprovider"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	hacatalog "github.com/SukramJ/go-ha-catalog"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
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
	if len(v) != 1 || v[0].Row != "camera/camera" || v[0].State != eligibility.StatePartial || v[0].DeviceType != uint16(device.CameraType.ID) {
		t.Fatalf("verdicts %+v", v)
	}
	if asked != 1 {
		t.Errorf("the host was asked %d times, want 1", asked)
	}
	if len(p.Endpoints) != 1 || p.Endpoints[0].Type != &device.CameraType || len(p.Endpoints[0].Declarations) != 2 {
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
