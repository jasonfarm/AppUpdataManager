#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# ==================== 配置 ====================
SERVER_IP="${DEPLOY_SERVER_IP:-192.168.9.11}"
SERVER_USER="${DEPLOY_SERVER_USER:-th}"
REMOTE_DIR="${DEPLOY_REMOTE_DIR:-/home/th/work_dir/appupdatemanager}"
LOCAL_DIST="dist/server"
SSH_CONTROL_PATH="${HOME}/.ssh/appupdatemanager-deploy-%r@%h:%p"
# ==================== 配置结束 ====================

usage() {
    cat <<EOF
一键部署服务端到 Ubuntu 服务器

用法: $0 [选项]

默认配置：
  服务器地址: ${SERVER_IP}
  登录用户:   ${SERVER_USER}
  部署目录:   ${REMOTE_DIR}

可通过环境变量覆盖：
  DEPLOY_SERVER_IP      服务器 IP（默认 192.168.9.11）
  DEPLOY_SERVER_USER    登录用户名（默认 th）
  DEPLOY_REMOTE_DIR     远程部署目录（默认 /home/th/work_dir/appupdatemanager）

选项：
  -r, --restart    部署完成后尝试重启 systemd 服务 appupdatemanager
  -h, --help       显示此帮助

示例：
  ./scripts/deploy-server.sh
  DEPLOY_SERVER_IP=10.0.0.5 ./scripts/deploy-server.sh --restart
EOF
}

RESTART_SERVICE=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        -r|--restart) RESTART_SERVICE=true; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "未知选项: $1"; usage; exit 1 ;;
    esac
done

echo "==> 部署目标: ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}"

# 1. 先构建 Linux 服务端（包含后端二进制与 Web 前端静态资源）
echo "==> 正在构建服务端后端与 Web 前端（Linux AMD64）..."
./scripts/build-server-linux.sh

# 1.5 确认前端静态资源已生成
if [ ! -f "${LOCAL_DIST}/static/index.html" ]; then
    echo "ERROR: ${LOCAL_DIST}/static/index.html 不存在，构建脚本未正确生成前端静态资源。"
    exit 1
fi

# 2. 准备 SSH 控制通道，实现一次密码输入后复用连接
mkdir -p "$(dirname "$SSH_CONTROL_PATH")"
SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ControlPath="$SSH_CONTROL_PATH")

echo ""
echo "==> 正在建立 SSH 连接（只需输入一次密码）..."
ssh -o ControlMaster=yes -o ControlPersist=5m "${SSH_OPTS[@]}" -fN "${SERVER_USER}@${SERVER_IP}"

# 函数：通过已建立的通道执行远程命令
run_remote() {
    ssh "${SSH_OPTS[@]}" "${SERVER_USER}@${SERVER_IP}" "$@"
}

cleanup() {
    echo "==> 关闭 SSH 控制通道..."
    ssh -O exit "${SSH_OPTS[@]}" "${SERVER_USER}@${SERVER_IP}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# 3. 检查远程是否已存在账号密码配置文件
EXCLUDE_ACCOUNTS=""
echo "==> 检查远程账号密码配置文件..."
if run_remote "test -f ${REMOTE_DIR}/accounts.txt"; then
    echo "    发现 ${REMOTE_DIR}/accounts.txt，部署时将保留该文件，不会覆盖。"
    EXCLUDE_ACCOUNTS="--exclude=accounts.txt"
else
    echo "    未找到 ${REMOTE_DIR}/accounts.txt，将使用本地默认配置。"
fi

# 4. 确保远程目录存在
run_remote "mkdir -p ${REMOTE_DIR}"

# 5. 备份远程旧二进制（如果存在）
echo "==> 备份远程旧二进制（如存在）..."
run_remote "if [ -f ${REMOTE_DIR}/appUpdateManager-server ]; then cp ${REMOTE_DIR}/appUpdateManager-server ${REMOTE_DIR}/appUpdateManager-server.bak.$(date +%Y%m%d%H%M%S); fi"

# 6. 同步文件到服务器
#    使用 rsync + ControlPath，既支持增量同步，又能复用已建立的 SSH 通道
RSYNC_SSH="ssh ${SSH_OPTS[*]}"
echo "==> 正在同步文件到服务器..."
# shellcheck disable=SC2086
rsync -avz --delete ${EXCLUDE_ACCOUNTS} -e "${RSYNC_SSH}" "${LOCAL_DIST}/" "${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/"

# 7. 设置可执行权限
run_remote "chmod +x ${REMOTE_DIR}/appUpdateManager-server"

# 7.5 确认远程静态资源已就位
if ! run_remote "test -f ${REMOTE_DIR}/static/index.html"; then
    echo "ERROR: 远程 ${REMOTE_DIR}/static/index.html 不存在，部署同步可能失败。"
    exit 1
fi

# 8. 可选：重启 systemd 服务
if [ "$RESTART_SERVICE" = true ]; then
    echo "==> 尝试重启 appupdatemanager 服务..."
    if run_remote "systemctl is-active --quiet appupdatemanager" 2>/dev/null; then
        run_remote "sudo systemctl restart appupdatemanager" || {
            echo "    警告：重启服务失败，请检查服务状态或 sudo 权限。"
        }
    else
        echo "    appupdatemanager 服务未运行，跳过重启。"
    fi
fi

echo ""
echo "==> 部署完成：${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}"
echo "    服务端二进制与 Web 控制台静态资源已同步。"
if [ -n "$EXCLUDE_ACCOUNTS" ]; then
    echo "    已保留远程账号密码配置文件：${REMOTE_DIR}/accounts.txt"
fi
echo ""
echo "    注意：systemd 服务文件 (scripts/appupdatemanager.service) 中的 WorkingDirectory"
echo "    当前为 /opt/appupdatemanager。如果实际部署目录不是该路径，请同步修改服务"
echo "    文件或使用 DEPLOY_REMOTE_DIR=/opt/appupdatemanager 进行部署。"
