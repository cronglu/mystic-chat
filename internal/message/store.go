package message

import (
	"sync"
	"time"
)

// Message 代表一条在内存中流转的江湖密信
type Message struct {
	ID          string        `json:"id"`
	Sender      string        `json:"sender"`
	SenderColor string        `json:"sender_color"`
	Content     []byte        `json:"content"`      // 使用 []byte 以便彻底零值擦除
	CreatedAt   time.Time     `json:"created_at"`
	TTL         time.Duration `json:"ttl"`
	IsSystem    bool          `json:"is_system"`
}

// DisplayItem 供 TUI 渲染展示的视图项
type DisplayItem struct {
	ID           string
	Sender       string
	SenderColor  string
	RenderedText string
	TextColor    string
	Remaining    time.Duration
	RemainingStr string
	IsSystem     bool
	CreatedAt    time.Time
}

// Store 纯内存消息存储中心，负责消息倒计时与物理内存抹除
type Store struct {
	mu       sync.RWMutex
	messages []*Message
}

// NewStore 创建纯内存消息存储中心
func NewStore() *Store {
	return &Store{
		messages: make([]*Message, 0, 100),
	}
}

// Add 添加一条新消息到内存
func (s *Store) Add(msg *Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 复制字节切片防止外部引用改变
	contentCopy := make([]byte, len(msg.Content))
	copy(contentCopy, msg.Content)

	newMsg := &Message{
		ID:          msg.ID,
		Sender:      msg.Sender,
		SenderColor: msg.SenderColor,
		Content:     contentCopy,
		CreatedAt:   msg.CreatedAt,
		TTL:         msg.TTL,
		IsSystem:    msg.IsSystem,
	}

	s.messages = append(s.messages, newMsg)
}

// ClearAll 立即将所有消息在内存中物理清零覆写并清空切片
func (s *Store) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range s.messages {
		wipeBytes(msg.Content)
		msg.Sender = ""
		msg.ID = ""
	}
	s.messages = s.messages[:0]
}

// PurgeAndGetActive 清理所有已过期的消息（清零内存），并返回当前未过期且经过风化计算的渲染列表
func (s *Store) PurgeAndGetActive(now time.Time) []DisplayItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	active := make([]*Message, 0, len(s.messages))
	items := make([]DisplayItem, 0, len(s.messages))

	for _, msg := range s.messages {
		text := string(msg.Content)
		state := ComputeAmnesia(text, msg.CreatedAt, msg.TTL, now)

		if state.IsExpired {
			// 已到期：内存物理抹除！
			wipeBytes(msg.Content)
			msg.Sender = ""
			msg.ID = ""
			continue
		}

		active = append(active, msg)

		items = append(items, DisplayItem{
			ID:           msg.ID,
			Sender:       msg.Sender,
			SenderColor:  msg.SenderColor,
			RenderedText: state.RenderedText,
			TextColor:    state.TextColor,
			Remaining:    state.Remaining,
			RemainingStr: FormatRemaining(state.Remaining),
			IsSystem:     msg.IsSystem,
			CreatedAt:    msg.CreatedAt,
		})
	}

	s.messages = active
	return items
}

// wipeBytes 将底层字节切片覆写为 0
func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
