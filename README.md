# 🧘 Idle Server - 修仙放置MMORPG微服务后端

> 一款以东方修仙为主题的放置类MMORPG服务端
> 采用 **微服务架构 + NATS消息队列 + Actor模型** 构建，
> 实现了高并发、高可扩展性的分布式游戏后端。

---

## 🎮 游戏概述

这是一款修仙题材的放置类游戏，玩家可以通过不同的"修炼序列"进行挂机修炼：
- 🧘 **打坐修炼** - 提升修为境界
- 🌿 **采药炼草** - 收集炼丹材料
- ⛏️ **灵矿采掘** - 开采珍贵矿石
- 🧪 **炼丹制药** - 炼制各种丹药
- ⚔️ **神兵锻造** - 打造强力装备
- 📜 **符箓制作** - 制作法术符箓

每个序列都有独立的等级系统、掉落奖励和奇遇事件，玩家可以随时切换修炼方式。

---

## 🏗️ 微服务架构

### 📊 服务组件

| 服务 | 端口 | 技术栈 | 职责 |
|------|------|--------|------|
| **Gateway** | 8006 | Gin + WebSocket | 客户端接入、消息路由 |
| **Auth** | 8081 | JWT + NATS | 用户认证、Token管理 |
| **Game** | 8082 | Actor + NATS | 游戏逻辑、状态管理 |
| **Persist** | 8083 | GORM + MySQL + Redis | 数据持久化、存储管理 |

### 🔄 架构特点

- **微服务独立部署**: 每个服务可独立开发、测试、部署
- **NATS消息通信**: 异步消息队列，服务解耦
- **Actor并发安全**: Game Service使用Actor模型保证并发安全
- **统一消息处理**: 所有服务使用统一的NATS处理框架

---

## 🚀 快速开始

### 方式一：一键启动（推荐）

```bash
# 进入项目目录
cd idle-server

# 启动所有微服务
./scripts/start_simplified.sh  # Linux/macOS
./scripts/start_simplified.bat # Windows
```

### 方式二：手动启动

1. **启动基础服务**
   ```bash
   # 启动NATS消息队列
   nats-server -p 4222

   # 启动MySQL数据库
   # Windows: net start mysql
   # Linux/macOS: sudo systemctl start mysql 或 brew services start mysql

   # 启动Redis缓存（可选但推荐）
   # Windows: net start redis
   # Linux/macOS: sudo systemctl start redis 或 brew services start redis
   ```

2. **启动微服务**（分别在不同终端）
   ```bash
   # Auth Service
   cd idlemmoserver/auth && go run main.go

   # Persist Service
   cd idlemmoserver/persist && go run main.go

   # Game Service
   cd idlemmoserver/game && go run main.go

   # Gateway Service
   cd idlemmoserver/gateway && go run main.go
   ```

3. **启动前端**
   ```bash
   cd idle-vue
   npm install
   npm run dev
   ```

### 🌐 访问地址

- **前端应用**: http://localhost:5173
- **API网关**: http://localhost:8006
- **WebSocket**: ws://localhost:8006/ws
- **健康检查**: http://localhost:8006/health

---

## ✅ 核心功能

### 🎮 游戏系统
- ✅ **9种修炼序列**: 打坐、采药、采矿、炼丹等完整修炼体系
- ✅ **序列等级系统**: 独立的等级提升和经验积累
- ✅ **装备系统**: 装备掉落、装备管理、属性加成
- ✅ **背包管理**: 24格背包系统，物品收集和管理
- ✅ **离线收益**: 智能的离线收益计算

### 🔧 技术架构
- ✅ **微服务架构**: Gateway、Auth、Game、Persist四大服务
- ✅ **NATS消息通信**: 服务间异步通信，完全解耦
- ✅ **统一NATS处理框架**: 消息处理器标准化
- ✅ **Actor并发模型**: Game Service使用Actor保证并发安全
- ✅ **JWT认证**: 安全的用户认证和授权

### 💾 数据持久化
- ✅ **MySQL数据库**: 主要数据存储，使用GORM ORM
- ✅ **Redis缓存**: 高性能数据缓存和排行榜
- ✅ **数据模型**: 用户、玩家、游戏进度完整模型
- ✅ **异步保存**: 非阻塞的数据持久化
- ✅ **事务支持**: 数据一致性和完整性保证

### 🌐 前端集成
- ✅ **Vue 3 + Vite**: 现代化前端技术栈
- ✅ **实时通信**: WebSocket双向通信
- ✅ **响应式UI**: Pinia状态管理
- ✅ **修仙主题**: 精美的中国风界面设计

---

## 📁 项目结构

```
idle-server/
├── idle-vue/                   # Vue 3 前端应用
│   ├── src/
│   │   ├── api/               # HTTP和WebSocket客户端
│   │   ├── store/             # Pinia状态管理
│   │   ├── views/             # Vue页面组件
│   │   └── router/            # 路由配置
│   └── package.json
├── idlemmoserver/              # Go微服务后端
│   ├── common/                # 共享代码库
│   │   ├── constants.go       # 常量定义
│   │   ├── messages.go        # 消息类型定义
│   │   ├── subjects.go        # NATS主题定义
│   │   ├── types.go           # 数据类型定义
│   │   ├── handler/           # 统一消息处理器
│   │   ├── nats/              # NATS管理器
│   │   ├── database/          # 数据库抽象层
│   │   └── service/           # 基础服务接口
│   ├── auth/                  # 认证服务 (8081)
│   ├── gateway/               # API网关 (8006)
│   ├── game/                  # 游戏服务 (8082)
│   ├── persist/               # 持久化服务 (8083)
│   └── go.work                # Go workspace配置
├── scripts/                   # 启动脚本
├── docs/                      # 文档目录
├── README.md                  # 项目主文档
├── CLAUDE.md                  # Claude Code开发指导
└── ARCHITECTURE.md            # 技术架构文档
```

## 🔧 技术栈

### 后端技术
- **Go 1.21+**: 主要开发语言
- **NATS**: 消息队列和服务通信
- **Gin**: HTTP Web框架
- **Gorilla/WebSocket**: WebSocket实现
- **GORM**: ORM数据库框架
- **MySQL**: 主数据库存储
- **Redis**: 缓存系统和排行榜
- **protoactor-go**: Actor模型框架

### 前端技术
- **Vue 3**: 现代化前端框架
- **Vite**: 快速构建工具
- **Pinia**: 状态管理
- **Vue Router**: 路由管理
- **Axios**: HTTP客户端
- **WebSocket**: 实时通信

## 📚 相关文档

- **[ARCHITECTURE.md](./ARCHITECTURE.md)**: 详细的技术架构说明
- **[CLAUDE.md](./CLAUDE.md)**: 开发指导和规范

## 🤝 贡献指南

1. **代码规范**: 遵循Go官方代码规范
2. **提交规范**: 使用清晰的commit信息
3. **文档更新**: 新功能需要更新相关文档
4. **测试要求**: 重要功能需要添加测试

## 📄 许可证

MIT License © 2025

---

> 🌟 **Idle Server** - 一个现代化的修仙放置MMORPG微服务后端，展示了如何在游戏开发中应用现代分布式架构设计。