package network

import (
	"bufio"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"mystic-chat/internal/message"
)

type peerConn struct {
	nodeID   string
	nick     string
	color    string
	tcpAddr  string
	conn     net.Conn
	writer   *bufio.Writer
	mu       sync.Mutex
	lastSeen time.Time
}

// MeshNetwork P2P 网格通信引擎（TLS 1.3 传输层加密 + 端到端 AES-GCM 双重防护 + 全网花名册同步）
type MeshNetwork struct {
	nodeID            string
	nick              string
	color             string
	tcpListenPort     int
	roomSecret        string
	roomKey           []byte
	adminKey          string
	adminTokenHashHex string
	listener          net.Listener
	tlsConfig         *tls.Config
	discovery         *DiscoveryService
	isRelayMode       bool

	mu          sync.RWMutex
	peers       map[string]*peerConn   // 直接相连的 TCP 连接 (nodeID -> peerConn)
	globalPeers map[string]PeerSummary // 全网已知的在线同门与中继 (nodeID -> PeerSummary)
	knownAddrs  map[string]bool        // 已知节点 TCP 地址池 (PEX)
	seenMsgIDs  map[string]time.Time   // 消息去重
	stopChan    chan struct{}
	running     bool

	OnMessageReceived func(msg *message.Message)
	OnMeltTriggered   func(reason string)
	OnPeerChange      func(count int, peers []PeerSummary)
	OnPeerEvent       func(action string, nick string, color string) // "join", "leave", "reroll"
}

// NewMeshNetwork 创建 P2P 网络实例
func NewMeshNetwork(nodeID, nick, color string, tcpPort int, udpPort int, disableUDP bool, roomSecret string, adminKey string) *MeshNetwork {
	roomKey := DeriveRoomKey(roomSecret)

	var adminHashHex string
	if len(adminKey) > 0 {
		h := DeriveAdminTokenHash(adminKey)
		adminHashHex = hex.EncodeToString(h[:])
	}

	tlsConfig, err := GenerateEphemeralTLSConfig()
	if err != nil {
		// 降级使用空配置
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}

	m := &MeshNetwork{
		nodeID:            nodeID,
		nick:              nick,
		color:             color,
		tcpListenPort:     tcpPort,
		roomSecret:        roomSecret,
		roomKey:           roomKey,
		adminKey:          adminKey,
		adminTokenHashHex: adminHashHex,
		tlsConfig:         tlsConfig,
		peers:             make(map[string]*peerConn),
		globalPeers:       make(map[string]PeerSummary),
		knownAddrs:        make(map[string]bool),
		seenMsgIDs:        make(map[string]time.Time),
		stopChan:          make(chan struct{}),
	}

	// 初始将自身加入 globalPeers
	m.globalPeers[nodeID] = PeerSummary{
		ID:       nodeID,
		Nick:     nick,
		Color:    color,
		IsRelay:  false,
		LastSeen: time.Now().Unix(),
	}

	// 初始化 UDP 嗅探（若启用且端口有效）
	if !disableUDP && udpPort > 0 {
		m.discovery = NewDiscoveryService(nodeID, nick, color, tcpPort, udpPort, func(remoteIP string, b Beacon) {
			remoteAddr := fmt.Sprintf("%s:%d", remoteIP, b.TCPPort)
			m.ConnectPeer(remoteAddr)
		})
	}

	return m
}

// GetNodeID 获取本机节点 ID
func (m *MeshNetwork) GetNodeID() string {
	return m.nodeID
}

// GetHumanCount 获取全网当前在线的实际人类侠客总人数
func (m *MeshNetwork) GetHumanCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, p := range m.globalPeers {
		if !p.IsRelay {
			count++
		}
	}
	if count < 1 && !m.isRelayMode {
		count = 1
	}
	return count
}

// GetGlobalPeers 获取全网已知的在线同门与中继列表
func (m *MeshNetwork) GetGlobalPeers() []PeerSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]PeerSummary, 0, len(m.globalPeers))
	for _, p := range m.globalPeers {
		res = append(res, p)
	}
	return res
}

// IsUDPEnabled 返回当前是否启用了 UDP 广播嗅探
func (m *MeshNetwork) IsUDPEnabled() bool {
	return m.discovery != nil
}

// SetRelayMode 设置是否作为纯无界面烽火台中继/引导节点（适合在 Linux 服务器上部署）
func (m *MeshNetwork) SetRelayMode(relay bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRelayMode = relay
	if self, ok := m.globalPeers[m.nodeID]; ok {
		self.IsRelay = relay
		m.globalPeers[m.nodeID] = self
	}
}

// Start 启动 TLS 1.3 TCP 监听与 P2P 网络
func (m *MeshNetwork) Start() error {
	rawListener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", m.tcpListenPort))
	if err != nil {
		// 若为普通客户端（非指定端口的中继烽火台），遇端口冲突时自动寻找空闲端口，支持同机直接多开！
		if !m.isRelayMode {
			altListener, altErr := net.Listen("tcp", "0.0.0.0:0")
			if altErr == nil {
				rawListener = altListener
				err = nil
			}
		}
		if err != nil {
			return fmt.Errorf("TCP 端口监听失败 :%d : %w", m.tcpListenPort, err)
		}
	}

	// 更新实际端口
	if tcpAddr, ok := rawListener.Addr().(*net.TCPAddr); ok {
		m.tcpListenPort = tcpAddr.Port
	}

	// 强制升级为 TLS 1.3 传输加密
	m.listener = tls.NewListener(rawListener, m.tlsConfig)

	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	go m.acceptLoop()
	go m.cleanupLoop()

	if m.discovery != nil {
		_ = m.discovery.Start()
	}
	return nil
}

// Stop 关闭网络连接与监听
func (m *MeshNetwork) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.stopChan)
	if m.listener != nil {
		m.listener.Close()
	}

	for _, p := range m.peers {
		p.conn.Close()
	}
	m.peers = make(map[string]*peerConn)
	m.mu.Unlock()

	if m.discovery != nil {
		m.discovery.Stop()
	}
}

// UpdateIdentity 易容术：更新自身身份并向全网广播问候
func (m *MeshNetwork) UpdateIdentity(newNick, newColor string) {
	m.mu.Lock()
	m.nick = newNick
	m.color = newColor
	if self, ok := m.globalPeers[m.nodeID]; ok {
		self.Nick = newNick
		self.Color = newColor
		self.LastSeen = time.Now().Unix()
		m.globalPeers[m.nodeID] = self
	}
	m.mu.Unlock()

	if m.discovery != nil {
		m.discovery.UpdateIdentity(newNick, newColor)
	}

	pkt := Packet{
		Type:        PacketHello,
		SenderID:    m.nodeID,
		SenderNick:  newNick,
		SenderColor: newColor,
		IsRelay:     m.isRelayMode,
		Timestamp:   time.Now().Unix(),
	}
	m.broadcastPacket(pkt, "")
	m.broadcastPeerSync()
	m.notifyPeerChange()
}

// SetRoomSecret 动态设置或修改江湖暗号
func (m *MeshNetwork) SetRoomSecret(secret string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roomSecret = secret
	m.roomKey = DeriveRoomKey(secret)
}

// GetPort 获取实际监听的 TCP 端口
func (m *MeshNetwork) GetPort() int {
	return m.tcpListenPort
}

// ConnectPeer 主动通过 TLS 1.3 连接远程对等节点（支持手动指定种子或 PEX 发现）
func (m *MeshNetwork) ConnectPeer(remoteAddr string) {
	if remoteAddr == "" {
		return
	}

	m.mu.Lock()
	if m.knownAddrs[remoteAddr] {
		for _, p := range m.peers {
			if p.tcpAddr == remoteAddr {
				m.mu.Unlock()
				return
			}
		}
	}
	m.knownAddrs[remoteAddr] = true
	m.mu.Unlock()

	go func() {
		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", remoteAddr, m.tlsConfig)
		if err != nil {
			return
		}
		m.handleNewConn(conn, remoteAddr)
	}()
}

func (m *MeshNetwork) acceptLoop() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			select {
			case <-m.stopChan:
				return
			default:
				continue
			}
		}
		go m.handleNewConn(conn, conn.RemoteAddr().String())
	}
}

func (m *MeshNetwork) handleNewConn(conn net.Conn, remoteAddr string) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	p := &peerConn{
		tcpAddr:  remoteAddr,
		conn:     conn,
		writer:   writer,
		lastSeen: time.Now(),
	}

	// 1. 发送自身 Hello 握手包与当前已知的全网花名册 (PEER_SYNC)
	m.mu.RLock()
	myNick := m.nick
	myColor := m.color
	myID := m.nodeID
	isRelay := m.isRelayMode
	known := make([]string, 0, len(m.knownAddrs))
	for addr := range m.knownAddrs {
		known = append(known, addr)
	}
	peersList := make([]PeerSummary, 0, len(m.globalPeers))
	for _, gp := range m.globalPeers {
		peersList = append(peersList, gp)
	}
	m.mu.RUnlock()

	helloPkt := Packet{
		Type:        PacketHello,
		SenderID:    myID,
		SenderNick:  myNick,
		SenderColor: myColor,
		IsRelay:     isRelay,
		Timestamp:   time.Now().Unix(),
		PeerAddrs:   known,
		PeerList:    peersList,
	}
	if err := sendPacket(writer, helloPkt); err != nil {
		conn.Close()
		return
	}

	// 2. 数据读取循环
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				// 连接断开
			}
			break
		}

		var pkt Packet
		if err := json.Unmarshal(line, &pkt); err != nil {
			continue
		}

		m.handleIncomingPacket(p, pkt)
	}

	m.removePeer(p)
}

func (m *MeshNetwork) handleIncomingPacket(p *peerConn, pkt Packet) {
	if pkt.SenderID == m.nodeID {
		return
	}

	p.mu.Lock()
	p.nodeID = pkt.SenderID
	p.nick = pkt.SenderNick
	p.color = pkt.SenderColor
	p.lastSeen = time.Now()
	p.mu.Unlock()

	m.mu.Lock()
	m.peers[pkt.SenderID] = p
	m.mu.Unlock()

	switch pkt.Type {
	case PacketHello:
		m.handlePEX(pkt.PeerAddrs)
		m.mergePeer(PeerSummary{
			ID:       pkt.SenderID,
			Nick:     pkt.SenderNick,
			Color:    pkt.SenderColor,
			IsRelay:  pkt.IsRelay,
			LastSeen: time.Now().Unix(),
		})
		if len(pkt.PeerList) > 0 {
			m.mergePeerList(pkt.PeerList, pkt.SenderID)
		}
		// 广播给其他直连节点同步
		m.broadcastPeerSync()

	case PacketPEX:
		m.handlePEX(pkt.PeerAddrs)

	case PacketPeerSync:
		m.mergePeerList(pkt.PeerList, pkt.SenderID)
		if m.isRelayMode {
			// 烽火台中继节点向其他客户端广播花名册
			m.broadcastPacket(pkt, pkt.SenderID)
		}

	case PacketPeerLeave:
		m.handlePeerLeave(pkt)

	case PacketMsg:
		// 消息去重
		m.mu.Lock()
		if _, seen := m.seenMsgIDs[pkt.MsgID]; seen {
			m.mu.Unlock()
			return
		}
		m.seenMsgIDs[pkt.MsgID] = time.Now()
		roomKey := m.roomKey
		m.mu.Unlock()

		// 转播给网格内其他节点 (Gossip Relay)
		m.broadcastPacket(pkt, pkt.SenderID)

		// 如果是纯中继/引导节点，无需本地渲染解密
		if m.isRelayMode {
			return
		}

		// 本地端到端解密
		decrypted, err := DecryptPayload(roomKey, pkt.Payload)
		if err != nil {
			// 暗号不符或被篡改，无法解密
			return
		}

		// 交付本地 TUI 展现
		if m.OnMessageReceived != nil {
			msg := &message.Message{
				ID:          pkt.MsgID,
				Sender:      pkt.SenderNick,
				SenderColor: pkt.SenderColor,
				Content:     decrypted,
				CreatedAt:   time.Unix(pkt.Timestamp, 0),
				TTL:         time.Duration(pkt.TTLSeconds) * time.Second,
				IsSystem:    false,
			}
			m.OnMessageReceived(msg)
		}

	case PacketMelt:
		// 收到紧急闭谷熔断令
		m.mu.RLock()
		configuredHash := m.adminTokenHashHex
		m.mu.RUnlock()

		// 校验口令哈希
		if configuredHash != "" && pkt.AdminHash == configuredHash {
			// 转播熔断令
			m.broadcastPacket(pkt, pkt.SenderID)

			if m.OnMeltTriggered != nil {
				m.OnMeltTriggered("掌门发动天绝闭谷令，全网信件已被天火焚毁")
			}
		}
	}
}

func (m *MeshNetwork) mergePeer(p PeerSummary) {
	if p.ID == m.nodeID || p.ID == "" {
		return
	}
	m.mu.Lock()
	old, exists := m.globalPeers[p.ID]
	p.LastSeen = time.Now().Unix()
	m.globalPeers[p.ID] = p
	m.mu.Unlock()

	if !exists {
		if !p.IsRelay && m.OnPeerEvent != nil {
			go m.OnPeerEvent("join", p.Nick, p.Color)
		}
		m.notifyPeerChange()
	} else if old.Nick != p.Nick {
		if !p.IsRelay && m.OnPeerEvent != nil {
			go m.OnPeerEvent("reroll", p.Nick, p.Color)
		}
		m.notifyPeerChange()
	}
}

func (m *MeshNetwork) mergePeerList(list []PeerSummary, fromID string) {
	changed := false
	now := time.Now().Unix()

	m.mu.Lock()
	for _, incoming := range list {
		if incoming.ID == m.nodeID || incoming.ID == "" {
			continue
		}

		old, exists := m.globalPeers[incoming.ID]
		incoming.LastSeen = now
		m.globalPeers[incoming.ID] = incoming

		if !exists {
			changed = true
			if !incoming.IsRelay && m.OnPeerEvent != nil {
				go m.OnPeerEvent("join", incoming.Nick, incoming.Color)
			}
		} else if old.Nick != incoming.Nick {
			changed = true
			if !incoming.IsRelay && m.OnPeerEvent != nil {
				go m.OnPeerEvent("reroll", incoming.Nick, incoming.Color)
			}
		}
	}
	m.mu.Unlock()

	if changed {
		m.notifyPeerChange()
	}
}

func (m *MeshNetwork) handlePeerLeave(pkt Packet) {
	m.mu.Lock()
	targetID := pkt.SenderID
	if p, exists := m.globalPeers[targetID]; exists {
		delete(m.globalPeers, targetID)
		if !p.IsRelay && m.OnPeerEvent != nil {
			go m.OnPeerEvent("leave", p.Nick, p.Color)
		}
	}
	m.mu.Unlock()

	if m.isRelayMode {
		m.broadcastPacket(pkt, pkt.SenderID)
	}
	m.notifyPeerChange()
}

func (m *MeshNetwork) handlePEX(addrs []string) {
	for _, addr := range addrs {
		if addr == "" {
			continue
		}
		m.mu.RLock()
		known := m.knownAddrs[addr]
		m.mu.RUnlock()

		if !known {
			m.ConnectPeer(addr)
		}
	}
}

// BroadcastMessage 发送并广播一条即焚聊天密信（AES-GCM 端到端加密）
func (m *MeshNetwork) BroadcastMessage(msgID string, content []byte, ttl time.Duration) error {
	m.mu.RLock()
	roomKey := m.roomKey
	myNick := m.nick
	myColor := m.color
	myID := m.nodeID
	m.mu.RUnlock()

	// 无论是否有暗号，一律进行端到端加密，杜绝明文
	encrypted, err := EncryptPayload(roomKey, content)
	if err != nil {
		return err
	}

	pkt := Packet{
		Type:        PacketMsg,
		SenderID:    myID,
		SenderNick:  myNick,
		SenderColor: myColor,
		MsgID:       msgID,
		Payload:     encrypted,
		TTLSeconds:  int(ttl.Seconds()),
		Timestamp:   time.Now().Unix(),
	}

	m.mu.Lock()
	m.seenMsgIDs[msgID] = time.Now()
	m.mu.Unlock()

	m.broadcastPacket(pkt, "")
	return nil
}

// TriggerMelt 管理员发号施令：广播紧急闭谷熔断令
func (m *MeshNetwork) TriggerMelt(adminKey string) error {
	h := DeriveAdminTokenHash(adminKey)
	hashHex := hex.EncodeToString(h[:])

	pkt := Packet{
		Type:        PacketMelt,
		SenderID:    m.nodeID,
		SenderNick:  m.nick,
		SenderColor: m.color,
		AdminHash:   hashHex,
		Timestamp:   time.Now().Unix(),
	}

	m.broadcastPacket(pkt, "")

	if m.OnMeltTriggered != nil {
		m.OnMeltTriggered("掌门发动天绝闭谷令，全网信件已被天火焚毁")
	}
	return nil
}

func (m *MeshNetwork) broadcastPeerSync() {
	m.mu.RLock()
	list := make([]PeerSummary, 0, len(m.globalPeers))
	for _, p := range m.globalPeers {
		list = append(list, p)
	}
	myID := m.nodeID
	myNick := m.nick
	myColor := m.color
	isRelay := m.isRelayMode
	m.mu.RUnlock()

	pkt := Packet{
		Type:        PacketPeerSync,
		SenderID:    myID,
		SenderNick:  myNick,
		SenderColor: myColor,
		IsRelay:     isRelay,
		Timestamp:   time.Now().Unix(),
		PeerList:    list,
	}
	m.broadcastPacket(pkt, "")
}

func (m *MeshNetwork) broadcastPacket(pkt Packet, excludeNodeID string) {
	m.mu.RLock()
	peers := make([]*peerConn, 0, len(m.peers))
	for id, p := range m.peers {
		if id != excludeNodeID {
			peers = append(peers, p)
		}
	}
	m.mu.RUnlock()

	for _, p := range peers {
		p.mu.Lock()
		_ = sendPacket(p.writer, pkt)
		p.mu.Unlock()
	}
}

func sendPacket(w *bufio.Writer, pkt Packet) error {
	data, err := json.Marshal(pkt)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		return err
	}
	return w.Flush()
}

func (m *MeshNetwork) removePeer(p *peerConn) {
	p.conn.Close()
	m.mu.Lock()
	nodeID := p.nodeID
	nick := p.nick
	color := p.color
	isRelay := false
	if nodeID != "" {
		delete(m.peers, nodeID)
		if gp, exists := m.globalPeers[nodeID]; exists {
			isRelay = gp.IsRelay
			delete(m.globalPeers, nodeID)
		}
	}
	m.mu.Unlock()

	if nodeID != "" {
		if !isRelay && m.OnPeerEvent != nil {
			go m.OnPeerEvent("leave", nick, color)
		}

		leavePkt := Packet{
			Type:        PacketPeerLeave,
			SenderID:    nodeID,
			SenderNick:  nick,
			IsRelay:     isRelay,
			Timestamp:   time.Now().Unix(),
		}
		m.broadcastPacket(leavePkt, "")
		m.notifyPeerChange()
	}
}

func (m *MeshNetwork) notifyPeerChange() {
	if m.OnPeerChange == nil {
		return
	}
	m.mu.RLock()
	peers := make([]PeerSummary, 0, len(m.globalPeers))
	humanCount := 0
	for _, p := range m.globalPeers {
		peers = append(peers, p)
		if !p.IsRelay {
			humanCount++
		}
	}
	if humanCount < 1 && !m.isRelayMode {
		humanCount = 1
	}
	m.mu.RUnlock()

	go m.OnPeerChange(humanCount, peers)
}

func (m *MeshNetwork) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case now := <-ticker.C:
			nowUnix := now.Unix()
			m.mu.Lock()

			// 清理过期的去重 ID
			for id, t := range m.seenMsgIDs {
				if now.Sub(t) > 5*time.Minute {
					delete(m.seenMsgIDs, id)
				}
			}

			// 刷新自身时间戳
			if self, ok := m.globalPeers[m.nodeID]; ok {
				self.LastSeen = nowUnix
				m.globalPeers[m.nodeID] = self
			}

			// 检查超时节点（非本地直连且超过 8 秒未有心跳的远端节点）
			changed := false
			for id, p := range m.globalPeers {
				if id == m.nodeID {
					continue
				}
				if _, isDirect := m.peers[id]; isDirect {
					p.LastSeen = nowUnix
					m.globalPeers[id] = p
				} else {
					if nowUnix-p.LastSeen > 8 {
						delete(m.globalPeers, id)
						changed = true
						if !p.IsRelay && m.OnPeerEvent != nil {
							go m.OnPeerEvent("leave", p.Nick, p.Color)
						}
					}
				}
			}
			m.mu.Unlock()

			if changed {
				m.notifyPeerChange()
			}

			// 定期广播全网花名册同步
			m.broadcastPeerSync()
		}
	}
}
