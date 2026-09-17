package network

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"mystic-chat/internal/message"
)

func TestBootstrapRelayAndClientMesh(t *testing.T) {
	// 1. 模拟部署在 Linux 服务器上的引导/烽火台中继服务器 (端口动态分配)
	roomSecret := "绝情谷断肠草"
	adminKey := "boss-killswitch-key"

	beaconRelay := NewMeshNetwork("beacon-relay-node", "【终南山·烽火台】", "#ffd700", 0, 0, true, roomSecret, adminKey)
	beaconRelay.SetRelayMode(true)
	if err := beaconRelay.Start(); err != nil {
		t.Fatalf("failed to start bootstrap server: %v", err)
	}
	defer beaconRelay.Stop()

	bootstrapAddr := fmt.Sprintf("127.0.0.1:%d", beaconRelay.GetPort())

	// 2. 模拟客户端 1：位于局域网 A，无 UDP，仅连接引导节点
	client1 := NewMeshNetwork("client-1", "【绝情谷·小龙女】", "#00f0ff", 0, 0, true, roomSecret, adminKey)
	if err := client1.Start(); err != nil {
		t.Fatalf("failed to start client1: %v", err)
	}
	defer client1.Stop()

	// 3. 模拟客户端 2：位于局域网 B，同样无 UDP，仅连接引导节点
	client2 := NewMeshNetwork("client-2", "【终南山·杨过】", "#ff007f", 0, 0, true, roomSecret, adminKey)
	if err := client2.Start(); err != nil {
		t.Fatalf("failed to start client2: %v", err)
	}
	defer client2.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	var receivedByClient2 *message.Message

	client2.OnMessageReceived = func(msg *message.Message) {
		receivedByClient2 = msg
		wg.Done()
	}

	// 双方均连接到引导服务器 (模拟 -peer <server-ip>:port)
	client1.ConnectPeer(bootstrapAddr)
	client2.ConnectPeer(bootstrapAddr)

	// 等待 TLS 握手与 PEX 节点就绪
	time.Sleep(300 * time.Millisecond)

	// Client 1 发送暗号密信
	secretPoem := []byte("十六年后，在此重会。夫妻情深，勿失信约。")
	err := client1.BroadcastMessage("poem-001", secretPoem, 30*time.Second)
	if err != nil {
		t.Fatalf("client1 failed to broadcast message: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if string(receivedByClient2.Content) != string(secretPoem) {
			t.Fatalf("client2 received corrupted content: %s", string(receivedByClient2.Content))
		}
		if receivedByClient2.Sender != "【绝情谷·小龙女】" {
			t.Fatalf("client2 received wrong sender: %s", receivedByClient2.Sender)
		}
		t.Logf("成功通过引导中继接收并解密密信: %s 来自 %s", string(receivedByClient2.Content), receivedByClient2.Sender)
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout: client2 did not receive message via relay")
	}
}
