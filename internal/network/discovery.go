package network

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	DefaultUDPPort   = 9988
	BeaconInterval   = 2 * time.Second
)

// Beacon UDP 广播发现信标
type Beacon struct {
	NodeID  string `json:"node_id"`
	Nick    string `json:"nick"`
	Color   string `json:"color"`
	TCPPort int    `json:"tcp_port"`
}

// DiscoveryService 负责局域网 UDP 嗅探与广播
type DiscoveryService struct {
	nodeID     string
	nick       string
	color      string
	tcpPort    int
	udpPort    int
	udpConn    *net.UDPConn
	stopChan   chan struct{}
	onDiscover func(remoteIP string, b Beacon)
	mu         sync.Mutex
	running    bool
	isUDPActive bool
}

// NewDiscoveryService 创建 UDP 嗅探服务
func NewDiscoveryService(nodeID, nick, color string, tcpPort int, udpPort int, onDiscover func(remoteIP string, b Beacon)) *DiscoveryService {
	return &DiscoveryService{
		nodeID:     nodeID,
		nick:       nick,
		color:      color,
		tcpPort:    tcpPort,
		udpPort:    udpPort,
		stopChan:   make(chan struct{}),
		onDiscover: onDiscover,
	}
}

// UpdateIdentity 当易容时更新广播身份
func (d *DiscoveryService) UpdateIdentity(nick, color string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nick = nick
	d.color = color
}

// Start 启动 UDP 嗅探监听与心跳广播
func (d *DiscoveryService) Start() error {
	if d == nil || d.udpPort <= 0 {
		return nil
	}
	d.mu.Lock()
	d.running = true
	d.mu.Unlock()

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", d.udpPort))
	if err == nil {
		conn, listenErr := net.ListenUDP("udp4", addr)
		if listenErr == nil {
			d.udpConn = conn
			d.isUDPActive = true
			go d.listenLoop()
		}
	}

	go d.broadcastLoop()
	return nil
}

// Stop 停止 UDP 嗅探
func (d *DiscoveryService) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	close(d.stopChan)
	if d.udpConn != nil {
		d.udpConn.Close()
	}
	d.mu.Unlock()
}

// listenLoop 持续接收广播包
func (d *DiscoveryService) listenLoop() {
	buf := make([]byte, 2048)
	for {
		select {
		case <-d.stopChan:
			return
		default:
		}

		if d.udpConn == nil {
			return
		}

		n, remoteAddr, err := d.udpConn.ReadFrom(buf)
		if err != nil {
			return
		}

		var b Beacon
		if err := json.Unmarshal(buf[:n], &b); err != nil {
			continue
		}

		if b.NodeID == d.nodeID {
			// 忽略自身发送的心跳
			continue
		}

		ip := ""
		if udpAddr, ok := remoteAddr.(*net.UDPAddr); ok {
			ip = udpAddr.IP.String()
		}

		if d.onDiscover != nil && ip != "" {
			d.onDiscover(ip, b)
		}
	}
}

// broadcastLoop 周期性发送广播
func (d *DiscoveryService) broadcastLoop() {
	ticker := time.NewTicker(BeaconInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.sendBeacon()
		}
	}
}

func (d *DiscoveryService) sendBeacon() {
	d.mu.Lock()
	beacon := Beacon{
		NodeID:  d.nodeID,
		Nick:    d.nick,
		Color:   d.color,
		TCPPort: d.tcpPort,
	}
	d.mu.Unlock()

	data, err := json.Marshal(beacon)
	if err != nil {
		return
	}

	// 尝试向 255.255.255.255 及 127.0.0.1 发送广播
	destinations := []string{
		fmt.Sprintf("255.255.255.255:%d", d.udpPort),
		fmt.Sprintf("127.0.0.1:%d", d.udpPort),
	}

	for _, dest := range destinations {
		addr, err := net.ResolveUDPAddr("udp4", dest)
		if err != nil {
			continue
		}

		conn, err := net.DialUDP("udp4", nil, addr)
		if err != nil {
			continue
		}
		_, _ = conn.Write(data)
		_ = conn.Close()
	}
}
