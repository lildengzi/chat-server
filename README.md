# Chat Server

[![Go Tests](https://github.com/lildengzi/chat-server/actions/workflows/go-tests.yml/badge.svg)](https://github.com/lildengzi/chat-server/actions/workflows/go-tests.yml)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![WebSocket](https://img.shields.io/badge/WebSocket-enabled-4B5563)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker%20Compose-ready-2496ED?logo=docker&logoColor=white)

A student-level cloud service learning project for building a small but complete chat backend with **Go, WebSocket, PostgreSQL, Redis, Docker Compose, and Nginx**.

This is not positioned as a production-ready IM system, and it intentionally does not focus on complex security hardening yet. The goal is to understand the full path from local development to deployment on a cloud server, while practicing common backend and database operations: HTTP APIs, authentication, WebSocket connections, online state, cross-instance message delivery, offline message storage, containerized deployment, and a unified reverse-proxy entry point.

Current learning scope:

- Build and run a small microservice-style system on one cloud server
- Use Docker Compose to orchestrate the application, PostgreSQL, Redis, and Nginx
- Learn basic PostgreSQL table design, initialization, inserts, queries, and deletion of offline messages
- Use Redis for online state and Pub/Sub between service instances
- Use GitHub Actions for CI testing
- Add CD next, so code pushed to GitHub can be deployed to the cloud server in a repeatable way

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

$env:DB_URL="postgres://postgres:chat_server_dev_password@localhost:5432/chatdb?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:INSTANCE_ID="server-1"
$env:JWT_SECRET="chat_server_dev_jwt_secret"

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

## Deployment and CD

The project can already be deployed manually on a cloud server with Docker Compose:

```bash
git pull
docker compose up -d --build
```

CD is the next missing step. For this learning project, the recommended simple path is:

1. Keep the project repository on the cloud server.
2. Let GitHub Actions run tests first.
3. After tests pass, use a GitHub Actions deployment job to SSH into the cloud server.
4. Run `git pull` and `docker compose up -d --build` on the server.

This is easier to understand than introducing an image registry immediately. Avoid making the cloud server continuously poll GitHub and pull automatically. A server-side pull is acceptable as the deployment command, but it should be triggered by CI/CD, or run manually, so deployment timing and logs are visible.

For a more standard later version, build a Docker image in GitHub Actions, push it to a registry, and let the server pull the image and restart Compose.

This repository includes `.github/workflows/k3s-deploy.yml` for the simple K3s CD path. Configure these GitHub repository secrets:

| Name | Description |
| --- | --- |
| `SERVER_HOST` | Cloud server public IP or domain. |
| `SERVER_USER` | SSH username. |
| `SERVER_SSH_KEY` | Private SSH key used by GitHub Actions to log in to the server. |
| `SERVER_PORT` | Optional SSH port. Defaults to `22` if omitted. |
| `SUDO_PASSWORD` | Optional sudo password. Prefer passwordless sudo for the deploy user and leave this unset. |
| `DEPLOY_PATH` | Optional deployment directory on the cloud server. If omitted, the workflow uses `$HOME/chat-server`. |

Generate a deployment key locally:

```bash
ssh-keygen -t ed25519 -C "chat-server-deploy" -f ~/.ssh/chat_server_deploy
```

Install the public key on the server:

```bash
ssh-copy-id -i ~/.ssh/chat_server_deploy.pub <server-user>@<server-host>
```

Add the private key content to GitHub Secrets:

```bash
cat ~/.ssh/chat_server_deploy
```

Use the output as the value of `SERVER_SSH_KEY`.

After key login works, disable SSH password login on the server:

```text
PasswordAuthentication no
PermitRootLogin no
PubkeyAuthentication yes
```

Then restart SSH:

```bash
sudo systemctl restart ssh
```

If the deploy user still needs a password for `sudo`, either configure passwordless sudo for the learning server or add `SUDO_PASSWORD` as a temporary GitHub Secret.

For passwordless sudo, create a sudoers file on the server:

```bash
sudo visudo -f /etc/sudoers.d/chat-server-deploy
```

Add:

```text
<server-user> ALL=(ALL) NOPASSWD:ALL
```

Then verify:

```bash
sudo -n true
```

If it succeeds, GitHub Actions can deploy without storing a server password.

The workflow runs Go tests first. If tests pass, it SSHs into the server, clones or updates the repository, and runs:

```bash
bash scripts/k3s-deploy.sh
```

## K3s Deployment

This repository also includes a simple K3s deployment path for learning Kubernetes concepts with a lighter single-node cluster.

The K3s manifests are in `k8s/`:

- `postgres.yaml`: PostgreSQL with a PVC
- `redis.yaml`: Redis single instance
- `chat-server.yaml`: two Go service replicas behind a Kubernetes Service
- `web.yaml`: Nginx serving `index.html`
- `ingress.yaml`: K3s Traefik Ingress for `/`, `/register`, `/login`, and `/ws`
- Runtime Kubernetes Secret: generated by `scripts/k3s-deploy.sh`, not committed to the repository

On a fresh cloud server, clone the repository and run:

```bash
bash scripts/k3s-deploy.sh
```

The script will:

1. Install K3s if it is not already installed.
2. Install Docker if it is not already installed.
3. Build the local `chat-server:local` image with Docker.
4. Import the image into K3s containerd.
5. Create a Kubernetes Secret from environment variables or learning defaults.
6. Create ConfigMaps from `init.sql` and `index.html`.
7. Apply the Kubernetes manifests.
8. Wait for PostgreSQL, Redis, chat-server, and web deployments to become ready.

Prerequisites on the server:

- Linux cloud server
- `curl`
- `sudo`, unless running as root

If the Docker Compose version is already running on the same server, stop it first because it may already be using port `80`:

```bash
docker compose down
```

After deployment, open:

```text
http://<server-public-ip>/
```

Useful commands:

```bash
sudo k3s kubectl -n chat-server get pods
sudo k3s kubectl -n chat-server logs deploy/chat-server
sudo k3s kubectl -n chat-server describe pod <pod-name>
```

If you already have an image in a registry, skip the local build by passing `IMAGE`:

```bash
IMAGE=ghcr.io/<user>/<image>:<tag> bash scripts/k3s-deploy.sh
```

In that mode, Docker is not required on the server.

For public repositories, do not commit real server addresses, SSH usernames, passwords, kubeconfig files, or production `.env` files. This project keeps deployment credentials in GitHub Secrets and generates the K3s Secret at deploy time.

## Notes

- This project is designed for backend engineering practice.
- The current friend table is present, but friend-related HTTP/WebSocket business flows are not fully implemented.
- The WebSocket upgrader currently allows all origins, which is convenient for local testing but should be restricted before production use.
- The default JWT secret is only suitable for local development.
