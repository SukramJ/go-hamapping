// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 SukramJ.

package hamapping_test

import (
	"testing"

	keypaddef "github.com/SukramJ/go-fabric/cluster/spec/keypadinput"
	levelcontroldef "github.com/SukramJ/go-fabric/cluster/spec/levelcontrol"
	playbackdef "github.com/SukramJ/go-fabric/cluster/spec/mediaplayback"
	onoffdef "github.com/SukramJ/go-fabric/cluster/spec/onoff"
	"github.com/SukramJ/go-fabric/controller/client"
	"github.com/SukramJ/go-fabric/device"
	"github.com/SukramJ/go-fabric/eligibility"
	"github.com/SukramJ/go-fabric/fabrictest"
	"github.com/SukramJ/go-hamqtt/model"

	"github.com/SukramJ/go-hamapping"
)

func player(key, class string, features ...string) model.Entity {
	return entity(key, "media_player", model.Description{
		DeviceClass: model.DeviceClass(class), Extra: map[string]any{"supported_features": features},
	})
}

func TestMediaPlayerSpeaker(t *testing.T) {
	r := serve(t, player("spk", "speaker", "TURN_ON", "TURN_OFF", "VOLUME_SET"))
	wantType(t, r, "spk", device.SpeakerType(), eligibility.StateMappable)
	r.apply("spk", on(map[string]any{"state": "playing", "volume_level": 0.5}))
	if st, _ := onoffdef.StateFrom(r.values("spk", onoffdef.ClusterID)); !st.OnOff {
		t.Fatal("OnOff false while playing")
	}
	if st, _ := levelcontroldef.StateFrom(r.values("spk", levelcontroldef.ClusterID)); st.CurrentLevel == nil || *st.CurrentLevel != 127 {
		t.Fatalf("CurrentLevel %v, want 127", st.CurrentLevel)
	}
	r.invoke("spk", onoffdef.ClusterID, onoffdef.CmdOff, nil)
	expect(t, r.command(), hamapping.MethodTurnOff, "OFF")
	r.invoke("spk", onoffdef.ClusterID, onoffdef.CmdToggle, nil)
	expect(t, r.command(), hamapping.MethodToggle, "TOGGLE")
	r.invoke("spk", levelcontroldef.ClusterID, levelcontroldef.CmdMoveToLevel, levelcontroldef.MoveToLevelRequest{Level: 200})
	expect(t, r.command(), hamapping.MethodVolumeLevel, "200")
	r.apply("spk", on(map[string]any{"state": "standby", "volume_level": 0.5}))
	if st, _ := onoffdef.StateFrom(r.values("spk", onoffdef.ClusterID)); st.OnOff {
		t.Fatal("OnOff true in standby")
	}
	r.noMore()
}

func TestMediaPlayerMute(t *testing.T) {
	r := serve(t, player("spk", "", "VOLUME_MUTE", "VOLUME_SET"))
	wantType(t, r, "spk", device.SpeakerType(), eligibility.StateMappable)
	r.apply("spk", on(map[string]any{"state": "playing", "is_volume_muted": true, "volume_level": 0.2}))
	if st, _ := onoffdef.StateFrom(r.values("spk", onoffdef.ClusterID)); st.OnOff {
		t.Fatal("OnOff true while muted")
	}
	r.invoke("spk", onoffdef.ClusterID, onoffdef.CmdOn, nil)
	expect(t, r.command(), hamapping.MethodVolumeMute, "false")
	r.invoke("spk", onoffdef.ClusterID, onoffdef.CmdOff, nil)
	expect(t, r.command(), hamapping.MethodVolumeMute, "true")
	// No service toggles mute: Toggle branches on the plug's OnOff ("not
	// muted") and calls volume_mute with the inverse (if: {state: OnOff}).
	r.invoke("spk", onoffdef.ClusterID, onoffdef.CmdToggle, nil)
	expect(t, r.command(), hamapping.MethodVolumeMute, "false")
	r.apply("spk", on(map[string]any{"state": "playing", "is_volume_muted": false, "volume_level": 0.2}))
	if st, _ := onoffdef.StateFrom(r.values("spk", onoffdef.ClusterID)); !st.OnOff {
		t.Fatal("OnOff false while unmuted")
	}
	r.invoke("spk", onoffdef.ClusterID, onoffdef.CmdToggle, nil)
	expect(t, r.command(), hamapping.MethodVolumeMute, "true")
	r.noMore()
}

func TestMediaPlayerPredicates(t *testing.T) {
	for _, c := range []struct {
		features []string
		row      string
		state    eligibility.State
	}{
		{[]string{"VOLUME_SET"}, "media_player/speaker_volume", eligibility.StatePartial},
		{[]string{"PLAY", "PAUSE"}, "media_player/unmapped", eligibility.StateUnmappable},
	} {
		_, v := hamapping.Map(testDevice, []model.Entity{player("p", "receiver", c.features...)})
		if v[0].Row != c.row || v[0].State != c.state {
			t.Errorf("%v: %+v", c.features, v[0])
		}
	}
}

func TestMediaPlayerTV(t *testing.T) {
	r := serve(t, player("tv", "tv", "TURN_ON", "TURN_OFF", "PLAY", "PAUSE", "STOP", "VOLUME_STEP", "VOLUME_MUTE"))
	wantType(t, r, "tv", device.BasicVideoPlayerType(), eligibility.StatePartial)
	r.apply("tv", on(map[string]any{"state": "paused"}))
	if st, _ := playbackdef.StateFrom(r.values("tv", playbackdef.ClusterID)); st.CurrentState != playbackdef.PlaybackStatePaused {
		t.Fatalf("CurrentState %v", st.CurrentState)
	}
	r.invoke("tv", playbackdef.ClusterID, playbackdef.CmdPlay, nil)
	expect(t, r.command(), hamapping.MethodMediaPlay, "PLAY")
	r.invoke("tv", playbackdef.ClusterID, playbackdef.CmdStop, nil)
	expect(t, r.command(), hamapping.MethodMediaStop, "STOP")
	r.invoke("tv", onoffdef.ClusterID, onoffdef.CmdOn, nil)
	expect(t, r.command(), hamapping.MethodTurnOn, "ON")
	r.invoke("tv", onoffdef.ClusterID, onoffdef.CmdToggle, nil)
	expect(t, r.command(), hamapping.MethodToggle, "TOGGLE")
	res, err := fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep("tv"), keypaddef.ClusterID, keypaddef.CmdSendKey),
		keypaddef.SendKeyRequest{KeyCode: keypaddef.CecKeyCodeVolumeUp})
	if err != nil {
		t.Fatal(err)
	}
	expect(t, r.command(), hamapping.MethodKey, "65")
	if got := sendKeyStatus(t, res); got != keypaddef.StatusSuccess {
		t.Fatalf("VolumeUp: %v", got)
	}
	res, err = fabrictest.Invoke(r.ctx, r.pair, client.CommandAt(r.ep("tv"), keypaddef.ClusterID, keypaddef.CmdSendKey),
		keypaddef.SendKeyRequest{KeyCode: keypaddef.CecKeyCodeSelect})
	if err != nil {
		t.Fatal(err)
	}
	if got := sendKeyStatus(t, res); got != keypaddef.StatusUnsupportedKey {
		t.Fatalf("Select: %v, want UnsupportedKey", got)
	}
	r.noMore()
}

func sendKeyStatus(t *testing.T, res fabrictest.Result) keypaddef.StatusEnum {
	t.Helper()
	var resp keypaddef.SendKeyResponse
	if err := resp.DecodeTLV(res.Fields); err != nil {
		t.Fatal(err)
	}
	return resp.Status
}
