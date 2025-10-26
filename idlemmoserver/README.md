# 🎮 IdleMMO 微服务后端

基于 **NATS + Actor 模型** 的分布式修仙放置游戏后端

---

## 🏗️ 项目架构

```
idlemmoserver/
├── common/             # 🧩 公共模块 - 消息、类型、工具
├── login/              # 🔐 登录服务 - 用户认证 (端口:8001)
├── gateway/            # 🌐 网关服务 - WebSocket接入 (端口:8002)
├── game/               # 🎮 游戏服务 - 游戏逻辑 (端口:8003)
├── persist/            # 💾 持久化服务 - 数据存储 (端口:8004)
├── scripts/            # 🚀 部署脚本
├── saves/              # 💾 玩家数据存储
├── go.work             # 📦 Go workspace配置
└── README.md           # 📖 项目文档
```

---

## 🚀 快速启动

### 前置要求
- Go 1.21+
- NATS Server

### 1. 安装并启动NATS
```bash
# macOS
brew install nats-server && nats-server -p 4222

# Linux
sudo apt-get install nats-server && nats-server -p 4222

# Windows
# 下载并运行 nats-server.exe
```

### 2. 启动所有服务
```bash
# Linux/macOS
./scripts/run_all.sh

# Windows
scripts\run_all.bat
```

### 3. 访问服务
- **Gateway**: http://localhost:8002
- **WebSocket**: ws://localhost:8002/ws
- **健康检查**: http://localhost:8002/health

---

## 📁 服务说明

### 🔐 Login Service (端口:8001)
- 用户注册和登录
- Token生成和验证
- 用户数据管理

### 🌐 Gateway Service (端口:8002)
- WebSocket连接管理
- 消息路由和转发
- 客户端状态管理

### 🎮 Game Service (端口:8003)
- 游戏核心逻辑
- 修炼序列系统
- Actor并发管理

### 💾 Persist Service (端口:8004)
- 玩家数据持久化
- JSON文件存储
- 异步数据操作

---

## 🔧 开发指南

### 添加新模块
1. 创建模块目录：`mkdir new_module`
2. 创建 `go.mod` 文件
3. 更新 `go.work` 添加新模块
4. 实现服务逻辑

### 构建项目
```bash
./scripts/build_all.sh
```

### 停止服务
```bash
./scripts/stop_all.sh
```

---

## 📊 服务通信

所有服务通过 **NATS** 消息总线进行通信：

- **login.*** - 登录相关消息
- **game.*** - 游戏逻辑消息
- **persist.*** - 数据持久化消息
- **gateway.*** - 网关广播消息

---

## 📋 日志和监控

```
logs/
├── login.log       # 登录服务日志
├── gateway.log     # 网关服务日志
├── game.log        # 游戏服务日志
└── persist.log     # 持久化服务日志
```

---

## 🎯 核心特性

- ✅ **微服务架构**: 独立部署、扩展、容错
- ✅ **Actor模型**: 并发安全、高性能
- ✅ **NATS通信**: 异步消息、解耦合
- ✅ **WebSocket**: 实时通信、自动重连
- ✅ **模块化设计**: 高内聚、低耦合

---

## 🔗 相关文档

- [REFACTOR_COMPLETE.md](./REFACTOR_COMPLETE.md) - 详细重构文档
- [modular_actor_project_design.md](../modular_actor_project_design.md) - 原始设计文档

---

> 🎮 **IdleMMO** - 东方修仙题材的放置类MMORPG微服务后端