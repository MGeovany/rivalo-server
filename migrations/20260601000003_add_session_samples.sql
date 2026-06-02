-- Per-sample time series for a session (heart rate / speed over time).
create table "public"."session_samples" (
  "session_id" uuid             not null references "public"."sessions" ("id") on delete cascade,
  "t_offset_s" integer          not null,
  "hr"         integer          null,
  "speed_kmh"  double precision null,
  primary key ("session_id", "t_offset_s")
);

-- Row Level Security: a user may only access samples of their own sessions.
alter table "public"."session_samples" enable row level security;

create policy "session_samples_select_own" on "public"."session_samples"
  for select using (
    exists (select 1 from public.sessions s where s.id = session_id and s.user_id = auth.uid())
  );

create policy "session_samples_insert_own" on "public"."session_samples"
  for insert with check (
    exists (select 1 from public.sessions s where s.id = session_id and s.user_id = auth.uid())
  );
