-- Checkut CMS — 本地数据库 Schema
-- 建库后执行：psql -f schema.sql <dbname>
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
  images         jsonb default '[]'::jsonb,
  video_url      varchar,
  latitude       numeric,
  longitude      numeric,
  duration       varchar,
  tag            varchar,
  source_url     varchar,
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
  route_line   jsonb default '[]'::jsonb,
  status       text not null default 'draft' check (status in ('draft','published','archived')),
  created_at   timestamptz default now(),
  updated_at   timestamptz default now(),
  deleted_at   timestamptz
);

-- day_number is server-recomputed from array order each PUT; uniqueness must only
-- hold among active rows so soft-deleted days don't block renumbering.
create unique index itinerary_days_active_uk on itinerary_days (itinerary_id, day_number) where deleted_at is null;

create table itinerary_activities (
  id            text primary key default (gen_random_uuid()::text),
  day_id        text not null references itinerary_days(id) on delete cascade,
  attraction_id text references attractions(id) on delete set null,
  time          varchar,
  title         varchar not null,
  location      varchar,
  description   text,
  tip           text,
  images        jsonb default '[]'::jsonb,
  video_url     varchar,
  latitude      numeric,
  longitude     numeric,
  poi_info      jsonb default '{}'::jsonb,
  source_url    varchar,
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

-- 触发 updated_at 自动更新
create or replace function set_updated_at() returns trigger as $$
begin
  new.updated_at = now();
  return new;
end $$ language plpgsql;

create trigger trg_destinations_updated_at before update on destinations for each row execute function set_updated_at();
create trigger trg_attractions_updated_at before update on attractions for each row execute function set_updated_at();
create trigger trg_itineraries_updated_at before update on itineraries for each row execute function set_updated_at();
create trigger trg_itinerary_days_updated_at before update on itinerary_days for each row execute function set_updated_at();
create trigger trg_itinerary_activities_updated_at before update on itinerary_activities for each row execute function set_updated_at();
