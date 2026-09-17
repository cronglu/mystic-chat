# ⚔️ mystic-chat · 极客武侠阅后即焚终端

[![Build and Release](https://github.com/cronglu/mystic-chat/actions/workflows/build.yml/badge.svg)](https://github.com/cronglu/mystic-chat/actions/workflows/build.yml)
[![GitHub Release](https://img.shields.io/github/v/release/cronglu/mystic-chat?color=cyan)](https://github.com/cronglu/mystic-chat/releases)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`mystic-chat`（密室）是一款专为局域网/内网极客打造的纯内存、防抓包、去中心化武侠风格阅后即焚终端。
支持字符平滑淡化与残卷风化消散（失忆特效）、TLS 1.3 传输加密与端到端 AES-256-GCM 双重加密、P2P 网格自愈及免安装 SSH 终端借道。

---

## ⚡ 极速一键安装

在支持 `bash` 的终端（macOS / Linux）中直接执行：

```bash
curl -fsSL https://raw.githubusercontent.com/cronglu/mystic-chat/main/install.bash | bash
```

*脚本会自动识别平台架构（amd64 / arm64），下载最新二进制并安装至系统 PATH（`/usr/local/bin` 或 `~/.local/bin`）。*

---

## 🚀 快速上手：两种核心接入方式

### 方式一：客户端执行模式 (本地运行)
适合日常使用者。程序在本地运行，本地掌控物理内存与生命周期：

```bash
# 1. 直接连入局域网/服务器烽火台（已默认禁用 UDP，自动补齐 :19001 端口）
mystic-chat -peer 10.20.13.115

# 2. 开启独立密令房间（端到端 AES-256-GCM 加密）
mystic-chat -peer 10.20.13.115 -room 光明顶禁地

# 3. 同台电脑多开互测（自动端口避让）
# 在同一台电脑打开多个终端窗口运行相同命令，程序自动退避端口互通
mystic-chat -peer 10.20.13.115
```

### 方式二：免安装 SSH 借道模式 (零环境依赖)
适合临时加入的同事。**本地电脑无需下载任何程序，无需配置任何环境**：

```bash
# 打开任意终端（Mac Terminal、PowerShell、Linux Shell 或 Termux）直接敲入：
ssh 10.20.13.115 -p 2222
```

- **瞬间入局**：按下回车直接进入武侠 TUI 即焚界面，自动分配随机行头；
- **同台互通**：与本地客户端完全互通，即焚倒计时、残卷瓦解与在线花名册完全一致；
- **了无痕迹**：关闭终端或按 `Ctrl+C` 退出，使用者电脑不留任何程序、缓存或历史记录。

---

## 🐧 部署中继“烽火台” (Linux 服务器)

在常开的服务器（如 `10.20.13.115`）上启动后台中继节点：

```bash
./mystic-chat -bootstrap -p2p-port 19001 -admin-key mySecretKey
```

*(如需配置为常驻 Systemd 系统服务，请参阅 [TIPS.md](./TIPS.md))*

---

## ⌨️ 常用快捷键与密令

| 快捷键 / 密令 | 作用说明 |
| :--- | :--- |
| `Enter` | 飞鸽传书（发送当前消息） |
| `Tab` / `F2` | 循环切换即焚时限（`15s` / `30s` / `60s` / `120s`） |
| `Ctrl+R` / `/reroll` | 施展易容术（随机更换门派名宿身份） |
| `Ctrl+L` / `/clear` | 天火焚书（立刻覆写清空当前屏幕与内存） |
| `Esc` / `Ctrl+C` | 避世归隐（内存物理抹除后安全退出） |
| `/room <暗号>` | 切换或进入端到端加密房间，输入 `/room` 退回大厅 |
| `/ttl <时限>` | 自定义即焚时限，例如 `/ttl 45s` 或 `/ttl 3m` |
| `/peers` | 查看当前已互通的同门侠客与烽火台列表 |
| `/killswitch <口令>` | [管理员] 广播天绝闭谷令，全网节点安全销毁并退出 |
| `/help` | 查看快捷键与密令指引 |

---

## ⚙️ 常用参数说明

```text
  -peer string       引导节点地址 (如 10.20.13.115，默认自动补全 :19001)
  -room string       江湖暗号 (开启端到端 AES-256-GCM 独立房间)
  -ttl string        消息即焚时限 (默认 "60s"，支持 15s/30s/60s/120s 等)
  -bootstrap         中继/引导节点模式 (适合部署在 Linux 服务器)
  -p2p-port int      P2P 网格 TCP 监听端口 (默认 19001)
  -ssh-port int      免客户端 SSH 借道端口 (默认 2222)
  -admin-key string  管理员紧急熔断口令 (配合 /killswitch 使用)
  -no-ssh            关闭嵌入式 SSH 服务
  -udp               启用局域网 UDP 嗅探广播 (默认关闭，纯 TCP 模式)
```

---

## 📚 延伸阅读与技术文档

- 🏛️ **[doc/architecture.md · 系统架构与技术实现文档](./doc/architecture.md)**  
  *深入解析系统拓扑图、TLS 1.3 动态握手、AES-256-GCM 载荷加密、纯内存倒计时失忆与风化引擎、PEX 花名册同步等底层实现。*
- 📜 **[TIPS.md · 江湖秘籍：进阶实战技巧手册](./TIPS.md)**  
  *涵盖 Systemd 常驻服务脚本、Wireshark 防抓包验证、同机多开隔离、断网穿透与安全熔断排查指南。*

---

## 📄 开源协议

本项目基于 [MIT License](./LICENSE) 开源。
