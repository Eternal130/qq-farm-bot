# QQ经典农场 挂机脚本

基于 **Go + Vue 3** 的 QQ/微信 经典农场小程序自动化挂机脚本。通过分析小程序 WebSocket 通信协议（Protocol Buffers），实现全自动农场管理。

**⚠️ 警告：已有用户反馈封号，请低调使用，被举报必封。**

## ✨ v2.0 新版本特性

v2.0 版本已使用 **Go + Vue 3** 重写，新增：
- **Web 管理界面** — 现代化的前端界面，支持多账号管理
- **多账号管理** — 同时管理多个农场账号，一键启动/停止
- **实时日志** — WebSocket 实时推送运行日志
- **Dashboard 统计** — 账号状态、金币、升级预估一目了然
- **作物收益分析** — 分析作物经验效率，最优种植策略
- **QQ 扫码登录** — 支持 Web 界面扫码获取登录凭证
- **单文件部署** — 前端资源嵌入二进制，一个文件即可运行

## 🛡️ 功能特性

### 自己农场
- **自动收获** — 检测成熟作物并自动收获
- **自动铲除** — 自动铲除枯死/收获后的作物残留
- **自动种植** — 收获后自动购买种子并种植（按经验效率最优选种）
- **自动施肥** — 种植后自动施放普通肥料加速生长
- **自动除草** — 检测并清除杂草
- **自动除虫** — 检测并消灭害虫
- **自动浇水** — 检测缺水作物并浇水
- **自动出售** — 自动出售仓库中的果实
- **自动购肥** — 自动购买肥料
- **自动升地** — 自动升级和解锁土地

### 好友农场
- **好友巡查** — 自动巡查好友农场
- **帮忙操作** — 帮好友浇水/除草/除虫
- **自动偷菜** — 偷取好友成熟作物（可配置禁用）

### 系统功能
- **自动领取任务** — 自动领取完成的任务奖励
- **心跳保活** — 自动维持 WebSocket 连接

## 安装

### 方式一：直接下载（推荐）

从 [Releases](https://github.com/Eternal130/qq-farm-bot/releases) 下载对应平台的可执行文件。

### 方式二：从源码构建

```bash
git clone https://github.com/Eternal130/qq-farm-bot.git
cd qq-farm-bot

# 安装依赖并构建
make deps    # 安装前端和后端依赖
make all     # 构建前端 + 后端

# 或单独构建
make frontend  # 仅构建前端
make backend   # 仅构建后端
```

## 使用

### 启动服务

```bash
# 直接运行（首次运行会生成 config.json）
./qq-farm-bot

# 指定配置文件路径
./qq-farm-bot -config /path/to/config.json
```

启动后访问 http://localhost:18080 进入管理界面。

**默认账号**：admin / admin123（⚠️ 请在 config.json 中修改默认密码）

### 获取登录 Code

你需要从小程序中抓取 code。可以通过抓包工具（如 Fiddler、Charles、mitmproxy 等）获取 WebSocket 连接 URL 中的 `code` 参数。

**QQ 平台支持扫码登录**：在 Web 管理界面添加账号时选择「QQ 扫码登录」即可自动获取 code。

> [lkeme/QRLib](https://github.com/lkeme/QRLib) - 扫码登录使用此项目代码，非常感谢。

### Web 管理界面

1. **Dashboard** — 查看所有账号状态、总金币、运行中的 Bot 数量、升级预估
2. **账号管理** — 添加/编辑/删除账号，配置巡查间隔、是否偷菜等
3. **实时日志** — 查看每个账号的运行日志，支持 WebSocket 实时推送
4. **作物收益** — 分析各作物的经验效率，辅助种植决策

### 账号配置（每个账号可独立配置）

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `farm_interval` | 自己农场巡查间隔（秒） | 2 |
| `friend_interval` | 好友巡查间隔（秒） | 1 |
| `enable_steal` | 是否启用偷菜 | true |
| `force_lowest` | 强制种植最低等级作物 | false |
| `auto_start` | 服务启动时自动运行 | false |

### 配置文件

配置文件 `config.json`：

```json
{
  "listen": "0.0.0.0:18080",
  "jwt_secret": "请修改为随机字符串",
  "db_path": "data/farm.db",
  "admin_user": "admin",
  "admin_pass": "请修改默认密码",
  "game_server_url": "wss://gate-obt.nqf.qq.com/prod/ws",
  "client_version": "1.6.0.14_20251224"
}
```

### 后台运行

```bash
# Linux/Mac 使用 nohup
nohup ./qq-farm-bot > farm.log 2>&1 &

# 查看日志
tail -f farm.log

# 停止
pkill qq-farm-bot
```

## 📋 项目结构

```
├── cmd/server/main.go     # 入口文件
├── internal/
│   ├── api/               # HTTP API 路由
│   │   ├── router.go      # 路由配置
│   │   ├── account.go     # 账号管理 API
│   │   ├── bot.go         # Bot 控制 API
│   │   ├── dashboard.go   # Dashboard 统计 API
│   │   └── log.go         # 日志 API + WebSocket
│   ├── auth/              # JWT 认证
│   │   ├── jwt.go         # JWT 生成/验证
│   │   ├── handler.go     # 登录/注册处理
│   │   └── middleware.go  # 认证中间件
│   ├── bot/               # Bot 核心
│   │   ├── manager.go     # 多账号管理器
│   │   ├── instance.go    # 单个 Bot 实例
│   │   ├── network.go     # WebSocket 连接/消息编解码
│   │   ├── farm.go        # 农场操作: 收获/种植/施肥
│   │   ├── friend.go      # 好友农场: 偷菜/帮忙
│   │   ├── task.go        # 任务系统
│   │   ├── warehouse.go   # 仓库出售
│   │   └── qqlogin.go     # QQ 扫码登录
│   ├── config/            # 配置加载
│   ├── model/             # 数据模型
│   └── store/             # SQLite 存储
├── proto/                 # Protobuf 消息定义（Go 生成）
├── gameConfig/            # 游戏配置数据
│   ├── RoleLevel.json     # 等级经验表
│   └── Plant.json         # 植物数据
├── web/                   # Vue 3 前端
│   ├── src/
│   │   ├── views/         # 页面组件
│   │   │   ├── DashboardView.vue
│   │   │   ├── AccountsView.vue
│   │   │   ├── LogsView.vue
│   │   │   └── CropYieldView.vue
│   │   ├── api/           # API 调用
│   │   ├── stores/        # Pinia 状态管理
│   │   └── router/        # 路由配置
│   └── package.json
├── config.json            # 服务配置
├── Makefile               # 构建脚本
└── go.mod                 # Go 模块定义
```

## 📋 运行示例

```
========================================
  QQ农场管理后台
  监听地址: 0.0.0.0:18080
  管理账号: admin
  数据目录: /opt/qq-farm-bot/data
========================================

# Web 界面日志
[GIN] 2026/02/27 - 15:00:00 | 200 |   12.345µs | 192.168.1.100 | GET     "/"

# Bot 运行日志（可在 Web 界面实时查看）
[15:00:02] [农场] 收获 15/种植 15
[15:00:03] [施肥] 已为 15/15 块地施肥
[15:00:05] [农场] 除草2/除虫1/浇水3
[15:00:08] [好友] 小明: 偷6(白萝卜)
[15:00:10] [好友] 巡查 5 人 → 偷12/除草3/浇水2
[15:00:15] [仓库] 出售 2 种果实共 300 个，获得 600 金币
[15:00:20] [任务] 领取: 收获5次 → 金币500/经验100
```

## 💻️ 开发

```bash
# 开发模式：前后端分离运行
go run ./cmd/server/     # 启动后端 (http://localhost:18080)
cd web && npm run dev   # 启动前端开发服务器 (http://localhost:5173)

# 构建
make deps       # 安装依赖
make frontend   # 构建前端
make backend    # 构建后端（嵌入前端资源）
make all        # 完整构建
make clean      # 清理构建产物
```

## ⚠️ 注意事项

1. **登录 Code 有效期有限**，过期后需要重新获取
2. **请合理设置巡查间隔**，过于频繁可能触发服务器限流
3. **QQ 环境**下 code 支持多次使用
4. **WX 环境**下 code 不支持多次使用，请抓包时将 code 拦截掉
5. **⚠️ 修改默认密码**：部署后请立即修改 config.json 中的 admin_pass 和 jwt_secret

## 技术栈

- **后端**：Go 1.23+ / Gin / SQLite / WebSocket / JWT
- **前端**：Vue 3 / TypeScript / Element Plus / Vite / Pinia
- **协议**：Protocol Buffers / WebSocket

## 免责声明

本项目仅供学习和研究用途。使用本脚本可能违反游戏服务条款，由此产生的一切后果由使用者自行承担。

![Star History Chart](https://api.star-history.com/svg?repos=Eternal130/qq-farm-bot&type=Date&theme=light)

## License

MIT
