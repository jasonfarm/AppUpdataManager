# AppUpdateManager

软件自动更新系统：包含一个 Windows 客户端程序和一个 Ubuntu Web 服务端程序。

## 项目结构

```
appUpdateManager/
├── build-progress.json      # 构建进度跟踪
├── target.md                # 构建方案文档
├── server/
│   ├── backend/             # Go 后端
│   └── frontend/            # Vue3 + TS 管理控制台
├── client/                  # Windows 客户端（Fyne）
├── scripts/                 # 构建与部署脚本
└── docs/                    # 补充文档
```

## 技术栈

- **客户端**：Go + Fyne v2（跨平台 GUI，支持系统托盘）
- **服务端后端**：Go + Gin + WebSocket + SQLite
- **服务端前端**：Vue 3 + TypeScript + Vite + Element Plus

## 快速开始

### 服务端（Ubuntu）

```bash
cd server/backend
GOPROXY=https://proxy.golang.org go run ./cmd/server
```

前端开发：

```bash
cd server/frontend
npm install
npm run dev
```

生产构建：

```bash
./scripts/build-server.sh
```

### 客户端（Windows）

在 macOS 上交叉编译：

```bash
./scripts/build-client.sh
```

> 需要安装 mingw-w64：`brew install mingw-w64`

在 Windows 上直接开发：

```bash
cd client
go run .
```

## 默认账号

- 用户名：`admin`
- 密码：`123456`

账号配置文件：`server/backend/config/accounts.txt`

## 部署

详见 `docs/deploy.md`。

## 注意事项

- 客户端自我更新在 Windows 上通过生成 `updater.bat` 实现，需要管理员权限以确保能替换运行中的 exe。
- 开机自启动通过写入 Windows 注册表实现。
- 生产环境请修改 JWT 密钥（`server/backend/internal/middleware/auth.go`）和默认密码。
