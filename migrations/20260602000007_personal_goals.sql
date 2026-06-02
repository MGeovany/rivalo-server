-- Personal goals (user-defined targets on session metrics).
create table "public"."personal_goals" (
  "id"          uuid        not null default gen_random_uuid(),
  "user_id"     uuid        not null references "public"."profiles" ("id") on delete cascade,
  "metric"      text        not null check ("metric" in ('distance','matches','sprints','rating')),
  "period"      text        not null check ("period" in ('week','month')),
  "target"      numeric     not null check ("target" > 0),
  "created_at"  timestamptz not null default now(),
  "achieved_at" timestamptz null,
  "archived"    boolean     not null default false,
  primary key ("id")
);

alter table "public"."personal_goals" enable row level security;
create policy "goals_select_own" on "public"."personal_goals"
  for select using ("user_id" = auth.uid());
create policy "goals_insert_own" on "public"."personal_goals"
  for insert with check ("user_id" = auth.uid());
create policy "goals_update_own" on "public"."personal_goals"
  for update using ("user_id" = auth.uid());
create policy "goals_delete_own" on "public"."personal_goals"
  for delete using ("user_id" = auth.uid());
