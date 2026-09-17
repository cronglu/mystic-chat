package network

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"mystic-chat/internal/message"
)

func TestMeshNetworkCommunicationAndCrypto(t *testing.T) {
	// 启动两个同一暗号的节点
	room := "武当后山"
	adminKey := "wudang-master-key"

	n1 := NewMeshNetwork("node1", "【武当山·张三丰】", "#00f0ff", 0, 0, false, room, adminKey)
	n2 := NewMeshNetwork("node2", "【武当山·张翠山】", "#39ff14", 0, 0, false, room, adminKey)

	if err := n1.Start(); err != nil {
		t.Fatalf("n1 start error: %v", err)
	}
	defer n1.Stop()

	if err := n2.Start(); err != nil {
		t.Fatalf("n2 start error: %v", err)
	}
	defer n2.Stop()

	var receivedMsg *message.Message
	var wg sync.WaitGroup
	wg.Add(1)

	n2.OnMessageReceived = func(msg *message.Message) {
		receivedMsg = msg
		wg.Done()
	}

	// 手动通过 TCP 种子直连 (验证即使无 UDP 也可直连)
	n1.ConnectPeer(fmt.Sprintf("127.0.0.1:%d", n2.GetPort()))

	// 等待连接握手完成
	time.Sleep(200 * time.Millisecond)

	// n1 广播消息
	testContent := []byte("太极拳经：以柔克刚，借力打力")
	err := n1.BroadcastMessage("msg-101", testContent, 30*time.Second)
	if err != nil {
		t.Fatalf("broadcast message error: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if string(receivedMsg.Content) != string(testContent) {
			t.Fatalf("content mismatch: got %s, want %s", string(receivedMsg.Content), string(testContent))
		}
		if receivedMsg.Sender != "【武当山·张三丰】" {
			t.Fatalf("sender mismatch: got %s", receivedMsg.Sender)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for message to be received by n2")
	}
}

func TestMeshAdminMelt(t *testing.T) {
	adminKey := "secret-close-valley"
	n1 := NewMeshNetwork("node1", "【恶人谷·燕南天】", "#ff007f", 0, 0, false, "", adminKey)
	n2 := NewMeshNetwork("node2", "【恶人谷·江小鱼】", "#ffd700", 0, 0, false, "", adminKey)

	_ = n1.Start()
	defer n1.Stop()
	_ = n2.Start()
	defer n2.Stop()

	var meltTriggered bool
	var wg sync.WaitGroup
	wg.Add(1)

	n2.OnMeltTriggered = func(reason string) {
		meltTriggered = true
		wg.Done()
	}

	n1.ConnectPeer(fmt.Sprintf("127.0.0.1:%d", n2.GetPort()))
	time.Sleep(200 * time.Millisecond)

	// n1 发起闭谷熔断令
	_ = n1.TriggerMelt(adminKey)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !meltTriggered {
			t.Fatalf("expected melt to be triggered on n2")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for melt signal on n2")
	}
}

func TestAutomaticPortFallbackForMultipleInstances(t *testing.T) {
	// 节点 1 占用某端口
	targetPort := 19123
	n1 := NewMeshNetwork("node1", "【武当·宋远桥】", "#00f0ff", targetPort, 0, true, "", "")
	if err := n1.Start(); err != nil {
		t.Fatalf("n1 start failed: %v", err)
	}
	defer n1.Stop()

	if n1.GetPort() != targetPort {
		t.Fatalf("expected n1 port %d, got %d", targetPort, n1.GetPort())
	}

	// 节点 2 在同一台机器使用相同的 targetPort 启动（普通客户端模式）
	n2 := NewMeshNetwork("node2", "【武当·俞莲舟】", "#39ff14", targetPort, 0, true, "", "")
	if err := n2.Start(); err != nil {
		t.Fatalf("expected n2 to start successfully with auto fallback, but got error: %v", err)
	}
	defer n2.Stop()

	// 验证 n2 没有报错，并且自动分配到了另一个空闲端口
	if n2.GetPort() == targetPort {
		t.Fatalf("expected n2 port to be different from %d, but got same", targetPort)
	}
	if n2.GetPort() <= 0 {
		t.Fatalf("expected n2 port to be valid positive port, got %d", n2.GetPort())
	}
	t.Logf("Successfully verified multi-instance port fallback: n1 on %d, n2 automatically on %d", n1.GetPort(), n2.GetPort())
}
