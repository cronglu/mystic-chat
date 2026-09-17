package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View 渲染整个 TUI 界面
func (m Model) View() string {
	if !m.Ready {
		return "\n  正在入定初始化江湖网络，请稍候..."
	}

	if m.IsMelted {
		// 熔断紧急状态全屏告示
		box := MeltAlertStyle.Width(m.Width - 4).Render(
			fmt.Sprintf("⚠️  【天绝闭谷令 · 全网紧急熔断】  ⚠️\n\n%s\n\n所有驻留内存已执行物理零值覆写 (Secure Zero-Wipe)，无任何痕迹残留。\n请按 Esc 或 Ctrl+C 安全退出终端。", m.MeltReason),
		)
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
	}

	// 1. 顶部状态横幅
	header := m.renderHeader()

	// 2. 防窥与防截屏溯源水印条
	watermark := m.renderWatermark()

	// 3. 底部输入框与快捷键状态栏
	statusBar := m.renderStatusBar()
	inputView := m.TextArea.View()

	// 4. 计算视口并拼接
	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		watermark,
		m.Viewport.View(),
		inputView,
		statusBar,
	)

	return mainView
}

func (m Model) renderHeader() string {
	title := TitleStyle.Render("⚔️  mystic-chat · 极客武侠即焚终端")

	roomLabel := "公开大厅 · 游历四方"
	if len(m.RoomSecret) > 0 {
		roomLabel = fmt.Sprintf("密令房间 · 【%s】", m.RoomSecret)
	}
	roomTag := RoomTagStyle.Render("🔒 " + roomLabel)

	displayCount := m.PeerCount
	if displayCount < 1 {
		displayCount = 1
	}
	peerTag := PeerCountStyle.Render(fmt.Sprintf("👥 在线同门: %d 人", displayCount))

	spaceWidth := m.Width - lipgloss.Width(title) - lipgloss.Width(roomTag) - lipgloss.Width(peerTag) - 2
	if spaceWidth < 1 {
		spaceWidth = 1
	}
	spaces := strings.Repeat(" ", spaceWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, title, " ", roomTag, spaces, peerTag)
}

func (m Model) renderWatermark() string {
	nowStr := time.Now().Format("15:04:05")
	text := fmt.Sprintf(" [防截屏水印 | 节点:%s | %s | TLS1.3双重加密 | 纯内存运行] ", m.sessionID, nowStr)
	return WatermarkStyle.Render(text)
}

func (m Model) renderStatusBar() string {
	// 当前武侠身份
	identStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.IdentityColor)).
		Background(lipgloss.Color("#1e293b")).
		Padding(0, 1)
	identView := identStyle.Render("当前行头: " + m.Identity)

	// 即焚时限徽章
	ttlView := TTLTagStyle.Render(fmt.Sprintf("⏳ 即焚倒计时: %s", m.CurrentTTL))

	// 快捷键提示
	helpText := fmt.Sprintf(
		"%s%s %s%s %s%s %s%s",
		HelpKeyStyle.Render("[Tab]"), HelpDescStyle.Render("即焚时限 "),
		HelpKeyStyle.Render("[Ctrl+R]"), HelpDescStyle.Render("易容 "),
		HelpKeyStyle.Render("[Ctrl+L]"), HelpDescStyle.Render("焚毁 "),
		HelpKeyStyle.Render("[Esc]"), HelpDescStyle.Render("避世"),
	)

	content := lipgloss.JoinHorizontal(
		lipgloss.Center,
		identView,
		" ",
		ttlView,
		"  ",
		helpText,
	)

	return StatusBarStyle.Width(m.Width).Render(content)
}

// RenderMessages 将当前活动且风化处理后的消息列表渲染为视口文本
func (m Model) RenderMessages() string {
	items := m.Store.PurgeAndGetActive(time.Now())
	if len(items) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569")).
			Italic(true).
			Render("\n  （长风萧萧，四顾寂寥。尚无密信传达，发一言以通江湖...）\n")
	}

	var sb strings.Builder
	for _, item := range items {
		if item.IsSystem {
			sb.WriteString(fmt.Sprintf("  %s %s\n\n",
				SystemMsgStyle.Render("⚙️ "+item.Sender),
				lipgloss.NewStyle().Foreground(lipgloss.Color(item.TextColor)).Render(item.RenderedText),
			))
			continue
		}

		// 发送者专属颜色渲染
		senderStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(item.SenderColor))

		// 倒计时剩余时间微章
		timeBadge := TimeRemainingBadgeStyle.Render(fmt.Sprintf("[%s]", item.RemainingStr))

		// 动态失忆渐变色渲染正文
		msgContentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(item.TextColor))

		line := fmt.Sprintf("  %s %s  %s\n  %s\n\n",
			senderStyle.Render(item.Sender),
			timeBadge,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#334155")).Render(item.CreatedAt.Format("15:04:05")),
			msgContentStyle.Render(item.RenderedText),
		)
		sb.WriteString(line)
	}

	return sb.String()
}
