-- V2 foundations: additive schema only. All columns nullable or defaulted so the
-- V1 client and queries keep working unchanged.

-- 1) Saved pitches (per user). Created before sessions.pitch_id so the FK resolves.
create table "public"."pitches" (
  "id"                 uuid             not null default gen_random_uuid(),
  "user_id"            uuid             not null references "public"."profiles" ("id") on delete cascade,
  "name"               text             not null,
  "latitude"           double precision null,
  "longitude"          double precision null,
  "type"               text             null,
  "surface"            text             null,
  "length_m"           double precision null,
  "width_m"            double precision null,
  "measurement_method" text             null,
  "created_at"         timestamptz      not null default now(),
  "updated_at"         timestamptz      not null default now(),
  primary key ("id"),
  constraint "pitches_type_check" check ("type" is null or "type" in ('5-a-side','7-a-side','9-a-side','11-a-side','Other')),
  constraint "pitches_surface_check" check ("surface" is null or "surface" in ('Natural grass','Artificial turf','Indoor','Concrete','Other')),
  constraint "pitches_measurement_check" check ("measurement_method" is null or "measurement_method" in ('walk','camera','manual'))
);

create index "pitches_user_idx" on "public"."pitches" ("user_id");

alter table "public"."pitches" enable row level security;
create policy "pitches_select_own" on "public"."pitches" for select using (auth.uid() = user_id);
create policy "pitches_insert_own" on "public"."pitches" for insert with check (auth.uid() = user_id);
create policy "pitches_update_own" on "public"."pitches" for update using (auth.uid() = user_id) with check (auth.uid() = user_id);
create policy "pitches_delete_own" on "public"."pitches" for delete using (auth.uid() = user_id);

-- 2) Sessions: V2 columns (mode, optional context, pitch link, halftime, match rating).
--    No effort_level (decision). Existing rows default to mode='quick'.
alter table "public"."sessions"
  add column "mode"              text             not null default 'quick',
  add column "match_type"        text             null,
  add column "surface"           text             null,
  add column "position"          text             null,
  add column "result"            text             null,
  add column "feeling"           integer          null,
  add column "match_tag"         text             null,
  add column "pitch_id"          uuid             null,
  add column "halftime_offset_s" integer          null,
  add column "match_rating"      numeric          null;

alter table "public"."sessions"
  add constraint "sessions_mode_check" check ("mode" in ('quick','structured','training')),
  add constraint "sessions_match_type_check" check ("match_type" is null or "match_type" in ('5-a-side','7-a-side','9-a-side','11-a-side','Other')),
  add constraint "sessions_v2_surface_check" check ("surface" is null or "surface" in ('Natural grass','Artificial turf','Indoor','Concrete','Other')),
  add constraint "sessions_position_check" check ("position" is null or "position" in ('Goalkeeper','Defender','Full-back','Midfielder','Winger','Forward')),
  add constraint "sessions_feeling_check" check ("feeling" is null or ("feeling" between 1 and 5)),
  add constraint "sessions_match_tag_check" check ("match_tag" is null or "match_tag" in ('friendly','league','training')),
  add constraint "sessions_match_rating_check" check ("match_rating" is null or ("match_rating" between 0 and 100)),
  add constraint "sessions_pitch_id_fkey" foreign key ("pitch_id") references "public"."pitches" ("id") on delete set null;

-- 3) Session samples: which half each point belongs to (structured matches).
alter table "public"."session_samples"
  add column "half" smallint null;
alter table "public"."session_samples"
  add constraint "session_samples_half_check" check ("half" is null or "half" in (1, 2));

-- 4) Session path: GPS trajectory (enables pitch normalization, heatmap and zones later).
create table "public"."session_path" (
  "session_id" uuid             not null references "public"."sessions" ("id") on delete cascade,
  "t_offset_s" integer          not null,
  "latitude"   double precision not null,
  "longitude"  double precision not null,
  primary key ("session_id", "t_offset_s")
);

alter table "public"."session_path" enable row level security;
create policy "session_path_select_own" on "public"."session_path"
  for select using (exists (select 1 from public.sessions s where s.id = session_id and s.user_id = auth.uid()));
create policy "session_path_insert_own" on "public"."session_path"
  for insert with check (exists (select 1 from public.sessions s where s.id = session_id and s.user_id = auth.uid()));

-- 5) Profiles: birth year to estimate HRmax for Match Rating (Edwards TRIMP).
alter table "public"."profiles"
  add column "birth_year" integer null;
alter table "public"."profiles"
  add constraint "profiles_birth_year_check" check ("birth_year" is null or ("birth_year" between 1900 and 2100));
