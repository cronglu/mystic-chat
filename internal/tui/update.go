package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"mystic-chat/internal/message"
	"mystic-chat/internal/network"
)

// Update 处理 Bubble Tea 事件与交互状态转移
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		headerHeight := 3 // 标题 + 水印
		footerHeight := 4 // 输入框 + 状态栏
		vpHeight := m.Height - headerHeight - footerHeight
		if vpHeight < 5 {
			vpHeight = 5
		}

		if !m.Ready {
			m.Viewport = viewport.New(m.Width, vpHeight)
			m.Viewport.SetContent(m.RenderMessages())
			m.Ready = true
		} else {
			m.Viewport.Width = m.Width
			m.Viewport.Height = vpHeight
		}

		m.TextArea.SetWidth(m.Width - 4)

	case TickMsg:
		if m.Ready && !m.IsMelted {
			m.Viewport.SetContent(m.RenderMessages())
		}
		cmds = append(cmds, tickCmd())

	case IncomingMsg:
		if msg != nil {
			m.Store.Add((*message.Message)(msg))
			if m.Ready {
				m.Viewport.SetContent(m.RenderMessages())
				m.Viewport.GotoBottom()
			}
		}
		cmds = append(cmds, waitForIncomingMessage(m.inMsgChan))

	case MeltMsg:
		m.IsMelted = true
		m.MeltReason = string(msg)
		// 内存瞬间物理抹除清零！
		m.Store.ClearAll()
		return m, tea.ClearScreen

	case PeerUpdateMsg:
		m.PeerCount = int(msg)
		cmds = append(cmds, waitForPeerUpdate(m.peerChan))

	case PeerEventMsg:
		switch msg.Action {
		case "join":
			m.addSystemMessage(fmt.Sprintf("同门侠客 %s 踏雪而至，现身江湖！", msg.Nick))
		case "leave":
			m.addSystemMessage(fmt.Sprintf("同门侠客 %s 飘然离去，归隐山林。", msg.Nick))
		case "reroll":
			m.addSystemMessage(fmt.Sprintf("有同门施展易容秘术，化身为 %s", msg.Nick))
		}
		if m.Ready {
			m.Viewport.SetContent(m.RenderMessages())
			m.Viewport.GotoBottom()
		}
		cmds = append(cmds, waitForPeerEvent(m.peerEventChan))

	case tea.KeyMsg:
		if m.IsMelted {
			// 熔断后任意退出按键
			return m, tea.Quit
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// 物理清零后退出
			m.Store.ClearAll()
			return m, tea.Quit

		case tea.KeyCtrlR:
			// 施展易容术
			m.RerollIdentity()
			m.Viewport.SetContent(m.RenderMessages())
			m.Viewport.GotoBottom()
			return m, nil

		case tea.KeyTab, tea.KeyF2:
			// 切换即焚时长
			m.CycleTTL()
			return m, nil

		case tea.KeyCtrlL:
			// 一键焚毁全屏与内存
			m.Store.ClearAll()
			m.Viewport.SetContent(m.RenderMessages())
			return m, tea.ClearScreen

		case tea.KeyEnter:
			input := strings.TrimSpace(m.TextArea.Value())
			if input == "" {
				return m, nil
			}

			// 指令解析
			if strings.HasPrefix(input, "/") {
				m.handleCommand(input)
				m.TextArea.Reset()
				m.Viewport.SetContent(m.RenderMessages())
				m.Viewport.GotoBottom()
				return m, nil
			}

			// 普通消息发送
			msgID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
			contentBytes := []byte(input)

			// 1. 存入本地内存
			localMsg := &message.Message{
				ID:          msgID,
				Sender:      m.Identity,
				SenderColor: m.IdentityColor,
				Content:     contentBytes,
				CreatedAt:   time.Now(),
				TTL:         m.CurrentTTL,
				IsSystem:    false,
			}
			m.Store.Add(localMsg)

			// 2. 广播至网格总线 (P2P + 加密)
			if m.Mesh != nil {
				_ = m.Mesh.BroadcastMessage(msgID, contentBytes, m.CurrentTTL)
			}

			m.TextArea.Reset()
			m.Viewport.SetContent(m.RenderMessages())
			m.Viewport.GotoBottom()
			return m, nil
		}
	}

	// 传递事件给 TextArea
	m.TextArea, tiCmd = m.TextArea.Update(msg)
	cmds = append(cmds, tiCmd)

	// 传递事件给 Viewport
	m.Viewport, vpCmd = m.Viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleCommand(cmdStr string) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/help":
		helpContent := "📜 【江湖密令指南】\n" +
			"  /reroll                - 施展易容术，随机更换武侠身份\n" +
			"  /ttl <时限>            - 设定即焚时限，如 /ttl 15s 或 /ttl 2m\n" +
			"  /room <暗号>           - 切换密令房间（AES-256 全内存加密）\n" +
			"  /connect <IP:端口>     - 主动连接已知侠客节点（应对无 UDP 局域网）\n" +
			"  /peers                 - 查看当前已连接的同门侠客列表\n" +
			"  /clear                 - 立即焚毁屏幕与内存中所有信件\n" +
			"  /killswitch <管理口令> - [风险管控] 广播天绝闭谷令，全网销毁并安全关停\n" +
			"快捷键: Tab(切换即焚) | Ctrl+R(易容) | Ctrl+L(焚毁) | Esc(退出)"
		m.addSystemMessage(helpContent)

	case "/reroll":
		m.RerollIdentity()

	case "/clear":
		m.Store.ClearAll()

	case "/ttl":
		if len(parts) < 2 {
			m.addSystemMessage("请指定时限，例如: /ttl 30s 或 /ttl 1m")
			return
		}
		dur, err := time.ParseDuration(parts[1])
		if err != nil || dur <= 0 {
			m.addSystemMessage("时限格式无效，示例: 15s, 30s, 1m, 2m")
			return
		}
		m.CurrentTTL = dur
		m.addSystemMessage(fmt.Sprintf("即焚时限已调整为: %s", dur))

	case "/room":
		if len(parts) < 2 {
			m.RoomSecret = ""
			if m.Mesh != nil {
				m.Mesh.SetRoomSecret("")
			}
			m.addSystemMessage("已退回【公开大厅】，使用默认防嗅探加密。")
		} else {
			secret := parts[1]
			m.RoomSecret = secret
			if m.Mesh != nil {
				m.Mesh.SetRoomSecret(secret)
			}
			m.addSystemMessage(fmt.Sprintf("已开启密令房间【%s】，AES-256 全内存加密已生效！", secret))
		}

	case "/connect":
		if len(parts) < 2 {
			m.addSystemMessage("请指定目标侠客地址，例如: /connect 192.168.1.100:19001")
			return
		}
		addr := parts[1]
		if m.Mesh != nil {
			m.Mesh.ConnectPeer(addr)
			m.addSystemMessage(fmt.Sprintf("已发起对 [%s] 的飞鸽连通...", addr))
		}

	case "/peers":
		if m.Mesh == nil {
			m.addSystemMessage("当前单机运行，未开启网络网格。")
			return
		}
		peers := m.Mesh.GetGlobalPeers()
		if len(peers) == 0 {
			m.addSystemMessage("当前暂无其他在线侠客。")
			return
		}
		var sb strings.Builder
		humanCount := 0
		for _, p := range peers {
			if !p.IsRelay {
				humanCount++
			}
		}
		sb.WriteString(fmt.Sprintf("👥 当前同门在线侠客 (%d 位):\n", humanCount))
		for _, p := range peers {
			if !p.IsRelay {
				isSelf := ""
				if p.ID == m.Mesh.GetNodeID() {
					isSelf = " (你自己)"
				}
				sb.WriteString(fmt.Sprintf("  • %s%s\n", p.Nick, isSelf))
			}
		}

		relays := make([]network.PeerSummary, 0)
		for _, p := range peers {
			if p.IsRelay {
				relays = append(relays, p)
			}
		}
		if len(relays) > 0 {
			sb.WriteString("\n📡 烽火台中继节点:\n")
			for _, r := range relays {
				sb.WriteString(fmt.Sprintf("  • %s\n", r.Nick))
			}
		}
		m.addSystemMessage(sb.String())

	case "/killswitch", "/melt":
		if len(parts) < 2 {
			m.addSystemMessage("⚠️ 请输入管理员口令，例如: /killswitch <管理密令>")
			return
		}
		key := parts[1]
		if m.Mesh != nil {
			_ = m.Mesh.TriggerMelt(key)
		}

	default:
		m.addSystemMessage(fmt.Sprintf("未知密令: %s，输入 /help 查看指引。", cmd))
	}
}

func (m *Model) addSystemMessage(text string) {
	sysMsg := &message.Message{
		ID:          fmt.Sprintf("sys-%d", time.Now().UnixNano()),
		Sender:      "【天机阁】",
		SenderColor: "#ffd700",
		Content:     []byte(text),
		CreatedAt:   time.Now(),
		TTL:         45 * time.Second,
		IsSystem:    true,
	}
	m.Store.Add(sysMsg)
}
