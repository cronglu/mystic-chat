package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mystic-chat/internal/identity"
	"mystic-chat/internal/message"
	"mystic-chat/internal/network"
	"mystic-chat/internal/sshserver"
	"mystic-chat/internal/tui"
)

func main() {
	p2pPort := flag.Int("p2p-port", 19001, "P2P 网格 TCP 监听端口")
	udpPort := flag.Int("udp-port", 9988, "局域网 UDP 嗅探广播端口")
	noUDP := flag.Bool("no-udp", true, "完全禁用 UDP 局域网广播（默认即为禁用，纯 TCP 安全模式）")
	enableUDP := flag.Bool("udp", false, "主动启用 UDP 局域网嗅探广播（默认关闭）")
	sshPort := flag.Int("ssh-port", 2222, "嵌入式 SSH 服务端口（免客户端通过 ssh 连接）")
	noSSH := flag.Bool("no-ssh", false, "关闭嵌入式 SSH 服务")
	peerAddr := flag.String("peer", "", "初始引导节点地址（如 10.20.13.115，若省略端口自动补全 :19001）")
	isBootstrap := flag.Bool("bootstrap", false, "以纯后台无界面引导/中继烽火台模式运行（适合部署在 Linux 服务器）")
	isRelay := flag.Bool("relay", false, "同 -bootstrap，中继转发模式")
	roomSecret := flag.String("room", "", "江湖暗号（开启端到端 AES-256 全内存加密房间）")
	adminKey := flag.String("admin-key", "", "管理员紧急熔断口令（配置后可通过 /killswitch 广播闭谷关停令）")
	defaultTTLStr := flag.String("ttl", "60s", "默认消息即焚时限（如 15s, 30s, 60s, 120s）")

	flag.Parse()

	// 默认禁用 UDP（除非显式传入 -udp）
	disableUDP := *noUDP && !*enableUDP

	defaultTTL, err := time.ParseDuration(*defaultTTLStr)
	if err != nil || defaultTTL <= 0 {
		defaultTTL = 60 * time.Second
	}

	// 随机节点标识
	nodeID := fmt.Sprintf("node-%d-%04d", time.Now().Unix(), rand.Intn(10000))
	nick := identity.GenerateIdentity()
	color := identity.GetIdentityColor(nick)

	// 1. 初始化纯内存存储中心
	store := message.NewStore()
	defer store.ClearAll() // 退出时确保内存彻底抹除

	// 2. 初始化 P2P 网格引擎（TLS 1.3 传输加密 + 端到端 AES-GCM 双重加密）
	mesh := network.NewMeshNetwork(nodeID, nick, color, *p2pPort, *udpPort, disableUDP, *roomSecret, *adminKey)
	if *isBootstrap || *isRelay {
		mesh.SetRelayMode(true)
	}

	if err := mesh.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动 P2P 网络失败: %v\n", err)
		os.Exit(1)
	}
	defer mesh.Stop()

	// 若指定了初始引导服务器（例如部署在 Linux 服务器上的节点）
	if *peerAddr != "" {
		targetAddr := *peerAddr
		if !strings.Contains(targetAddr, ":") {
			targetAddr = fmt.Sprintf("%s:%d", targetAddr, *p2pPort)
		}
		mesh.ConnectPeer(targetAddr)
	}

	// 3. 处理纯后台引导/中继模式（适合在 Linux 服务器运行）
	if *isBootstrap || *isRelay {
		fmt.Println("==================================================================")
		fmt.Println("⚔️  mystic-chat 江湖烽火台（引导与中继服务器）已启动")
		fmt.Printf("📡 P2P 监听端口: :%d (强制 TLS 1.3 传输加密，防网关/路由器嗅探)\n", mesh.GetPort())
		if mesh.IsUDPEnabled() {
			fmt.Printf("🌐 UDP 自动嗅探: 已启用 (广播端口 :%d)\n", *udpPort)
		} else {
			fmt.Println("🚫 UDP 自动嗅探: 已完全禁用 (纯 TCP 种子直连与 PEX 对等互换自愈)")
		}
		if *roomSecret != "" {
			fmt.Printf("🔒 当前密令房间: 【%s】 (AES-256-GCM 端到端加密)\n", *roomSecret)
		} else {
			fmt.Println("🔓 当前处于公开大厅模式（底层默认防嗅探混淆加密）")
		}
		if *adminKey != "" {
			fmt.Println("🛡️  管理员紧急熔断机制已就绪")
		}
		fmt.Println("\n💡 终端用户可通过以下命令连接至此引导服务器:")
		fmt.Printf("   ./mystic-chat -peer <此服务器IP>:%d\n", mesh.GetPort())
		if !*noSSH {
			fmt.Printf("   或者直接免客户端终端接入: ssh <此服务器IP> -p %d\n", *sshPort)
		}
		fmt.Println("==================================================================")

		// 启动 SSH 服务（若未关闭）
		if !*noSSH {
			sshSrv, err := sshserver.NewServer(*sshPort, store, mesh, *roomSecret, *adminKey, defaultTTL)
			if err == nil {
				_ = sshSrv.Start()
				defer func() {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					_ = sshSrv.Shutdown(ctx)
				}()
			}
		}

		// 阻塞等待信号退出
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\n正在熄灭烽火，抹除内存并安全退出...")
		return
	}

	// 4. 普通终端模式：启动本地嵌入式 SSH 服务（借道免客户端接入）
	if !*noSSH {
		sshSrv, err := sshserver.NewServer(*sshPort, store, mesh, *roomSecret, *adminKey, defaultTTL)
		if err == nil {
			_ = sshSrv.Start()
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = sshSrv.Shutdown(ctx)
			}()
		}
	}

	// 5. 启动本地 Bubble Tea TUI 终端
	model := tui.NewModel(store, mesh, nick, color, *roomSecret, *adminKey, defaultTTL)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "运行终端界面出错: %v\n", err)
		os.Exit(1)
	}
}
