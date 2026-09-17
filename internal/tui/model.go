package tui

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"mystic-chat/internal/identity"
	"mystic-chat/internal/message"
	"mystic-chat/internal/network"
)

// TTLOptions 可快速切换的即焚时限池
var TTLOptions = []time.Duration{
	15 * time.Second,
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
}

// TickMsg 驱动文字风化渐变动画的时钟消息
type TickMsg time.Time

// IncomingMsg 接收到局域网密信
type IncomingMsg *message.Message

// MeltMsg 收到全网闭谷熔断令
type MeltMsg string

// PeerUpdateMsg 侠客上下线变动
type PeerUpdateMsg int

// PeerEventMsg 侠客踏雪而来或离去事件
type PeerEventMsg struct {
	Action string // "join", "leave", "reroll"
	Nick   string
	Color  string
}

// Model Bubble Tea 状态模型
type Model struct {
	ID            string
	Identity      string
	IdentityColor string
	Store         *message.Store
	Mesh          *network.MeshNetwork
	RoomSecret    string
	AdminKey      string

	Viewport  viewport.Model
	TextArea  textarea.Model
	Width     int
	Height    int
	Ready     bool
	PeerCount int

	CurrentTTLIndex int
	CurrentTTL      time.Duration

	IsMelted   bool
	MeltReason string

	// 异步事件通道
	inMsgChan     chan *message.Message
	meltChan      chan string
	peerChan      chan int
	peerEventChan chan PeerEventMsg
	sessionID     string
}

// NewModel 创建 TUI 状态模型
func NewModel(store *message.Store, mesh *network.MeshNetwork, initialNick, initialColor, roomSecret, adminKey string, defaultTTL time.Duration) Model {
	ta := textarea.New()
	ta.Placeholder = "输入江湖密信... (回车飞鸽传书，Tab 切换即焚时长，输入 /help 查看密宗指令)"
	ta.Focus()
	ta.Prompt = "❯ "
	ta.CharLimit = 500
	ta.SetHeight(2)
	ta.ShowLineNumbers = false

	if defaultTTL <= 0 {
		defaultTTL = 60 * time.Second
	}

	ttlIdx := 2 // 默认 60s
	for i, opt := range TTLOptions {
		if opt == defaultTTL {
			ttlIdx = i
			break
		}
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	sessID := fmt.Sprintf("%06d", n.Int64())

	initialPeerCount := 1
	if mesh != nil {
		initialPeerCount = mesh.GetHumanCount()
	}

	m := Model{
		ID:              sessID,
		Identity:        initialNick,
		IdentityColor:   initialColor,
		Store:           store,
		Mesh:            mesh,
		RoomSecret:      roomSecret,
		AdminKey:        adminKey,
		TextArea:        ta,
		PeerCount:       initialPeerCount,
		CurrentTTLIndex: ttlIdx,
		CurrentTTL:      TTLOptions[ttlIdx],
		inMsgChan:       make(chan *message.Message, 100),
		meltChan:        make(chan string, 10),
		peerChan:        make(chan int, 20),
		peerEventChan:   make(chan PeerEventMsg, 50),
		sessionID:       sessID,
	}

	// 挂载网络层回调
	if mesh != nil {
		mesh.OnMessageReceived = func(msg *message.Message) {
			m.inMsgChan <- msg
		}
		mesh.OnMeltTriggered = func(reason string) {
			m.meltChan <- reason
		}
		mesh.OnPeerChange = func(count int, peers []network.PeerSummary) {
			m.peerChan <- count
		}
		mesh.OnPeerEvent = func(action, nick, color string) {
			m.peerEventChan <- PeerEventMsg{Action: action, Nick: nick, Color: color}
		}
	}

	return m
}

// Init 初始化订阅
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tickCmd(),
		waitForIncomingMessage(m.inMsgChan),
		waitForMelt(m.meltChan),
		waitForPeerUpdate(m.peerChan),
		waitForPeerEvent(m.peerEventChan),
	)
}

func waitForPeerEvent(sub <-chan PeerEventMsg) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-sub
		if !ok {
			return nil
		}
		return evt
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func waitForIncomingMessage(sub <-chan *message.Message) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return IncomingMsg(msg)
	}
}

func waitForMelt(sub <-chan string) tea.Cmd {
	return func() tea.Msg {
		reason, ok := <-sub
		if !ok {
			return nil
		}
		return MeltMsg(reason)
	}
}

func waitForPeerUpdate(sub <-chan int) tea.Cmd {
	return func() tea.Msg {
		count, ok := <-sub
		if !ok {
			return nil
		}
		return PeerUpdateMsg(count)
	}
}

// RerollIdentity 施展易容术更换武侠身份
func (m *Model) RerollIdentity() {
	newIdentity := identity.GenerateIdentity()
	newColor := identity.GetIdentityColor(newIdentity)
	m.Identity = newIdentity
	m.IdentityColor = newColor

	if m.Mesh != nil {
		m.Mesh.UpdateIdentity(newIdentity, newColor)
	}

	// 本地插入系统提示
	sysMsg := &message.Message{
		ID:          fmt.Sprintf("sys-%d", time.Now().UnixNano()),
		Sender:      "【天机阁】",
		SenderColor: "#ffd700",
		Content:     []byte(fmt.Sprintf("侠客施展易容秘术，化身为 %s", newIdentity)),
		CreatedAt:   time.Now(),
		TTL:         30 * time.Second,
		IsSystem:    true,
	}
	m.Store.Add(sysMsg)
}

// CycleTTL 循环切换即焚时限
func (m *Model) CycleTTL() {
	m.CurrentTTLIndex = (m.CurrentTTLIndex + 1) % len(TTLOptions)
	m.CurrentTTL = TTLOptions[m.CurrentTTLIndex]
}
