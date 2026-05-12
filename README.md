# Chat Server

A personal learning project for building a small but complete chat backend with **Go, WebSocket, PostgreSQL, Redis, Docker Compose, and Nginx**.

This is not positioned as a production-ready IM system. The goal is to practice a realistic backend workflow: HTTP APIs, authentication, WebSocket connections, online state, cross-instance message delivery, offline message storage, containerized deployment, and a unified reverse-proxy entry point.

Chinese version: [README.zh-CN.md](README.zh-CN.md)

## Features

- User registration and login
- bcrypt password hashing
- JWT-based login sessions and WebSocket authentication
- WebSocket long-lived connections
- One-to-one chat messages
- Online user list for the current server instance
- Cross-instance message delivery through Redis Pub/Sub
- Offline message storage and retrieval through PostgreSQL
- WebSocket ping/pong heartbeat and read/write deadlines
- Single active connection policy per user, including cross-instance kick-out
- Unified JSON responses for HTTP APIs
- Basic request and WebSocket message validation

## Tech Stack

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

## Architecture

```text
Browser / Client
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

Nginx exposes a single HTTP entry point on port `80` and forwards HTTP/WebSocket traffic to two chat-server instances.

Each chat-server instance manages only its local WebSocket clients. Redis stores online user state and broadcasts messages between instances. PostgreSQL stores users and offline messages.

## Project Structure

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

## Quick Start

### Prerequisites

- Docker
- Docker Compose

### Run with Docker Compose

```bash
docker compose up --build
```

After the services start:

- Web UI: `http://localhost/`
- HTTP API entry point: `http://localhost/`
- WebSocket entry point: `ws://localhost/ws?token=<jwt>`

Compose starts:

- `postgres`
- `redis`
- `chat-server-1`
- `chat-server-2`
- `nginx`

### Stop

```bash
docker compose down
```

To remove the PostgreSQL volume as well:

```bash
docker compose down -v
```

## Local Development

You can run the Go service directly from the `chat-server` module if PostgreSQL and Redis are available:

```bash
cd chat-server

$env:DB_URL="postgres://postgres:123456@localhost:5432/chatdb?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:INSTANCE_ID="server-1"
$env:JWT_SECRET="chat-server-local-dev-secret"

go run .
```

On macOS/Linux, use `export` instead of `$env:`.

The service listens on `:8080`.

## Configuration

| Environment variable | Default | Description |
| --- | --- | --- |
| `DB_URL` | empty | PostgreSQL connection string. Required for startup. |
| `REDIS_ADDR` | `localhost:6379` | Redis address. |
| `INSTANCE_ID` | `server-1` | Logical server instance ID used in Redis online state. |
| `JWT_SECRET` | `dev-secret-change-me` | JWT signing secret. Use a strong shared secret for all instances. |

## HTTP API

All HTTP responses use the same envelope:

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

Error responses use the HTTP status code as `code` and include a `message`.

### Register

```http
POST /register
Content-Type: application/json
```

Request:

```json
{
  "username": "alice",
  "password": "secret"
}
```

Response:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id": 1
  }
}
```

### Login

```http
POST /login
Content-Type: application/json
```

Request:

```json
{
  "username": "alice",
  "password": "secret"
}
```

Response:

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

## WebSocket Protocol

Connect with a JWT:

```text
GET /ws?token=<jwt>
```

Alternatively, pass the token in the request header:

```text
Authorization: Bearer <jwt>
```

### Send a Chat Message

```json
{
  "type": "chat",
  "to_user_id": 2,
  "content": "hello"
}
```

The receiver gets:

```json
{
  "type": "chat",
  "to_user_id": 2,
  "from_user_id": 1,
  "content": "hello"
}
```

If the receiver is offline, the message is saved in PostgreSQL.

### Get Online Users

```json
{
  "type": "get_online_list"
}
```

Response:

```json
{
  "type": "online_list",
  "data": [1, 2]
}
```

Note: this list is returned from the current chat-server instance's in-memory hub.

### Get Offline Messages

```json
{
  "type": "get_offline"
}
```

Response:

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

Offline messages are deleted after they are fetched.

### WebSocket Error

```json
{
  "type": "error",
  "content": "error message"
}
```

## Data Model

`init.sql` creates:

- `users`: user account data
- `friends`: reserved friend relationship table
- `offline_messages`: messages waiting for offline users

## Tests

Tests are configured to run in GitHub Actions instead of being run locally by default.

The workflow is defined in `.github/workflows/go-tests.yml` and runs `go test ./...` from the `chat-server` module on pushes, pull requests, and manual workflow dispatches.

Current tests cover basic JWT generation, parsing, and invalid token rejection.

## Notes

- This project is designed for backend engineering practice.
- The current friend table is present, but friend-related HTTP/WebSocket business flows are not fully implemented.
- The WebSocket upgrader currently allows all origins, which is convenient for local testing but should be restricted before production use.
- The default JWT secret is only suitable for local development.
