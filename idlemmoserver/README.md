# 🎮 IdleMMO 微服务后端

基于 **NATS + Actor 模型** 的分布式修仙放置游戏后端

---

## 🏗️ 项目架构

```
idlemmoserver/
├── common/             # 🧩 公共模块 - 消息、类型、工具
├── auth/               # 🔐 认证服务 - 用户认证 & JWT (端口:8001)
├── gateway/            # 🌐 网关服务 - WebSocket接入 (端口:8005)
├── game/               # 🎮 游戏服务 - 游戏逻辑 & PlayerActor (端口:8003)
├── persist/            # 💾 持久化服务 - 数据存储 & JSON仓库 (端口:8004)
├── scripts/            # 🚀 部署脚本
├── saves/              # 💾 玩家数据存储
├── go.work             # 📦 Go workspace配置
├── AUTH_REFACTOR_PROGRESS.md  # 📝 认证重构进度报告
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
brew install nats-server && nats-server -p 4223

# Linux
sudo apt-get install nats-server && nats-server -p 4223

# Windows
# 下载并运行 nats-server.exe -p 4223
``

### 2. 启动所有服务
```bash
# Linux/macOS
./scripts/run_all.sh

# Windows
scripts\run_all.bat
```

### 3. 访问服务
- **Gateway**: http://localhost:8005
- **WebSocket**: ws://localhost:8005/ws
- **健康检查**: http://localhost:8005/health

### 4. 当前系统状态 ✅
- **NATS Server**: 端口4223 - 运行正常
- **Auth Service**: 端口8001 - NATS通信正常
- **Gateway Service**: 端口8005 - Web API和WebSocket正常
- **Game Service**: 端口8003 - Actor模型正确
- **Persist Service**: 端口8004 - 数据持久化正常
- **用户注册**: ✅ 正常工作，返回PlayerId
- **用户登录**: ✅ 正常工作，返回JWT token

---

## 📁 服务说明

### 🔐 Auth Service (端口:8001) - **已重构** ✅
- 用户注册和登录（基于NATS分布式通信）
- JWT Token生成和验证
- 用户数据管理和持久化
- NATS消息处理和回复

### 🌐 Gateway Service (端口:8005) - **已更新** ✅
- WebSocket连接管理和认证
- HTTP API请求路由
- NATS服务间通信代理
- 客户端状态管理

### 🎮 Game Service (端口:8003) - **Actor模型修复** ✅
- 游戏核心逻辑和修炼序列系统
- PlayerActor生命周期管理
- Actor并发通信（修复了ActorSystem问题）
- NATS消息处理和PlayerActor管理

### 💾 Persist Service (端口:8004) - **功能扩展** ✅
- 玩家和用户数据持久化
- JSON文件存储和管理
- 异步数据操作
- NATS消息处理和回复（修复了回复机制）

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

- [AUTH_REFACTOR_PROGRESS.md](./AUTH_REFACTOR_PROGRESS.md) - 认证服务重构进度报告
- [modular_actor_project_design.md](../modular_actor_project_design.md) - 原始设计文档

---

## 📝 重构日志

### 2025-10-27: Auth服务分布式重构 ✅
- **重构范围**: Login → Auth 服务完整重构
- **架构变化**: 内存存储 → NATS分布式通信
- **核心改进**:
  - 添加用户数据消息类型和NATS主题
  - 扩展Persist服务支持用户数据存储
  - 修复NATS回复机制，解决超时问题
  - 修复Game服务ActorSystem问题
- **验证结果**: 用户注册/登录API正常工作，JWT token生成正确

### 🔄 待完善事项
- [ ] Auth服务NATS消息Actor模式重构
- [ ] Game服务配置文件路径修复
- [ ] 完整游戏功能端到端测试
- [ ] 前端WebSocket连接验证

---

> 🎮 **IdleMMO** - 东方修仙题材的放置类MMORPG微服务后端