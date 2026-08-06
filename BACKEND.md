# Checkut CMS — 后端开发清单（Go）

> 前端已实现（本仓库 `src/`），按本清单的 REST 契约对接。**后端未实现**，本文件是给开发者的完整实施清单。
> 请按契约实现，字段名一律 **snake_case**（与数据库列一致，前端类型一一对应，无需 DTO 转换层）。

---

## 1. 架构

```
┌─ 本地 PostgreSQL（CMS 主库）──────────────────┐
│ destinations / attractions / itineraries      │   ← 所有编辑发生在这里
│   + itinerary_days + itinerary_activities      │     额外列: status, deleted_at, updated_at
└───────────────┬───────────────────────────────┘
                │ ① CRUD (REST /api/v1/*)
                │ ② 图片上传 → Backblaze B2 → Cloudflare CDN URL
                │ ③ 基线导入（首次从 Supabase 拉取）
                │ ④ 发布（diff 预览 + 一键同步到 Supabase）
                ▼
┌─ Supabase（线上 App 只读，RLS 公共读）─────────┐
│ 同构表结构（无状态列），App 直接消费            │
└───────────────────────────────────────────────┘
```

- 数据流**严格单向**：本地为源，发布到 Supabase；基线导入后不再反向。
- 纯本地运行，绑 `127.0.0.1`，无登录。
- `destinations` / `itineraries` 的 `id` 在 App 端是 `varchar`：**新行由后端生成 uuid 字符串**；存量非 uuid id 原样保留。本地 id 列统一用 `text` 兼容两者。

## 2. 技术选型建议

| 用途 | 建议 | 说明 |
|---|---|---|
| HTTP | 标准库 `net/http` 或 `chi` | 单机低并发，无需 gin 重量框架 |
| 本地 PG | `jackc/pgx/v5` (pgxpool) | |
| Supabase 写入 | 直连 `pgx` 到 Supabase PG（DSN） | 用 service_role；或 postgREST REST API，二选一。直连 upsert 更高效 |
| B2 上传 | `aws-sdk-go-v2` 的 `s3` 客户端 | B2 兼容 S3 API，endpoint 为 `https://s3.{region}.backblazeb2.com` |
| 配置 | 环境变量 + `.env` 加载 | 见 §3 |

## 3. 配置项（.env）

```dotenv
# 服务监听（仅本机）
HTTP_ADDR=127.0.0.1:8080

# 本地 CMS 主库
CMS_DB_DSN=postgres://cms:cms@127.0.0.1:5432/checkut_cms?sslmode=disable

# Supabase（用于基线导入 + 发布目标）
SUPABASE_DB_DSN=postgresql://postgres.<ref>:<service_role>@aws-0-<region>.pooler.supabase.com:5432/postgres
# 或改用 REST：
SUPABASE_URL=https://<ref>.supabase.co
SUPABASE_SERVICE_KEY=eyJ...

# Backblaze B2（S3 兼容）
B2_ENDPOINT=https://s3.us-west-004.backblazeb2.com
B2_BUCKET=checkut-cms
B2_KEY_ID=...
B2_APP_KEY=...

# Cloudflare CDN 前缀（B2 的公开域名 / 自定义 CDN 域名）
CDN_BASE_URL=https://cdn.example.com
```

**安全铁律**：B2/Supabase 密钥只存在于 Go 服务端，**绝不进入前端 bundle**。

## 4. 本地数据库 Schema（`schema.sql`）

本地库 = App 表结构镜像 + 状态列。建库后执行。

```sql
create extension if not exists pgcrypto;

create table destinations (
  id            text primary key default (gen_random_uuid()::text),
  title         varchar not null,
  country       varchar,
  continent     varchar,
  rating        numeric,
  reviews_count int4,
  image_url     varchar,
  description   text,
  tags          jsonb default '[]'::jsonb,
  best_time     varchar,
  duration      varchar,
  -- CMS 状态列
  status        text not null default 'draft' check (status in ('draft','published','archived')),
  created_at    timestamptz default now(),
  updated_at    timestamptz default now(),
  deleted_at    timestamptz
);

create table attractions (
  id             text primary key default (gen_random_uuid()::text),
  destination_id text not null references destinations(id) on delete cascade,
  title          varchar not null,
  subtitle       varchar,
  image_url      varchar,
  duration       varchar,
  tag            varchar,
  status         text not null default 'draft' check (status in ('draft','published','archived')),
  created_at     timestamptz default now(),
  updated_at     timestamptz default now(),
  deleted_at     timestamptz
);

create table itineraries (
  id               text primary key default (gen_random_uuid()::text),
  user_id          uuid,            -- CMS 维护内容为 NULL
  destination_id   text not null references destinations(id) on delete cascade,
  title            varchar not null,
  total_days       varchar,
  cities_count     varchar,
  activities_count varchar,
  status           text not null default 'draft' check (status in ('draft','published','archived')),
  created_at       timestamptz default now(),
  updated_at       timestamptz default now(),
  deleted_at       timestamptz
);

create table itinerary_days (
  id           text primary key default (gen_random_uuid()::text),
  itinerary_id text not null references itineraries(id) on delete cascade,
  day_number   int4 not null,
  title        varchar,
  subtitle     varchar,
  image_url    varchar,
  status       text not null default 'draft' check (status in ('draft','published','archived')),
  created_at   timestamptz default now(),
  updated_at   timestamptz default now(),
  deleted_at   timestamptz,
  unique (itinerary_id, day_number)
);

create table itinerary_activities (
  id            text primary key default (gen_random_uuid()::text),
  day_id        text not null references itinerary_days(id) on delete cascade,
  attraction_id text references attractions(id) on delete set null,
  time          varchar,
  title         varchar not null,
  location      varchar,
  description   text,
  tip           text,
  status        text not null default 'draft' check (status in ('draft','published','archived')),
  created_at    timestamptz default now(),
  updated_at    timestamptz default now(),
  deleted_at    timestamptz
);

-- 发布元信息
create table publish_meta (
  id             int primary key default 1 check (id = 1),
  last_synced_at timestamptz
);
insert into publish_meta (id, last_synced_at) values (1, null);

-- 触发 updated_at 自动更新（可选：更可靠的做法是服务端写 updated_at）
create or replace function set_updated_at() returns trigger as $$
begin
  new.updated_at = now();
  return new;
end $$ language plpgsql;
-- 对上述 5 张内容表各建触发器：before update on <t> for each row execute function set_updated_at();
```

> 说明：`itinerary_days` / `itinerary_activities` 也带 `status` 列，但**发布语义由父攻略的 `status` 主导**（见 §8），子行 status 仅作兜底。前端编辑时不暴露子行 status。

## 5. REST API 契约

统一信封：成功 `{ "data": <T> }`；错误非 2xx：`{ "error": { "message": "...", "code": "..." } }`。
列表分页：`?page=1&page_size=20&status=&q=&destination_id=` → `{ "data": { "items": [...], "total": n, "page": p, "page_size": s } }`。
列表默认**排除软删行**（`deleted_at IS NULL`）。`q` 按 `title`（景点可含 `subtitle`）模糊匹配。

### 5.1 destinations / attractions（结构相同，`<res>` = `destinations` | `attractions`）

| 方法 | 路径 | 请求 | 响应 `data` |
|---|---|---|---|
| GET | `/api/v1/<res>` | 分页 + `status` + `q`（景点另支持 `destination_id`） | `Page<T>` |
| GET | `/api/v1/<res>/:id` | — | 单行 `T` |
| POST | `/api/v1/<res>` | 草稿对象（不含 id/审计列，`status` 可选默认 `draft`） | 创建后的 `T`（后端生成 id） |
| PUT | `/api/v1/<res>/:id` | 完整草稿对象 | 更新后的 `T` |
| PATCH | `/api/v1/<res>/:id/status` | `{ "status": "draft"\|"published"\|"archived" }` | 更新后的 `T` |
| DELETE | `/api/v1/<res>/:id` | — | `204` / `{data:null}`，**软删**：置 `deleted_at=now(), status='archived'` |

字段：`Destination` / `Attraction` 见 `src/api/types.ts`（含 `status`、`created_at`、`updated_at`、`deleted_at`）。

### 5.2 itineraries（嵌套树）

| 方法 | 路径 | 请求 | 响应 `data` |
|---|---|---|---|
| GET | `/api/v1/itineraries` | 分页 + `status` + `q` | `Page<Itinerary>`（扁平，不含树） |
| GET | `/api/v1/itineraries/:id` | — | `ItineraryWithTree`（含 `days[].activities[]`） |
| POST | `/api/v1/itineraries` | 整树草稿 | 创建的整树（后端回填所有 id） |
| PUT | `/api/v1/itineraries/:id` | 整树 | **整树替换**：更新/插入/删除 days 与 activities 按 id 对账 |
| PATCH | `/api/v1/itineraries/:id/status` | `{ "status": ... }` | `Itinerary` |
| DELETE | `/api/v1/itineraries/:id` | — | 软删 + 级联置子行 `deleted_at` |

**整树对账规则**（PUT）：
- `days` 中 `id != ''` → 已存在，更新；`id == ''` → 新增，后端生成 uuid 并回填。
- 当前行不在请求 `days` 中且存在 → **软删**（置 `deleted_at`）。
- `activities` 同理，挂在所属 `day` 下。
- 校验 `day_number` 由服务端按数组序重排，并据此重算 `itineraries.total_days` / `activities_count`。

### 5.3 图片上传

| 方法 | 路径 | 请求 | 响应 `data` |
|---|---|---|---|
| POST | `/api/v1/uploads` | `multipart/form-data`，字段名 `file` | `{ "url": string, "filename": string, "size": int }` |

实现：校验 MIME（`image/*`，拒绝伪装）+ 大小上限（建议 10MB）→ 生成随机 key（如 `images/{uuid}.{ext}`）→ 上传 B2 bucket → 返回 `CDN_BASE_URL + "/" + key`。前端只在拿到 `url` 后落库。

### 5.4 同步与发布

| 方法 | 路径 | 响应 `data` |
|---|---|---|
| GET | `/api/v1/sync/status` | `SyncStatus`：`{ configured, last_synced_at, local_counts, message? }` |
| POST | `/api/v1/sync/import` | `ImportResult`：`{ imported: {<table>: n}, errors: [] }` |
| GET | `/api/v1/publish/diff` | `PublishDiff`（见 §8） |
| POST | `/api/v1/publish` | `PublishResult`：`{ ok, applied: {<table>: n}, errors: [] }` |

## 6. 基线导入（`POST /sync/import`）

1. 从 Supabase 读五张表全量（`destinations, attractions, itineraries, itinerary_days, itinerary_activities`）。
2. 以主键为基准 upsert 进本地，`status='published'`，保留原 `created_at`，置 `deleted_at=null`。
3. 刷新 `publish_meta.last_synced_at = now()`。
4. 单表失败记入 `errors`，不整体回滚（可重复执行，幂等）。
> 前置：本地上游表（如 destinations）必须已在，否则子表 FK 失败——按依赖序导入。

## 7. 发布引擎

### diff（`GET /publish/diff`）

基准 = `publish_meta.last_synced_at`（首次为 null → 全部视为 created）。逐表判定：

- **created**：`created_at > last_synced_at` 且 `deleted_at is null`。
- **updated**：`updated_at > last_synced_at` 且 `deleted_at is null`。
- **deleted**：`deleted_at is not null`（或 `status='archived'`）且此前已同步过（即该 id 出现在 Supabase，或 `last_synced_at` 之后本地有该行）。

**攻略树规则**（父主导）：
- 攻略 `status != 'published'`（草稿不推、归档删）：
  - 若该攻略 id 已存在于 Supabase → 标记为 deleted（含其全部 days/activities）。
  - 若从未发布 → 忽略（不产生 diff）。
- 攻略 `published`：其 days/activities 按各自 diff 规则 upsert；子行软删 → deleted。
- 草稿攻略下的子行不产生任何 upsert。

响应按表分组，`created/updated/deleted` 各为 `ChangeItem[]`（`{ id, label, change }`，`label` 取 title 供展示）。

### run（`POST /publish`）

按依赖序执行，全部在单事务/尽力逐条，收集错误：

1. `destinations`：upsert published（排除软删/归档）；归档/软删 → **delete**。
   - ⚠️ Supabase 侧若存在 `user_favorites` / `itineraries` 外键，删除可能失败 → 记入 `errors`，跳过该行（前端会展示错误）。
2. `attractions`：upsert published；软删 → delete。仅当所属 destination 存在于 Supabase。
3. `itineraries`：upsert published；归档/软删 → 先删其 days/activities 再删本行。
4. `itinerary_days`、`itinerary_activities`：仅 upsert 属于 published 攻略且非软删的行；软删/父归档 → delete。

成功后 `publish_meta.last_synced_at = now()`。返回 `applied` 计数与 `errors`。**可重复执行**，失败重跑基于新的 last_synced_at。

## 8. 验收清单

- [ ] `schema.sql` 建库成功，五张表 + `publish_meta` + 触发器
- [ ] 5.1 CRUD：增删改查、软删、status 切换均生效
- [ ] 5.2 攻略整树 PUT 对账正确（新增/更新/删除 days 与 activities）
- [ ] 上传真实图片 → 返回 CDN URL 可公网访问
- [ ] 基线导入：本地出现 Supabase 现有数据且为 published
- [ ] 发布一个草稿攻略不产生任何线上变更
- [ ] 发布 published 攻略 → 线上出现该攻略及天数/活动
- [ ] 归档已发布攻略 → 发布后线上该攻略与其子行被删除
- [ ] `pnpm dev`（前端）+ Go 服务联调，发布页 diff 与实际一致
- [ ] 服务只监听 127.0.0.1，密钥未泄露

## 9. 常见坑

- **id 类型**：本地 id 统一 `text`，勿用 `uuid` 类型（存量非 uuid id 会写入失败）。
- **时区**：`last_synced_at` / `updated_at` 统一 timestamptz，比较用 UTC。
- **删除语义**：本地 `DELETE` 与 `archived` 都经发布在 Supabase 侧物理删除；`user_favorites` 引用 destination 时删除会失败，按 §7 处理。
- **前端契约**：任何字段名/响应结构改动需同步 `src/api/types.ts` 与 `src/api/index.ts`。
