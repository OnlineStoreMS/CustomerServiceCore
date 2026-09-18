# CustomerServiceCore

OSMS 客服中心（抖店客服消息监控、云端回复、自动回复）。

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
- `POST /api/v1/plugin/outbound/claim` — claim pending cloud/auto/llm replies for Feige send
- `POST /api/v1/plugin/outbound/:id/ack` — body `{ "ok": true }`

Heartbeat returns `{ monitorEnabled, shopName, platform, platformShopId, pendingOutbound }`.

## Admin API

JWT auth (same secret as UserCore). Routes under `/api/v1/admin`:

- Shops: list/create/get/patch, rotate-bind-code, reset-plugin
- Conversations: list, messages, **reply** (queues outbound for WindowsAgent)
- Auto-reply rules: list/create/patch/delete, presets (寒暄模板)
- LLM settings: `GET/PATCH /llm-settings`（DeepSeek 简短问答，关键词未命中才调用）

## DeepSeek 自动回复

配置 `llm.api_key` 或环境变量 `LLM_API_KEY` / `DEEPSEEK_API_KEY`。页面「自动回复」里打开开关。

链路：买家进线 → 先走关键词规则 → 未命中再异步调 DeepSeek → 结合最近对话直接答（规格对比不再连说稍等）→ 压成 ≤80 字口语 → 排队给出站 → WindowsAgent 发到飞鸽。

模型被要求用真人短句，禁止分点、Markdown、「您好 / 希望对您有帮助」等 AI 腔。本店单号、库存、券后价、到货时间不编；商品型号对比要用对话上下文直接答。

## Docker

```bash
docker build -f docker/Dockerfile.api -t customerservicecore-api .
docker build -f docker/Dockerfile.web --build-arg VITE_BASE=/apps/customer-service/ -t customerservicecore-web .
```

推送到 `main` / `dev_yeyazhou` 或手动触发 Actions：`.github/workflows/docker-push-acr.yml` 会构建并推送阿里云 ACR 镜像 `customerservicecore-api` / `customerservicecore-web`（需仓库 Secrets：`ALIYUN_ACR_*`）。
