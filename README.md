# CustomerServiceCore

OSMS 客服中心（phase 1：抖店客服消息监控）。

## Ports

| Service | Port / Path |
|---------|-------------|
| API | `8108` |
| Web (Vite) | `5193` |
| Caddy / nginx base | `/apps/customer-service/` |

## Local run

```bash
# API
go run ./cmd/api -config configs/config.yaml

# Web
cd web && npm install && npm run dev
```

## Plugin API

- `POST /api/v1/plugin/bind` — body `{ "bindCode": "..." }`
- `POST /api/v1/plugin/heartbeat` — headers `X-Plugin-Key` / `X-Plugin-Secret`
- `POST /api/v1/plugin/messages` — batch upsert messages (idempotent on `platformMessageId`)

Heartbeat returns `{ monitorEnabled, shopName, platform, platformShopId }` so WindowsAgent knows whether to keep listening.

## Admin API

JWT auth (same secret as UserCore). Routes under `/api/v1/admin`:

- Shops: list/create/get/patch, rotate-bind-code, reset-plugin
- Conversations: list + messages

## Docker

```bash
docker build -f docker/Dockerfile.api -t customerservicecore-api .
docker build -f docker/Dockerfile.web --build-arg VITE_BASE=/apps/customer-service/ -t customerservicecore-web .
```

推送到 `main` / `dev_yeyazhou` 或手动触发 Actions：`.github/workflows/docker-push-acr.yml` 会构建并推送阿里云 ACR 镜像 `customerservicecore-api` / `customerservicecore-web`（需仓库 Secrets：`ALIYUN_ACR_*`）。
