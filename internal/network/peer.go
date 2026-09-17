package network

import (
	"time"
)

// PacketType 数据包类型
type PacketType string

const (
	PacketMsg       PacketType = "MSG"        // 普通聊天密信
	PacketHello     PacketType = "HELLO"      // 节点问候与握手
	PacketPEX       PacketType = "PEX"        // 对等节点交换 (Peer Exchange)
	PacketMelt      PacketType = "MELT"       // 管理员紧急熔断闭谷令
	PacketPeerSync  PacketType = "PEER_SYNC"  // 全网在线同门花名册同步
	PacketPeerLeave PacketType = "PEER_LEAVE" // 侠客离线或闭谷退出公告
)

// PeerSummary 全网在线侠客摘要信息（用于全网在线人数感知与跨中继同步）
type PeerSummary struct {
	ID       string `json:"id"`
	Nick     string `json:"nick"`
	Color    string `json:"color"`
	IsRelay  bool   `json:"is_relay"`
	LastSeen int64  `json:"last_seen"`
}

// Packet 网络传输封包
type Packet struct {
	Type        PacketType    `json:"type"`
	SenderID    string        `json:"sender_id"`
	SenderNick  string        `json:"sender_nick"`
	SenderColor string        `json:"sender_color"`
	IsRelay     bool          `json:"is_relay,omitempty"`
	MsgID       string        `json:"msg_id,omitempty"`
	Payload     []byte        `json:"payload,omitempty"`
	TTLSeconds  int           `json:"ttl_seconds,omitempty"`
	Timestamp   int64         `json:"timestamp"`
	PeerAddrs   []string      `json:"peer_addrs,omitempty"`
	AdminHash   string        `json:"admin_hash,omitempty"`
	PeerList    []PeerSummary `json:"peer_list,omitempty"` // 用于 PEER_SYNC 全网状态同步
}

// PeerInfo 在线侠客元数据
type PeerInfo struct {
	ID       string    `json:"id"`
	Nick     string    `json:"nick"`
	Color    string    `json:"color"`
	TCPAddr  string    `json:"tcp_addr"`
	IsRelay  bool      `json:"is_relay"`
	LastSeen time.Time `json:"last_seen"`
}
