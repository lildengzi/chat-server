#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NAMESPACE="${NAMESPACE:-chat-server}"
IMAGE="${IMAGE:-chat-server:local}"
INSTALL_K3S="${INSTALL_K3S:-true}"
INSTALL_DOCKER="${INSTALL_DOCKER:-true}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-chat_server_dev_password}"
JWT_SECRET="${JWT_SECRET:-chat_server_dev_jwt_secret}"
DB_URL="${DB_URL:-postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/chatdb?sslmode=disable}"

cd "$ROOT_DIR"

log() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

need_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

if ! command -v k3s >/dev/null 2>&1; then
  if [ "$INSTALL_K3S" != "true" ]; then
    echo "k3s is not installed. Set INSTALL_K3S=true or install k3s manually." >&2
    exit 1
  fi

  need_command curl
  log "Installing K3s"
  if [ "$(id -u)" -eq 0 ]; then
    curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644
  else
    need_command sudo
    curl -sfL https://get.k3s.io | sudo sh -s - --write-kubeconfig-mode 644
  fi
fi

if [ "$(id -u)" -eq 0 ]; then
  KUBECTL=(k3s kubectl)
  CTR=(k3s ctr --namespace k8s.io)
  DOCKER=(docker)
else
  need_command sudo
  KUBECTL=(sudo k3s kubectl)
  CTR=(sudo k3s ctr --namespace k8s.io)
  DOCKER=(sudo docker)
fi

log "Waiting for K3s node"
"${KUBECTL[@]}" wait --for=condition=Ready node --all --timeout=180s

if [ "$IMAGE" = "chat-server:local" ]; then
  if ! command -v docker >/dev/null 2>&1; then
    if [ "$INSTALL_DOCKER" != "true" ]; then
      echo "docker is not installed. Set INSTALL_DOCKER=true, install docker manually, or pass IMAGE=<registry-image>." >&2
      exit 1
    fi

    need_command curl
    log "Installing Docker for local image build"
    if [ "$(id -u)" -eq 0 ]; then
      curl -fsSL https://get.docker.com | sh
    else
      curl -fsSL https://get.docker.com | sudo sh
    fi
  fi

  log "Building local image: $IMAGE"
  "${DOCKER[@]}" build -t "$IMAGE" ./chat-server

  IMAGE_TAR="$(mktemp -t chat-server-image.XXXXXX.tar)"
  log "Importing image into K3s containerd"
  "${DOCKER[@]}" save -o "$IMAGE_TAR" "$IMAGE"
  "${CTR[@]}" images import "$IMAGE_TAR"
  rm -f "$IMAGE_TAR"
else
  log "Using external image: $IMAGE"
fi

log "Applying namespace and static manifests"
"${KUBECTL[@]}" apply -f k8s/namespace.yaml

log "Creating runtime Secret and ConfigMaps"
"${KUBECTL[@]}" -n "$NAMESPACE" create secret generic chat-secrets \
  --from-literal=postgres-password="$POSTGRES_PASSWORD" \
  --from-literal=db-url="$DB_URL" \
  --from-literal=jwt-secret="$JWT_SECRET" \
  --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -

"${KUBECTL[@]}" -n "$NAMESPACE" create configmap chat-postgres-initdb \
  --from-file=init.sql=init.sql \
  --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -

"${KUBECTL[@]}" -n "$NAMESPACE" create configmap chat-web-index \
  --from-file=index.html=index.html \
  --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -

"${KUBECTL[@]}" apply -f k8s/postgres.yaml
"${KUBECTL[@]}" apply -f k8s/redis.yaml
"${KUBECTL[@]}" apply -f k8s/chat-server.yaml
"${KUBECTL[@]}" apply -f k8s/web.yaml
"${KUBECTL[@]}" apply -f k8s/ingress.yaml

if [ "$IMAGE" != "chat-server:local" ]; then
  "${KUBECTL[@]}" -n "$NAMESPACE" set image deployment/chat-server chat-server="$IMAGE"
fi

log "Waiting for deployments"
"${KUBECTL[@]}" -n "$NAMESPACE" rollout status deployment/postgres --timeout=180s
"${KUBECTL[@]}" -n "$NAMESPACE" rollout status deployment/redis --timeout=180s
"${KUBECTL[@]}" -n "$NAMESPACE" rollout status deployment/chat-server --timeout=180s
"${KUBECTL[@]}" -n "$NAMESPACE" rollout status deployment/chat-web --timeout=180s

log "Deployment status"
"${KUBECTL[@]}" -n "$NAMESPACE" get pods,svc,ingress

cat <<'EOF'

K3s deployment finished.

Open the cloud server public IP in your browser:
  http://<server-public-ip>/

Useful commands:
  sudo k3s kubectl -n chat-server get pods
  sudo k3s kubectl -n chat-server logs deploy/chat-server
  sudo k3s kubectl -n chat-server describe pod <pod-name>

EOF
