# Checkut CMS Server

按 `BACKEND.md` 契约实现的 Go 后台管理系统后端。本地 PostgreSQL 为源，发布到 Supabase。

- Go 1.26+，`chi/v5` 路由，`pgx/v5` 连接池
- 纯本地运行，只监听 `127.0.0.1`，无登录
- Supabase 走 postgREST（读取/写入）+ Storage（图片上传）

## 结构

```
cmd/server           入口 + 路由
internal/config      .env 配置加载（godotenv）
internal/model       snake_case 数据模型
internal/repository  pgx 数据访问（含攻略整树对账 ReconcileTree）
internal/service     业务：CRUD / 上传 / 基线导入 / 发布 diff+run
internal/controller  HTTP handler（chi）
internal/supabase    postgREST + Storage 客户端
internal/pkg/response 统一 {data}/{error} 信封
schema.sql           本地建库脚本
```

## 快速开始

1. 建库并执行 schema：

```bash
createdb checkut_cms
psql -d checkut_cms -f schema.sql
```

2. 配置环境：

```bash
cp .env.example .env
# 编辑 .env，填入 CMS_DB_DSN / SUPABASE_URL / SUPABASE_SERVICE_KEY / SUPABASE_STORAGE_BUCKET
```

3. 运行：

```bash
go run ./cmd/server
```

服务默认监听 `http://127.0.0.1:8080`。

## API（统一 `/api/v1`）

- `GET/POST /destinations`，`GET/PUT/PATCH{status}/DELETE /destinations/:id`
- 同上路由集合 for `/attractions`（列表支持 `destination_id`）
- `GET/POST /itineraries`，`GET/PUT/PATCH{status}/DELETE /itineraries/:id`（整树）
- `POST /uploads`（multipart `file` 字段，返回 `{url,filename,size}`）
- `GET /sync/status`，`POST /sync/import`
- `GET /publish/diff`，`POST /publish`

信封：成功 `{"data": <T>}`；错误非 2xx `{"error":{"message","code"}}`。
列表分页 `?page=&page_size=&status=&q=`（景点另支持 `destination_id=`）→ `{data:{items,total,page,page_size}}`。

错误码：`not_found` / `invalid_request` / `db_error` / `upload_error` / `upstream_error` / `conflict`。

## 关键语义

- 软删：`DELETE` 置 `deleted_at=now(), status='archived'`，列表默认排除软删行。
- 攻略整树 PUT：`id==""` 新增、`id!=""` 更新、缺失行软删；`day_number` 按数组序从 1 重排并重算 `total_days`/`activities_count`；被软删的行再次出现时恢复。
- 发布 diff 基准 = `publish_meta.last_synced_at`；攻略树父状态主导（草稿不推、归档删）。
- 导入幂等：以主键 upsert 为 `published`，保留原 `created_at`。

## 测试

```bash
go test ./...
```

- 纯逻辑单测（`ReconcileTree`、发布 diff 判定）无需外部依赖。
- 集成测试由环境变量 `TEST_CMS_DB_DSN` / `TEST_SUPABASE_URL` / `TEST_SUPABASE_SERVICE_KEY` guard，未设置则跳过。

## 安全

Supabase service_role Secret key 只存在于服务端 `.env`，绝不进入前端 bundle。
