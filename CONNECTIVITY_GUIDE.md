# 🎯 前后端连接配置完成指南

## 📊 服务状态

✅ **所有服务运行正常！**

### 🌐 服务地址
- **前端 (Vue.js)**: http://localhost:5173
- **后端API (Gateway)**: http://localhost:8002
- **WebSocket连接**: ws://localhost:8002/ws
- **登录接口**: http://localhost:8002/login
- **注册接口**: http://localhost:8002/register
- **健康检查**: http://localhost:8002/health

## 🚀 启动服务

### 方法1: 使用提供的脚本
```bash
# 在项目根目录运行
./full_test.sh
```

### 方法2: 手动启动服务
```bash
# 启动前端
cd idle-vue
npm run dev

# 启动后端服务 (在新终端)
cd idlemmoserver

# 启动Gateway服务
go run ./gateway/main.go &

# 启动Login服务
go run ./login/main.go &

# 启动Game服务 (如果需要游戏功能)
go run ./game/main.go &

# 启动Persistence服务 (如果需要数据持久化)
go run ./persist/main.go &
```

## 🔌 API端点

### 认证相关
- `POST /login` - 用户登录
  ```json
  {
    "username": "your_username",
    "password": "your_password"
  }
  ```

- `POST /register` - 用户注册
  ```json
  {
    "username": "new_username",
    "password": "new_password"
  }
  ```

### 健康检查
- `GET /health` - 服务健康状态

## 🎮 使用流程

1. **访问前端**: 打开浏览器访问 http://localhost:5173
2. **注册用户**: 在登录页面点击注册，创建新账户
3. **登录游戏**: 使用注册的账户登录
4. **开始游戏**: 登录成功后自动进入游戏界面

## 🌐 CORS跨域支持

✅ **已解决跨域问题！**

后端服务现在完全支持跨域请求：
- `Access-Control-Allow-Origin: *` - 允许所有来源
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS` - 允许常用HTTP方法
- `Access-Control-Allow-Headers: Content-Type, Authorization` - 允许常用请求头
- `Access-Control-Max-Age: 86400` - 预检请求缓存24小时

### 测试跨域请求
```bash
# 测试预检请求
curl -H "Origin: http://localhost:5173" -X OPTIONS http://localhost:8002/login

# 测试实际请求
curl -H "Origin: http://localhost:5173" -X POST \
     -H "Content-Type: application/json" \
     -d '{"username":"test","password":"123"}' \
     http://localhost:8002/login
```

## 📝 技术架构

- **前端**: Vue.js 3 + Vite + Pinia
- **后端**: Go + Proto.Actor + NATS消息队列
- **通信**: WebSocket实时连接 + HTTP REST API
- **架构**: 微服务架构 (Gateway, Login, Game, Persistence)

## 🔧 故障排除

### 如果前端无法连接后端
1. 检查后端服务是否正在运行
2. 确认端口8002和8001没有被其他程序占用
3. 查看浏览器控制台的错误信息

### 如果登录失败
1. 检查Login服务是否在端口8001运行
2. 测试直接访问: http://localhost:8001/health
3. 查看后端日志文件

### 如果WebSocket连接失败
1. 确认Gateway服务正在端口8002运行
2. 检查防火墙设置
3. 确认获取到了有效的登录token

## 📋 日志文件位置

```
idlemmoserver/logs/
├── gateway_final.log      # Gateway服务日志
├── login_final_working.log # Login服务日志
├── game.log               # Game服务日志
└── persist.log            # Persistence服务日志
```

---

🎉 **恭喜！前后端连接配置完成，现在可以开始使用游戏了！**