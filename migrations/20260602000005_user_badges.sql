-- Earned achievement badges (one row per user+badge, granted idempotently).
create table "public"."user_badges" (
  "id"        uuid        not null default gen_random_uuid(),
  "user_id"   uuid        not null references "public"."profiles" ("id") on delete cascade,
  "badge_key" text        not null,
  "earned_at" timestamptz not null default now(),
  "meta"      jsonb       null,
  primary key ("id"),
  unique ("user_id", "badge_key")
);

alter table "public"."user_badges" enable row level security;
create policy "user_badges_select_own" on "public"."user_badges"
  for select using ("user_id" = auth.uid());
create policy "user_badges_insert_own" on "public"."user_badges"
  for insert with check ("user_id" = auth.uid());
