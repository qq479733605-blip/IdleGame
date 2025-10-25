# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此代码仓库中工作时提供指导。

## 项目概述

这是一款以东方修仙为主题的放置类 MMORPG 服务端，搭配 Vue.js 前端。项目实现了一个修仙题材的放置游戏，玩家通过修炼不同的"修炼序列"来获取资源和经验。

## 技术架构

项目采用 **纯 Actor 驱动 + DDD（领域驱动设计）** 架构：

- **后端**: Go 1.25.2 + protoactor-go Actor 系统
- **前端**: Vue 3 + Vite + Pinia + Vue Router (需要 Node.js >=18.0.0)
- **通信方式**: WebSocket 实时通信，HTTP 用于身份验证
- **数据存储**: 基于 JSON 文件的存储（计划迁移到 Redis/PostgreSQL）

### 核心架构模式

1. **Actor 模型**: 所有游戏逻辑封装在独立的 Actor 中（GatewayActor, PlayerActor, SequenceActor, PersistActor）
2. **领域驱动设计**: 业务逻辑抽象为领域对象（Sequence, Formula, Item, Inventory）
3. **消息驱动**: 组件间仅通过消息传递通信，无共享内存
4. **异步持久化**: 通过 PersistActor 处理持久化，避免阻塞游戏逻辑
5. **配置驱动**: 所有游戏数据通过 JSON 配置表驱动

## 常用开发命令

### 前端 (idle-vue/)
```bash
cd idle-vue
npm install          # 安装依赖
npm run dev         # 启动开发服务器 (端口 5173)
npm run build       # 构建生产版本
npm run preview     # 预览生产构建
```

### 后端 (idlemmoserver/)
```bash
cd idlemmoserver
go mod tidy         # 下载依赖
go run cmd/server/main.go    # 启动服务器 (端口 8080)
```

### 测试
- **项目当前没有测试文件**
- 手动测试需要同时运行前后端服务器
- 测试流程：启动后端 → 启动前端 → 打开浏览器 → 登录 → 测试修炼序列

## 项目结构

```
idle-server/
├── idle-vue/                 # Vue.js 前端
│   ├── src/
│   │   ├── api/             # HTTP 和 WebSocket 客户端
│   │   ├── store/           # Pinia 状态管理
│   │   ├── views/           # Vue 组件/页面
│   │   └── router/          # Vue Router 配置
│   └── package.json
├── idlemmoserver/            # Go 后端
│   ├── cmd/server/          # 应用程序入口
│   ├── internal/
│   │   ├── actors/          # Actor 层（核心游戏逻辑）
│   │   ├── domain/          # 领域层（DDD 实体）
│   │   ├── gateway/         # HTTP + WebSocket 层
│   │   ├── persist/         # 持久化层
│   │   └── config/          # 配置
│   ├── saves/               # JSON 玩家存档文件
│   └── go.mod
└── README.md
```

## 核心游戏概念

### 修炼序列
- 玩家选择不同的修炼序列进行挂机修炼
- 例如：采药、采矿、炼丹、打坐
- 每个序列有独立的等级、经验和奖励
- 修炼序列在每个时间间隔产生资源、物品和奇遇事件

### Actor 系统
- **GatewayActor**: 处理 WebSocket/HTTP 连接和路由 (`internal/actors/gateway_actor.go`)
- **PlayerActor**: 管理玩家状态并协调其他 Actor (`internal/actors/player_actor.go`)
- **SequenceActor**: 处理修炼序列逻辑和时间计算 (`internal/actors/sequence_actor.go`)
- **PersistActor**: 异步保存/加载操作 (`internal/actors/persist_actor.go`)
- **SchedulerActor**: 统一时间调度 (`internal/actors/scheduler_actor.go`)
- **TeamActor**: 未来多人功能的队伍管理 (`internal/actors/team_actor.go`)

### 消息流程
1. 客户端通过 WebSocket 连接（带身份验证令牌）
2. GatewayActor 将消息路由到对应的 PlayerActor
3. PlayerActor 为活跃的修炼序列生成/管理 SequenceActor
4. SequenceActor 将时间结果发送给 PlayerActor
5. PlayerActor 转发给 PersistActor 进行异步保存
6. 结果通过 WebSocket 广播回客户端

## 关键文件及其作用

### 后端核心文件
- `cmd/server/main.go`: 服务器启动和 ActorSystem 初始化
- `internal/actors/messages.go`: 所有 Actor 消息定义（MessageFromWS, MsgClientPayload 等）
- `internal/actors/player_actor.go`: 玩家状态管理和 Actor 协调
- `internal/actors/sequence_actor.go`: 修炼序列时间逻辑和奖励计算
- `internal/actors/persist_actor.go`: 异步保存/加载操作，使用 JSON 仓库
- `internal/domain/sequence.go`: 修炼序列领域逻辑和进度系统
- `internal/domain/items.go`: 物品系统和库存管理
- `internal/domain/inventory.go`: 库存操作和背包管理
- `internal/domain/equipment.go`: 装备系统和装备处理
- `internal/gateway/connection.go`: WebSocket 连接处理
- `internal/gateway/http.go`: HTTP 身份验证端点，包含 CORS

### 前端核心文件
- `src/main.js`: Vue 应用初始化，包含 Pinia 和 Router
- `src/api/ws.js`: WebSocket 客户端，带自动重连和消息处理
- `src/api/http.js`: HTTP 客户端，用于身份验证和 API 调用
- `src/store/user.js`: 玩家状态管理（Pinia），响应式更新
- `src/views/MainView.vue`: 主游戏界面，包含修炼序列管理
- `src/views/LoginView.vue`: 身份验证界面
- `src/views/BagView.vue`: 库存和背包管理界面

### 配置文件
- `internal/domain/config.json`: 游戏配置表（修炼序列、物品、掉落、经验率）
- `internal/domain/equipment_config.json`: 装备系统配置
- `saves/`: 玩家 JSON 存档文件（当前为空，首次保存时创建）

## 开发指南

### 后端开发
- 所有 Actor 通信必须使用 protoactor-go 消息传递
- Actor 应该是单线程且尽可能无状态的
- 所有通信使用 `internal/actors/messages.go` 中的消息定义
- 游戏逻辑应该在领域层，而不是 Actor 层
- 所有持久化操作必须通过 PersistActor
- 配置更改应在 `internal/domain/config.json` 中进行
- 游戏内容遵循中文命名规范（修炼序列、灵草等）

### 前端开发
- 使用 Pinia 进行状态管理，响应式存储
- WebSocket 通信由 `src/api/ws.js` 处理，带自动重连
- 遵循 Vue 3 Composition API 模式
- 所有游戏 UI 应响应存储状态变化
- 通过 `src/api/http.js` 使用 axios 发起 HTTP 请求
- 中文 UI 文本应与后端游戏术语保持一致

### 消息协议
- 客户端消息流：WebSocket → GatewayActor → PlayerActor → 领域逻辑
- 使用 `messages.go` 中的结构化消息类型（MsgClientPayload 等）
- 所有 WebSocket 响应为 JSON 格式的游戏状态更新
- 身份验证通过 HTTP 端点处理，然后建立 WebSocket 连接

### 完整系统测试
1. 启动后端：`cd idlemmoserver && go run cmd/server/main.go`
2. 启动前端：`cd idle-vue && npm run dev`
3. 打开浏览器访问 `http://localhost:5173`
4. 登录并测试修炼序列功能
5. 检查浏览器控制台的 WebSocket 连接状态
6. 验证存档文件在 `saves/` 目录中创建

## 当前实现状态

项目处于第一阶段开发，核心挂机循环已实现：
- ✅ HTTP 身份验证端点
- ✅ WebSocket 实时通信，带自动重连
- ✅ 完整的 Actor 系统（Gateway, Player, Sequence, Persist, Scheduler actors）
- ✅ 修炼序列时间逻辑和奖励，包含奇遇事件
- ✅ 库存和背包管理系统
- ✅ 装备系统及配置
- ✅ JSON 文件持久化，异步保存
- ✅ 前端开发 CORS 支持

## 下一步开发重点

- 完成 SchedulerActor 集成，实现统一时间管理
- 修炼序列等级和进度系统
- 增强背包管理 UI 和命令
- 装备使用和属性修改
- 多人功能（第二阶段+）
- 数据库持久化迁移到 Redis/PostgreSQL（第四阶段）

## 附加文档

详细规范请参考以下文件：
- `README.md`: 综合项目文档（中文）
- `GAME_DESIGN_DOCUMENT_V2.md`: 详细游戏设计规范
- `MILESTONE_V1.md`: 第一阶段实现细节和需求
- `FUTURE_ROADMAP.md`: 五阶段开发路线图

MainView.vue:261
服务器错误: 子项目未解锁
ws.value.onmessage    @    MainView.vue:261  
居然会有这个错误，明明选的是最低级的子项目，然后现在游戏右上角的当前序列里，永远都是0重，然后从子窗口开始后，又没有进度条了，不知道是不是因为中间报错，导致的。因为重连的时候进度条又能正确运行，只不过还是比后端的结算推送慢。
我发现右上角的当前序列的层数不会改变。还想没有在序列结算的时候，更新这里，只更新了序列卡片里的层数 