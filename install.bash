#!/usr/bin/env bash
# ==============================================================================
# mystic-chat 一键极速安装脚本 (One-click Installer)
# 用法: curl -fsSL https://raw.githubusercontent.com/cronglu/mystic-chat/main/install.bash | bash
# ==============================================================================

set -e

REPO="cronglu/mystic-chat"
BINARY_NAME="mystic-chat"

echo "⚔️  正在安装 mystic-chat · 极客武侠即焚终端..."

# 1. 检测操作系统与架构
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ 不支持的 CPU 架构: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux|darwin)
        ;;
    *)
        echo "❌ 不支持的操作系统: $OS (仅支持 macOS 与 Linux)"
        exit 1
        ;;
esac

ASSET_NAME="mystic-chat-${OS}-${ARCH}"
echo "🔍 匹配目标平台: ${OS}/${ARCH} (${ASSET_NAME})"

# 2. 决定安装路径
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    if command -v sudo >/dev/null 2>&1; then
        USE_SUDO=true
    else
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        USE_SUDO=false
    fi
else
    USE_SUDO=false
fi

TARGET_PATH="${INSTALL_DIR}/${BINARY_NAME}"
TMP_DIR="$(mktemp -d)"
TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

# 3. 尝试下载预编译二进制，若尚无 Release 则尝试源码构建
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"
echo "📥 正在从 GitHub 获取最新二进制产物..."

DOWNLOAD_SUCCESS=false
if curl --head --silent --fail "$DOWNLOAD_URL" >/dev/null 2>&1; then
    if curl -fSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
        DOWNLOAD_SUCCESS=true
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = false ]; then
    echo "ℹ️  未找到已发布的 Release 资产，正在检测本地构建环境..."
    if command -v go >/dev/null 2>&1; then
        echo "🔨 发现 Go 环境，正在通过 Go 自动编译安装..."
        GOBIN="$TMP_DIR" go install -ldflags="-s -w" "github.com/${REPO}@latest"
        if [ -f "${TMP_DIR}/${BINARY_NAME}" ]; then
            DOWNLOAD_SUCCESS=true
        fi
    else
        echo "❌ 无法下载预编译二进制且本地未安装 Go 环境。"
        echo "💡 请前往 https://github.com/${REPO}/releases 手动下载对应架构二进制并放入 PATH。"
        exit 1
    fi
fi

# 4. 移动并赋予执行权限
chmod +x "$TMP_FILE"

echo "📦 正在部署二进制至 ${TARGET_PATH}..."
if [ "$USE_SUDO" = true ]; then
    sudo mv "$TMP_FILE" "$TARGET_PATH"
    sudo chmod +x "$TARGET_PATH"
else
    mv "$TMP_FILE" "$TARGET_PATH"
    chmod +x "$TARGET_PATH"
fi

echo "=================================================================="
echo "✅ mystic-chat 安装成功！"
echo ""
echo "🚀 客户端启动方法:"
echo "   mystic-chat                           # 本地体验模式"
echo "   mystic-chat -peer <烽火台IP>           # 连入局域网/服务器烽火台"
echo ""
echo "🚪 非客户端免安装执行方法 (同事无需安装任何程序):"
echo "   ssh <烽火台IP> -p 2222                 # 终端借道直接进入江湖密聊"
echo "=================================================================="

# PATH 提醒
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "⚠️  注意: $INSTALL_DIR 不在你的 PATH 环境变量中，请添加:"
    echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
fi
