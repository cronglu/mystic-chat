package network

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPresenceTrackingThroughRelay(t *testing.T) {
	// 1. 模拟启动烽火台中继服务器 (端口动态分配, 纯 TCP 禁用 UDP)
	relay := NewMeshNetwork("beacon-relay", "【终南山·烽火台】", "#ffd700", 0, 0, true, "", "")
	relay.SetRelayMode(true)
	if err := relay.Start(); err != nil {
		t.Fatalf("failed to start relay: %v", err)
	}
	defer relay.Stop()

	relayAddr := fmt.Sprintf("127.0.0.1:%d", relay.GetPort())

	// 2. Client 1 (Alice - 张无忌)
	alice := NewMeshNetwork("alice-id", "【光明顶·张无忌】", "#00f0ff", 0, 0, true, "", "")
	if err := alice.Start(); err != nil {
		t.Fatalf("failed to start alice: %v", err)
	}
	defer alice.Stop()

	var alicePeerCount int
	var aliceMu sync.Mutex
	var aliceSawBobJoin bool
	var aliceSawBobLeave bool

	alice.OnPeerChange = func(count int, peers []PeerSummary) {
		aliceMu.Lock()
		alicePeerCount = count
		aliceMu.Unlock()
	}
	alice.OnPeerEvent = func(action string, nick string, color string) {
		aliceMu.Lock()
		if action == "join" && nick == "【黑木崖·令狐冲】" {
			aliceSawBobJoin = true
		}
		if action == "leave" && nick == "【黑木崖·令狐冲】" {
			aliceSawBobLeave = true
		}
		aliceMu.Unlock()
	}

	// Alice 连入烽火台
	alice.ConnectPeer(relayAddr)
	time.Sleep(300 * time.Millisecond)

	aliceMu.Lock()
	if alicePeerCount != 1 {
		t.Fatalf("expected alice initial human peer count = 1, got %d", alicePeerCount)
	}
	aliceMu.Unlock()

	// 3. Client 2 (Bob - 令狐冲) 连入烽火台
	bob := NewMeshNetwork("bob-id", "【黑木崖·令狐冲】", "#39ff14", 0, 0, true, "", "")
	if err := bob.Start(); err != nil {
		t.Fatalf("failed to start bob: %v", err)
	}

	bob.ConnectPeer(relayAddr)

	// 等待花名册同步给 Alice
	time.Sleep(500 * time.Millisecond)

	aliceMu.Lock()
	if alicePeerCount != 2 {
		t.Fatalf("expected alice peer count to increment to 2 when Bob joins, got %d", alicePeerCount)
	}
	if !aliceSawBobJoin {
		t.Fatalf("expected alice to see Bob join event")
	}
	aliceMu.Unlock()

	// 4. Bob 退出/断开连接
	bob.Stop()

	// 等待下线广播与计数递减
	time.Sleep(500 * time.Millisecond)

	aliceMu.Lock()
	if alicePeerCount != 1 {
		t.Fatalf("expected alice peer count to decrement to 1 when Bob leaves, got %d", alicePeerCount)
	}
	if !aliceSawBobLeave {
		t.Fatalf("expected alice to see Bob leave event")
	}
	aliceMu.Unlock()

	t.Logf("Presence tracking test passed: count dynamically updated 1 -> 2 -> 1, join/leave events fired!")
}
