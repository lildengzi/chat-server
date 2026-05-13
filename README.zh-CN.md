# Chat Server

[![Go Tests](https://github.com/lildengzi/chat-server/actions/workflows/go-tests.yml/badge.svg)](https://github.com/lildengzi/chat-server/actions/workflows/go-tests.yml)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![WebSocket](https://img.shields.io/badge/WebSocket-enabled-4B5563)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker%20Compose-ready-2496ED?logo=docker&logoColor=white)

一个学生级云服务学习项目，基于 **Go、WebSocket、PostgreSQL、Redis、Docker Compose 和 Nginx** 构建一个小而完整的聊天后端。

这个项目不是生产级 IM 系统，也不是完整聊天产品，当前阶段也暂时不追求复杂的安全加固。它的目标是理解一个服务从本地开发到部署到云服务器的完整过程，同时练习后端和数据库的基础操作：HTTP API、认证、WebSocket 长连接、在线状态、跨实例消息投递、离线消息存储、容器化部署和统一反向代理入口。

当前学习范围：

- 在一台云服务器上搭建一个小型微服务风格系统
- 使用 Docker Compose 编排应用、PostgreSQL、Redis 和 Nginx
- 学习 PostgreSQL 的基础表设计、初始化、插入、查询和离线消息删除
- 使用 Redis 维护在线状态，并通过 Pub/Sub 完成多实例消息转发
- 使用 GitHub Actions 做 CI 测试
- 下一步补上 CD，让推送到 GitHub 的代码可以以可重复的方式部署到云服务器

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

$env:DB_URL="postgres://postgres:chat_server_dev_password@localhost:5432/chatdb?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:INSTANCE_ID="server-1"
$env:JWT_SECRET="chat_server_dev_jwt_secret"

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

## 部署与 CD

当前项目已经可以在云服务器上通过 Docker Compose 手动部署：

```bash
git pull
docker compose up -d --build
```

现在还差的是 CD。对于这个学习项目，先采用最容易理解的方式：

1. 云服务器上保留一份项目仓库。
2. GitHub Actions 先运行测试。
3. 测试通过后，由 GitHub Actions 的部署任务通过 SSH 登录云服务器。
4. 在服务器上执行 `git pull` 和 `docker compose up -d --build`。

也就是说，可以在云服务器上执行 `git pull`，但不建议让服务器长期自动轮询 GitHub 并自己 pull。更清晰的做法是由 CI/CD 触发，或者先手动执行，这样部署时间、日志和失败原因都更容易观察。

后续如果想更接近正式流程，可以改成 GitHub Actions 构建 Docker 镜像，推送到镜像仓库，再让云服务器拉取镜像并重启 Compose。

仓库中已经添加 `.github/workflows/k3s-deploy.yml`，用于简单的 K3s CD 流程。需要在 GitHub 仓库中配置这些 Secrets：

| 名称 | 说明 |
| --- | --- |
| `SERVER_HOST` | 云服务器公网 IP 或域名。 |
| `SERVER_USER` | SSH 用户名。 |
| `SERVER_SSH_KEY` | GitHub Actions 登录服务器使用的 SSH 私钥。 |
| `SERVER_PORT` | 可选 SSH 端口，不配置时默认使用 `22`。 |
| `SUDO_PASSWORD` | 可选 sudo 密码。更推荐给部署用户配置免密 sudo，然后不设置这一项。 |
| `DEPLOY_PATH` | 可选的云服务器部署目录。不配置时，workflow 使用 `$HOME/chat-server`。 |

本地生成部署用 SSH key：

```bash
ssh-keygen -t ed25519 -C "chat-server-deploy" -f ~/.ssh/chat_server_deploy
```

把公钥安装到服务器：

```bash
ssh-copy-id -i ~/.ssh/chat_server_deploy.pub <服务器用户>@<服务器地址>
```

查看私钥内容，并把输出填到 GitHub Secrets 的 `SERVER_SSH_KEY`：

```bash
cat ~/.ssh/chat_server_deploy
```

确认 key 可以登录后，再在服务器上禁用 SSH 密码登录：

```text
PasswordAuthentication no
PermitRootLogin no
PubkeyAuthentication yes
```

然后重启 SSH：

```bash
sudo systemctl restart ssh
```

如果部署用户执行 `sudo` 仍然需要密码，可以先给这个学习服务器配置免密 sudo，或者临时添加 `SUDO_PASSWORD` Secret。

配置免密 sudo：

```bash
sudo visudo -f /etc/sudoers.d/chat-server-deploy
```

写入：

```text
<服务器用户> ALL=(ALL) NOPASSWD:ALL
```

验证：

```bash
sudo -n true
```

如果命令成功，GitHub Actions 部署时就不需要保存服务器密码。

这个 workflow 会先运行 Go 测试。测试通过后，通过 SSH 登录服务器，clone 或更新仓库，然后执行：

```bash
bash scripts/k3s-deploy.sh
```

## K3s 部署

仓库中也提供了一套简单的 K3s 部署方式，用来在单节点云服务器上学习 Kubernetes 的核心概念。

K3s 配置文件位于 `k8s/`：

- `postgres.yaml`：带 PVC 持久化卷的 PostgreSQL
- `redis.yaml`：单实例 Redis
- `chat-server.yaml`：两个 Go 服务副本，并通过 Kubernetes Service 暴露
- `web.yaml`：Nginx 静态服务，用来托管 `index.html`
- `ingress.yaml`：基于 K3s 默认 Traefik 的 Ingress，转发 `/`、`/register`、`/login` 和 `/ws`
- 运行时 Kubernetes Secret：由 `scripts/k3s-deploy.sh` 创建，不提交到仓库

在一台新的云服务器上，clone 仓库后执行：

```bash
bash scripts/k3s-deploy.sh
```

脚本会完成：

1. 如果服务器还没有 K3s，则自动安装 K3s。
2. 如果服务器还没有 Docker，则自动安装 Docker。
3. 使用 Docker 构建本地镜像 `chat-server:local`。
4. 将镜像导入 K3s 使用的 containerd。
5. 根据环境变量或学习默认值创建 Kubernetes Secret。
6. 根据 `init.sql` 和 `index.html` 创建 ConfigMap。
7. 应用 Kubernetes manifests。
8. 等待 PostgreSQL、Redis、chat-server 和 web 服务就绪。

服务器前置条件：

- Linux 云服务器
- `curl`
- `sudo`，如果不是 root 用户运行

如果同一台服务器上已经运行了 Docker Compose 版本，建议先停止它，因为它可能已经占用了 `80` 端口：

```bash
docker compose down
```

部署完成后访问：

```text
http://<云服务器公网 IP>/
```

常用排错命令：

```bash
sudo k3s kubectl -n chat-server get pods
sudo k3s kubectl -n chat-server logs deploy/chat-server
sudo k3s kubectl -n chat-server describe pod <pod-name>
```

如果后续已经把镜像推送到了镜像仓库，可以通过 `IMAGE` 跳过本地构建：

```bash
IMAGE=ghcr.io/<user>/<image>:<tag> bash scripts/k3s-deploy.sh
```

这种模式下，服务器不需要 Docker。

公开仓库中不要提交真实服务器地址、SSH 用户名、密码、kubeconfig 或生产 `.env` 文件。本项目的部署凭据保存在 GitHub Secrets 中，K3s Secret 会在部署时动态创建。

## 说明

- 这个项目主要用于后端工程学习。
- 当前已经存在好友关系表，但好友相关 HTTP/WebSocket 业务流程尚未完整实现。
- WebSocket upgrader 当前允许所有 Origin，便于本地调试；生产环境需要收紧。
- 默认 JWT 密钥只适合本地开发。
