package sshserver

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	wishbubbletea "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	"mystic-chat/internal/identity"
	"mystic-chat/internal/message"
	"mystic-chat/internal/network"
	"mystic-chat/internal/tui"
)

// Server 嵌入式 Wish SSH 终端服务端
type Server struct {
	server *ssh.Server
	port   int
}

// NewServer 创建嵌入式 SSH 服务，让局域网其他无安装环境的同事只需 `ssh -p 2222 <ip>` 即可接入
func NewServer(port int, store *message.Store, mesh *network.MeshNetwork, roomSecret, adminKey string, defaultTTL time.Duration) (*Server, error) {
	teaHandler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, active := s.Pty()
		if !active {
			wish.Fatalln(s, "请在具有 PTY 终端的环境下使用 SSH 连接")
			return nil, nil
		}

		// 为连入的 SSH 用户生成独立的武侠江湖身份
		nick := identity.GenerateIdentity()
		color := identity.GetIdentityColor(nick)

		model := tui.NewModel(store, mesh, nick, color, roomSecret, adminKey, defaultTTL)
		model.Width = pty.Window.Width
		model.Height = pty.Window.Height

		return model, []tea.ProgramOption{tea.WithAltScreen()}
	}

	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("0.0.0.0:%d", port)),
		wish.WithMiddleware(
			wishbubbletea.Middleware(teaHandler),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, err
	}

	return &Server{
		server: s,
		port:   port,
	}, nil
}

// Start 在后台协程启动 SSH 监听
func (s *Server) Start() error {
	go func() {
		_ = s.server.ListenAndServe()
	}()
	return nil
}

// Shutdown 优雅关闭 SSH 服务
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
