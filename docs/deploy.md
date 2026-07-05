# 部署文档

## 服务端部署（Ubuntu）

### 1. 准备环境

```bash
sudo apt update
sudo apt install -y golang-go nodejs npm build-essential
```

### 2. 构建

```bash
./scripts/build-server.sh
```

构建完成后，`dist/server/` 目录应包含：

- `appUpdateManager-server` — 后端可执行文件
- `accounts.txt` — 账号密码配置文件
- `static/index.html` 与 `static/assets/*` — Web 控制台静态资源

> 如果 `dist/server/static/index.html` 缺失，部署后访问控制台会出现 404。

### 3. 安装

#### 方式一：手动安装到 /opt/appupdatemanager

```bash
sudo mkdir -p /opt/appupdatemanager
sudo cp dist/server/appUpdateManager-server /opt/appupdatemanager/
sudo cp dist/server/accounts.txt /opt/appupdatemanager/
sudo cp -r dist/server/static /opt/appupdatemanager/
sudo mkdir -p /opt/appupdatemanager/data
sudo useradd -r -s /bin/false appupdate || true
sudo chown -R appupdate:appupdate /opt/appupdatemanager
```

#### 方式二：使用一键部署脚本

```bash
./scripts/deploy-server.sh --restart
```

默认部署到 `/home/th/work_dir/appupdatemanager`，可通过环境变量修改：

```bash
DEPLOY_REMOTE_DIR=/opt/appupdatemanager ./scripts/deploy-server.sh --restart
```

> 注意：`scripts/appupdatemanager.service` 中的 `WorkingDirectory` 必须与部署目录一致；否则即使文件已同步，systemd 服务仍会从旧目录启动，导致 404。

### 4. 配置 systemd

```bash
sudo cp scripts/appupdatemanager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable appupdatemanager
sudo systemctl start appupdatemanager
```

查看日志：

```bash
sudo journalctl -u appupdatemanager -f
```

### 5. 访问

打开浏览器访问 `http://<server-ip>:8080`，使用 `accounts.txt` 中的账号登录。

> 服务端默认监听 `0.0.0.0:8080`，即绑定所有网卡，因此局域网或公网中的其他电脑可以直接通过服务器 IP 访问。如果无法访问，请检查系统防火墙是否放行 8080 端口：
>
> ```bash
> sudo ufw allow 8080/tcp
> # 或云服务器安全组中放行 8080 端口
> ```

## 客户端部署（Windows）

### 1. 构建

在 macOS 上：

```bash
brew install mingw-w64
./scripts/build-client.sh
```

### 2. 分发

将生成的 `dist/client/appUpdateManager-client.exe` 复制到 Windows PC。

### 3. 运行

双击运行，首次运行打开设置页填写服务器地址、端口和客户端名称。

## 更新流程

1. 在 Web 控制台上传新的被控软件版本或客户端版本。
2. 将对应版本设为“最新”。
3. 在客户端列表中选择目标客户端，点击“更新软件”或“自我更新”。
4. 客户端接收到命令后自动下载、替换并运行新版本。

## 常见问题

### 客户端无法连接服务端

- 检查 Windows 防火墙是否放行客户端程序。
- 确认服务端端口 8080 已开放。
- 检查客户端设置中的服务器地址和端口。

### 自我更新失败

- 确保以管理员身份运行客户端。
- 检查目标目录是否有写入权限。
- 查看 `updater.bat` 是否成功生成。

### 前端页面空白或返回 404

- 确认已执行 `npm run build` 生成 `static` 目录。
- 检查后端是否正确配置了静态文件服务。
- 若使用 `deploy-server.sh` 部署，确认 `scripts/appupdatemanager.service` 中的 `WorkingDirectory` 与实际部署目录一致。
- 在服务器上检查 `static/index.html` 是否存在：

  ```bash
  ls -la <部署目录>/static/index.html
  ```
