-- Create "sessions" table: one row per recorded sport session, owned by a user.
create table "public"."sessions" (
  "id"            uuid             not null default gen_random_uuid(),
  "user_id"       uuid             not null,
  "started_at"    timestamptz      not null,
  "ended_at"      timestamptz      not null,
  "duration_s"    integer          not null,
  "distance_m"    double precision not null default 0,
  "hr_avg"        integer          null,
  "hr_max"        integer          null,
  "speed_max_kmh" double precision null,
  "sprints"       integer          not null default 0,
  "intensity"     double precision null,
  "calories_kcal" double precision null,
  "source"        text             not null,
  "created_at"    timestamptz      not null default now(),
  primary key ("id"),
  constraint "sessions_user_id_fkey" foreign key ("user_id") references "public"."profiles" ("id") on delete cascade,
  constraint "sessions_source_check" check ("source" in ('manual', 'watch'))
);

create index "sessions_user_started_idx" on "public"."sessions" ("user_id", "started_at" desc);

-- Row Level Security: a user may only access their own sessions. The Go backend
-- also enforces ownership in its queries; these policies protect any direct
-- access made with a user's token (e.g. the Supabase client).
alter table "public"."sessions" enable row level security;

create policy "sessions_select_own" on "public"."sessions"
  for select using (auth.uid() = user_id);

create policy "sessions_insert_own" on "public"."sessions"
  for insert with check (auth.uid() = user_id);

create policy "sessions_update_own" on "public"."sessions"
  for update using (auth.uid() = user_id) with check (auth.uid() = user_id);
