# 软件自动更新系统 — 构建方案

> 本文档根据 `plan.txt` 整理，用于指导从零开始构建项目。建议通读后再开始动手；每完成一个步骤请在 `build-progress.json` 中标记进度，以便随时中断和恢复。

---

## 1. 项目概述

构建一个“软件自动更新系统”，包含两部分：

| 部分 | 运行环境 | 技术栈 | 主要职责 |
|------|---------|--------|----------|
| 客户端 | Windows | Go + Fyne v2 | 带 UI 和托盘图标，管理一个被控 exe 的版本，与服务器通信完成下载、替换、启动/停止，并支持自我更新 |
| 服务端 | Ubuntu | Go + Gin + WebSocket + SQLite + Vue3/TS | Web 控制台管理版本、客户端、触发更新，接收客户端心跳和状态上报 |

开发环境统一使用 VSCode；客户端可在 macOS 上交叉编译为 Windows 可执行文件。

---

## 2. 技术栈选型说明

### 2.1 客户端

- **GUI 框架：Fyne v2**
  - 纯 Go 跨平台 GUI，无需额外前端工程
  - 支持 Windows 系统托盘、最小化到托盘、菜单
  - 支持 macOS → Windows 交叉编译
  - VSCode 即可完整开发
- **进程管理**：标准库 `os/exec` + `golang.org/x/sys/windows`
- **自启动**：写入 Windows 注册表或启动菜单
- **自我更新**：下载 `.exe.new` → 生成临时批处理脚本 → 退出自身 → 脚本替换并重启

### 2.2 服务端

- **后端：Go + Gin**
  - 模块化目录，便于阅读和维护
  - WebSocket 用于服务端主动推送更新命令
- **数据库：SQLite**
  - 轻量，无需独立服务，适合本项目规模
- **前端：Vue 3 + TypeScript + Vite + Element Plus**
  - 组件丰富，控制台美观、操作简单
- **文件上传**：本地文件系统存储上传的客户端/被控软件文件

---

## 3. 项目目录结构

建议按以下结构初始化仓库：

```
appUpdateManager/
├── build-progress.json          # 构建进度跟踪文件（必须维护）
├── README.md                    # 项目说明与快速启动
├── docs/                        # 补充文档
│   ├── api.md
│   └── client-protocol.md
├── scripts/
│   ├── build-client.sh          # macOS 交叉编译客户端脚本
│   ├── build-server.sh          # 编译服务端脚本
│   └── package-server.sh        # 服务端打包脚本
├── client/                      # 客户端（Fyne）
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── build/
│   └── internal/
│       ├── config/              # 本地配置读写
│       ├── server/              # 与后端通信（HTTP + WS）
│       ├── updater/             # 自我更新逻辑
│       ├── software/            # 被控 exe 管理
│       ├── systray/             # 托盘与主窗口控制
│       ├── autostart/           # 开机自启动
│       └── sysinfo/             # 系统信息收集
└── server/
    ├── backend/                 # Go 后端
    │   ├── cmd/server/main.go
    │   ├── go.mod
    │   ├── go.sum
    │   ├── config/
    │   │   ├── config.go
    │   │   └── accounts.txt     # 账号密码配置文件
    │   ├── internal/
    │   │   ├── api/             # HTTP 处理器
    │   │   ├── ws/              # WebSocket 连接管理
    │   │   ├── service/         # 业务逻辑
    │   │   ├── model/           # 数据模型
    │   │   ├── store/           # SQLite 数据访问
    │   │   ├── middleware/      # 认证、日志
    │   │   └── upload/          # 上传文件处理
    │   └── static/              # 内嵌前端产物
    └── frontend/                # Vue3 + TS 管理控制台
        ├── index.html
        ├── package.json
        ├── vite.config.ts
        ├── tsconfig.json
        └── src/
            ├── main.ts
            ├── App.vue
            ├── router/
            ├── stores/
            ├── api/
            ├── views/
            └── components/
```

---

## 4. 构建阶段与核心任务

总体分为 6 个阶段，建议按顺序执行。每个阶段包含若干原子任务，任务 ID 与 `build-progress.json` 中保持一致。

### 阶段 1：环境准备与项目脚手架

| 任务 ID | 任务内容 | 验收标准 |
|---------|----------|----------|
| P1-T1 | 安装 Go、Node.js、Git，验证 Fyne 模块可下载 | `go version`、`node -v` 正常，`go get fyne.io/fyne/v2` 成功 |
| P1-T2 | 初始化服务端后端模块 `server/backend/go.mod` | 模块名为 `github.com/yourname/appUpdateManager/server` |
| P1-T3 | 初始化服务端前端 `server/frontend`（Vite + Vue3 + TS） | `npm install` 成功，dev 服务器可启动 |
| P1-T4 | 初始化客户端 Fyne 项目 `client/` | `go mod init` 完成，目录结构正确 |
| P1-T5 | 创建 `build-progress.json` 并配置跟踪机制 | 文件存在，后续每完成一步手动更新 |

### 阶段 2：服务端后端开发

| 任务 ID | 任务内容 | 验收标准 |
|---------|----------|----------|
| P2-T1 | 设计并实现 SQLite 数据模型与表结构 | 数据库初始化脚本可运行，表正常创建 |
| P2-T2 | 实现配置加载与 `accounts.txt` 账号读取 | 启动时能从 txt 读取账号并支持登录校验 |
| P2-T3 | 实现 JWT/Session 登录认证中间件 | `/api/login` 可登录，受保护接口需认证 |
| P2-T4 | 实现软件版本管理 API（CRUD、设置最新版本） | Postman 测试通过 |
| P2-T5 | 实现文件上传 API（客户端程序、被控软件） | 上传文件保存到指定目录并记录数据库 |
| P2-T6 | 实现 WebSocket Hub 与客户端连接管理 | 客户端可连接 `/ws`，服务端能识别客户端 |
| P2-T7 | 实现客户端管理 API 与命令下发（立即更新、指定版本更新、启停） | API 能将命令推送到对应客户端 |
| P2-T8 | 实现客户端心跳与状态上报处理 | 客户端上报的状态能正确入库并展示 |

### 阶段 3：服务端前端开发

| 任务 ID | 任务内容 | 验收标准 |
|---------|----------|----------|
| P3-T1 | 搭建 Element Plus、路由、Pinia、Axios | 页面可正常跳转，主题统一 |
| P3-T2 | 实现登录页面 | 登录成功跳转，失败提示 |
| P3-T3 | 实现软件版本管理页面 | 列表、上传、删除、设为最新、修改名称 |
| P3-T4 | 实现客户端列表页面 | 显示版本、状态、运行时长、系统信息 |
| P3-T5 | 实现客户端控制面板 | 批量/单个触发更新、启动、停止、重启 |
| P3-T6 | 前端打包并配置后端静态资源服务 | `npm run build` 产物可被 Go 静态文件服务托管 |

### 阶段 4：客户端程序开发

| 任务 ID | 任务内容 | 验收标准 |
|---------|----------|----------|
| P4-T1 | 实现本地配置读写（服务器地址、端口、客户端名称、版本号） | 配置可持久化，重启后保留 |
| P4-T2 | 实现客户端 UI（设置页面、状态展示） | `go run main.go` 可看到 Fyne 窗口并可交互 |
| P4-T3 | 实现系统托盘（最小化、显示主窗口、退出） | 最小化后在托盘显示，右键/单击可操作 |
| P4-T4 | 实现开机自启动开关 | 开启后在 Windows 启动项中可见 |
| P4-T5 | 实现与服务端的 HTTP/WebSocket 通信 | 能注册、心跳、接收命令 |
| P4-T6 | 实现被控 exe 的下载、替换、启动、停止、重启 | 服务端触发后客户端能完成软件更新并运行 |
| P4-T7 | 实现客户端自我更新 | 下载新版本并在退出后替换自身 |
| P4-T8 | 实现系统信息收集（IP、系统版本、内存、CPU） | 服务端可正确展示 |

### 阶段 5：集成测试

| 任务 ID | 任务内容 | 验收标准 |
|---------|----------|----------|
| P5-T1 | 本地端到端测试：上传软件 → 触发更新 → 客户端下载运行 | 客户端 PC 上目标软件被正确替换启动 |
| P5-T2 | 测试客户端自我更新流程 | 旧版本退出，新版本启动，版本号更新 |
| P5-T3 | 测试多客户端同时控制 | 服务端可批量选择多个客户端下发更新 |
| P5-T4 | 测试异常场景：断网、重启、文件损坏 | 系统能恢复或给出明确错误提示 |

### 阶段 6：打包与部署

| 任务 ID | 任务内容 | 验收标准 |
|---------|----------|----------|
| P6-T1 | 编写 macOS → Windows 交叉编译脚本 | 在 Mac 上运行后产出 `client.exe` |
| P6-T2 | 编写 Ubuntu 服务端编译与打包脚本 | 在 Ubuntu 上运行后产出可执行服务端 + 前端产物 |
| P6-T3 | 编写 systemd 服务文件（服务端后台运行） | Ubuntu 开机自启稳定运行 |
| P6-T4 | 输出部署文档与常见问题排查 | 文档放入 `docs/` |

---

## 5. 构建进度跟踪机制

为避免构建中断后从头开始，必须在项目根目录维护 `build-progress.json`。

### 5.1 文件位置

```
appUpdateManager/build-progress.json
```

### 5.2 更新规则

1. **开始一个任务前**，将该任务状态改为 `in_progress`。
2. **完成任务后**，将状态改为 `completed`，并填写 `completedAt`。
3. **若任务被阻塞**，改为 `blocked`，在 `notes` 中写明原因。
4. **每次修改后**，更新顶层 `lastUpdated` 字段。
5. **恢复构建时**，先读取本文件，找到第一个未完成的阶段/任务，从该处继续。

### 5.3 状态定义

- `not_started`：未开始
- `in_progress`：进行中
- `completed`：已完成
- `blocked`：被阻塞

### 5.4 恢复构建流程

```bash
# 1. 进入项目目录
cd /Users/th/Documents/golang/appUpdateManager

# 2. 查看 build-progress.json，定位当前进度
# 3. 从第一个 not_started / in_progress / blocked 的任务继续
# 4. 完成后更新 build-progress.json
```

---

## 6. 关键设计细节

### 6.1 服务端数据库表结构（SQLite）

```sql
-- 用户账号
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL
);

-- 客户端程序版本
CREATE TABLE client_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    is_latest BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 被控软件版本
CREATE TABLE software_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    is_latest BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 在线/注册客户端
CREATE TABLE clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    client_version TEXT,
    software_version TEXT,
    status TEXT,              -- online/offline/running/error
    is_running BOOLEAN DEFAULT 0,
    ip TEXT,
    os_version TEXT,
    memory TEXT,
    cpu TEXT,
    last_seen DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 下发给客户端的命令队列
CREATE TABLE client_commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id INTEGER NOT NULL,
    command_type TEXT NOT NULL,  -- update_self / update_software / start / stop / restart
    payload TEXT,                -- JSON，如 {"version":"1.2.3"}
    status TEXT DEFAULT 'pending', -- pending / sent / done / failed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 6.2 服务端核心 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/login | 登录，返回 Token |
| GET  | /api/me | 当前登录用户 |
| GET  | /api/software | 被控软件版本列表 |
| POST | /api/software | 新增版本 |
| DELETE | /api/software/:id | 删除版本 |
| POST | /api/software/:id/latest | 设为最新版本 |
| GET  | /api/client-versions | 客户端程序版本列表 |
| POST | /api/client-versions | 上传客户端新版本 |
| POST | /api/client-versions/:id/latest | 设为客户端最新版本 |
| GET  | /api/clients | 客户端列表 |
| GET  | /api/clients/:id | 客户端详情 |
| POST | /api/clients/:id/update-software | 触发更新被控软件 |
| POST | /api/clients/:id/update-self | 触发客户端自我更新 |
| POST | /api/clients/:id/start | 启动被控软件 |
| POST | /api/clients/:id/stop | 停止被控软件 |
| POST | /api/clients/:id/restart | 重启被控软件 |
| WS   | /ws | 客户端长连接 |

### 6.3 客户端与服务端通信协议

客户端通过 WebSocket 连接后，周期性发送心跳与状态：

```json
// 客户端心跳/上报
{
  "type": "heartbeat",
  "data": {
    "name": "client-001",
    "client_version": "1.0.0",
    "software_version": "2.1.0",
    "is_running": true,
    "ip": "192.168.1.100",
    "os_version": "Windows 10 Pro",
    "memory": "16GB",
    "cpu": "Intel i7-9700"
  }
}
```

服务端下发命令：

```json
{
  "type": "command",
  "data": {
    "command": "update_software",
    "version": "2.2.0",
    "download_url": "http://server/files/software_2.2.0.exe"
  }
}
```

### 6.4 客户端自我更新方案

1. 接收到 `update_self` 命令。
2. 下载新客户端到 `client.exe.new`。
3. 生成 `updater.bat`：
   ```batch
   @echo off
   timeout /t 2 /nobreak >nul
   move /Y "client.exe.new" "client.exe"
   start "" "client.exe"
   del "%~f0"
   ```
4. 执行 `updater.bat` 后客户端进程退出。
5. 批处理完成替换并启动新版本，随后自删除。

### 6.5 被控软件更新方案

1. 客户端接收到 `update_software` 命令，包含目标版本与下载地址。
2. 下载目标 exe 到本地版本目录，例如 `software/v2.2.0/app.exe`。
3. 若当前软件正在运行，则调用 `taskkill` 或优雅终止。
4. 启动新版本 exe。
5. 上报新版本状态与运行信息。

### 6.6 账号配置文件 `accounts.txt` 格式

```
# 每行一个账号，格式：用户名:密码
admin:123456
operator:operator123
```

服务端启动时读取此文件并同步到 `users` 表。修改后需重启服务端生效（后续可扩展为热重载）。

---

## 7. 常用构建命令

### 7.1 服务端后端

```bash
cd server/backend
go mod tidy
go run cmd/server/main.go
```

### 7.2 服务端前端

```bash
cd server/frontend
npm install
npm run dev
npm run build
```

### 7.3 客户端开发

```bash
cd client
go run main.go
```

### 7.4 客户端交叉编译（macOS → Windows）

```bash
cd client
GOOS=windows GOARCH=amd64 go build -o appUpdateManager-client.exe
# 产物在当前目录
```

---

## 8. 注意事项

1. **Windows 权限**：客户端修改注册表实现开机自启动、终止其他进程等操作可能需要管理员权限，测试时请以管理员运行。
2. **文件占用**：Windows 下无法直接覆盖正在运行的 exe，自我更新必须使用“先退出、再替换、再启动”的批处理方案。
3. **路径处理**：客户端所有路径统一使用 `filepath` 包处理，避免反斜杠问题。
4. **版本号规范**：统一采用语义化版本 `MAJOR.MINOR.PATCH`，例如 `1.2.3`。
5. **安全第一**：生产环境不要明文存储密码，`accounts.txt` 仅用于开发/内部场景，后续应迁移到数据库加盐哈希。
6. **进度维护**：务必在每次完成/开始任务时更新 `build-progress.json`，这是本方案支持断点续建的关键。

---

## 9. 下一步行动

1. 确认本方案是否符合预期；如有调整请在 `target.md` 上直接批注或告知。
2. 若方案通过，从 `build-progress.json` 中将 `P1-T1` 状态改为 `in_progress`，开始安装环境。
3. 按照阶段顺序逐步实施，随时更新进度文件。
