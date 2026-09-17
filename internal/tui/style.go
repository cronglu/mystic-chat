package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// 主题底色与边框（墨黑暗夜高科技风）
	BgColor         = lipgloss.Color("#080c10")
	BorderColor     = lipgloss.Color("#1e293b")
	ActiveBorderColor = lipgloss.Color("#00f0ff")

	// 标题与横幅样式
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00f0ff")).
			Background(lipgloss.Color("#0f172a")).
			Padding(0, 1)

	RoomTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffd700")).
			Background(lipgloss.Color("#1e1b4b")).
			Bold(true).
			Padding(0, 1)

	PeerCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#39ff14")).
			Background(lipgloss.Color("#064e3b")).
			Padding(0, 1)

	WatermarkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1e293b")).
			Italic(true)

	// 状态栏
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0f172a")).
			Foreground(lipgloss.Color("#94a3b8")).
			Padding(0, 1)

	TTLTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff007f")).
			Background(lipgloss.Color("#4a044e")).
			Bold(true).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#38bdf8")).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748b"))

	// 消息区域
	SystemMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fbbf24")).
			Italic(true)

	TimeRemainingBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f43f5e")).
				Bold(true)

	MeltAlertStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#b91c1c")).
			Padding(1, 2).
			Align(lipgloss.Center)
)
