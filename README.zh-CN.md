# Chat Server

一个用于个人学习的聊天服务项目，基于 **Go、WebSocket、PostgreSQL、Redis、Docker Compose 和 Nginx** 构建。

这个项目不是生产级 IM 系统，也不是完整聊天产品。它的目标是通过一个小而完整的聊天场景，练习后端工程中的常见链路：HTTP API、认证、WebSocket 长连接、在线状态、跨实例消息投递、离线消息存储、容器化部署和统一反向代理入口。

默认英文版文档：[README.md](README.md)

## 功能

- 用户注册和登录
- bcrypt 密码哈希存储
- JWT 登录态签发和 WebSocket 鉴权
- WebSocket 长连接
- 单聊消息发送
- 获取当前服务实例上的在线用户列表
- 基于 Redis Pub/Sub 的跨实例消息投递
- 基于 PostgreSQL 的离线消息存储与拉取
- WebSocket ping/pong 心跳和读写超时控制
- 同一用户单端在线策略，包括跨实例踢下线
- HTTP API 统一 JSON 响应格式
- 基础请求参数和 WebSocket 消息校验

## 技术栈

- Go 1.25
- `net/http`
- `gorilla/websocket`
- PostgreSQL 15
- `pgx/v5`
- Redis 7
- `go-redis/v9`
- bcrypt
- JWT
- Docker Compose
- Nginx

## 架构

```text
浏览器 / 客户端
      |
      v
    Nginx
   /     \
  v       v
chat-1  chat-2
   \     /
    Redis
      |
      v
 PostgreSQL
```

Nginx 对外暴露统一入口 `80` 端口，并把 HTTP 和 WebSocket 请求转发到两个 chat-server 实例。

每个 chat-server 实例只管理本实例内的 WebSocket 连接。Redis 用来保存在线状态并在多个实例之间广播消息。PostgreSQL 用来保存用户和离线消息。

## 项目结构

```text
.
|-- README.md
|-- README.zh-CN.md
|-- docker-compose.yml
|-- nginx.conf
|-- init.sql
|-- index.html
|-- go.work
|-- chat-server/
|   |-- Dockerfile
|   |-- go.mod
|   |-- go.sum
|   |-- main.go
|   |-- handlers.go
|   |-- hub.go
|   |-- db.go
|   |-- redis.go
|   |-- auth.go
|   |-- response.go
|   |-- models.go
|   `-- auth_test.go
`-- 云服开发文档v2.0.md
```

## 快速开始

### 前置条件

- Docker
- Docker Compose

### 使用 Docker Compose 启动

```bash
docker compose up --build
```

服务启动后：

- Web 页面：`http://localhost/`
- HTTP API 入口：`http://localhost/`
- WebSocket 入口：`ws://localhost/ws?token=<jwt>`

Compose 会启动：

- `postgres`
- `redis`
- `chat-server-1`
- `chat-server-2`
- `nginx`

### 停止服务

```bash
docker compose down
```

如果需要同时删除 PostgreSQL 数据卷：

```bash
docker compose down -v
```

## 本地开发

如果本地已经有 PostgreSQL 和 Redis，可以直接从 `chat-server` 模块启动 Go 服务：

```bash
cd chat-server

$env:DB_URL="postgres://postgres:123456@localhost:5432/chatdb?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:INSTANCE_ID="server-1"
$env:JWT_SECRET="chat-server-local-dev-secret"

go run .
```

macOS/Linux 请使用 `export` 设置环境变量。

服务监听 `:8080`。

## 配置项

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DB_URL` | 空 | PostgreSQL 连接字符串，启动时必须可用。 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址。 |
| `INSTANCE_ID` | `server-1` | 服务实例 ID，用于 Redis 在线状态记录。 |
| `JWT_SECRET` | `dev-secret-change-me` | JWT 签名密钥。多实例部署时必须保持一致，生产环境应使用强密钥。 |

## HTTP API

HTTP 响应统一使用以下结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

错误响应会使用 HTTP 状态码作为 `code`，并返回 `message`。

### 注册

```http
POST /register
Content-Type: application/json
```

请求：

```json
{
  "username": "alice",
  "password": "secret"
}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id": 1
  }
}
```

### 登录

```http
POST /login
Content-Type: application/json
```

请求：

```json
{
  "username": "alice",
  "password": "secret"
}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id": 1,
    "username": "alice",
    "token": "<jwt>"
  }
}
```

## WebSocket 协议

使用 JWT 建立连接：

```text
GET /ws?token=<jwt>
```

也可以通过请求头传递：

```text
Authorization: Bearer <jwt>
```

### 发送聊天消息

```json
{
  "type": "chat",
  "to_user_id": 2,
  "content": "hello"
}
```

接收方收到：

```json
{
  "type": "chat",
  "to_user_id": 2,
  "from_user_id": 1,
  "content": "hello"
}
```

如果接收方不在线，消息会写入 PostgreSQL。

### 获取在线用户

```json
{
  "type": "get_online_list"
}
```

响应：

```json
{
  "type": "online_list",
  "data": [1, 2]
}
```

注意：该列表来自当前 chat-server 实例的内存 Hub。

### 获取离线消息

```json
{
  "type": "get_offline"
}
```

响应：

```json
{
  "type": "offline_list",
  "data": [
    {
      "msg_id": 1,
      "to_user_id": 2,
      "from_user_id": 1,
      "content": "hello",
      "created_at": "2026-05-12T10:00:00Z"
    }
  ]
}
```

离线消息在拉取成功后会被删除。

### WebSocket 错误

```json
{
  "type": "error",
  "content": "error message"
}
```

## 数据模型

`init.sql` 会创建：

- `users`：用户账号数据
- `friends`：好友关系预留表
- `offline_messages`：离线消息表

## 测试

测试默认交给 GitHub Actions 执行，不要求在本地运行。

workflow 定义在 `.github/workflows/go-tests.yml`，会在 push、pull request 和手动触发时进入 `chat-server` 模块执行 `go test ./...`。

当前测试覆盖 JWT 签发、解析和非法 token 拒绝。

## 说明

- 这个项目主要用于后端工程学习。
- 当前已经存在好友关系表，但好友相关 HTTP/WebSocket 业务流程尚未完整实现。
- WebSocket upgrader 当前允许所有 Origin，便于本地调试；生产环境需要收紧。
- 默认 JWT 密钥只适合本地开发。
