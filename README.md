# 发布准备度与上线门禁协同平台

一个完全离线可运行的发布治理系统。它围绕“清单执行 → 阻塞/豁免 → 多角色会签 → 最终放行 → 本地模拟执行 → 失败恢复”闭环设计，不是通用项目管理后台。

## 模块与目录

- `internal/domain`：发布状态机、阻塞和有期限豁免、会签、模板版本、只读发布快照、执行记录。
- `internal/application`：用例编排及仓储、时钟、ID、自动校验和发布适配器端口。
- `internal/repository`：并发安全的内存实现（测试）和基于 pgx 的 PostgreSQL 实现。
- `internal/transport/http`、`internal/middleware`：`/api/v1`、统一错误、request_id、超时、恢复、CORS 和安全头。
- `internal/platform`、`internal/service`：受控附件目录、本地校验和本地模拟发布。
- `migrations`：可重复执行的结构迁移与不会覆盖用户数据的演示模板。
- `api/openapi`：OpenAPI 3.0 契约。
- `web`：Vue 3 + TypeScript + Vite + Pinia 中文控制台，含发布列表、清单工作台、阻塞/审批、模板、比较与历史快照入口。

## 本地启动

要求 Go 1.24、Node 22、PostgreSQL 17。复制 `.env.example` 为本地 `.env` 后自行加载；不要提交 `.env`。

```sh
docker compose up -d postgres
make migrate
make seed
go run ./cmd/server
cd web && npm ci && npm run dev
```

也可以用 `docker compose up --build` 启动完整环境。健康检查为 `/healthz`，数据库就绪检查为 `/readyz`。

## 演示数据与流程

迁移内置 `standard` 第 1 版生产清单（备份、监控）。使用 `demo-owner`、`quality-user`、`ops-user` 三个演示身份，通过 `X-Actor-ID` 请求头切换。演示流程：创建版本，填报清单和证据，打开阻塞项，提交带未来到期时间、审批人和补救措施的豁免，完成质量与运维会签，评估门禁，固化放行快照，执行本地发布并查看失败恢复指引和审计时间线。

未关闭且没有有效豁免的阻塞项必定令版本进入 `blocked`；豁免到期会重新生效。发布状态为 `preparing → reviewing/blocked → ready → released → rolled_back`。`released` 时固化清单、阻塞、会签与审计头，之后不能直接覆盖。所有写接口使用 `If-Match` 乐观版本；创建接口使用 `Idempotency-Key`。

## API 示例

```sh
curl -X POST http://localhost:8080/api/v1/releases \
  -H 'Content-Type: application/json' -H 'X-Actor-ID: demo-owner' \
  -H 'Idempotency-Key: demo-v1' \
  -d '{"application_id":"pay","environment_id":"prod","version_name":"v1.4.0","owner_id":"demo-owner","risk":"high","template_id":"standard","template_version":1}'
```

列表和工作台支持分页上限、排序白名单和状态/负责人/风险/时间筛选。错误响应统一包含稳定 `code`、中文可读 `message`、`field_errors` 与 `request_id`。日志只记录动作与 request_id，不记录令牌、附件正文或其他敏感值。

## 验证

```sh
go build ./...
go test ./...
go test -race ./...
go vet ./...
cd web && npm ci && npm test && npm run build
```

基线验证结果（2026-08-17）：上述 Go build/test/race/vet 与前端测试、生产构建均应通过；Docker Compose 使用本地 PostgreSQL、文件卷和模拟适配器，不调用外部业务接口、CDN、云存储、在线模型或真实消息服务。
