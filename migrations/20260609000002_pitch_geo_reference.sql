-- Geo-reference for absolute pitch-position heatmaps.

-- Orientation of the pitch's long axis (own goal -> rival goal), degrees from
-- true north. Combined with latitude/longitude (center) and length_m/width_m it
-- defines an oriented rectangle that GPS paths can be projected onto.
alter table "public"."pitches"
  add column "heading_deg" double precision null;
alter table "public"."pitches"
  add constraint "pitches_heading_deg_check"
  check ("heading_deg" is null or ("heading_deg" >= 0 and "heading_deg" < 360));

-- Denormalized geo-reference snapshot stored on each session, so a recorded GPS
-- path keeps projecting to the same absolute pitch position even if the source
-- pitch is later edited, or was never synced to this user's account.
alter table "public"."sessions"
  add column "pitch_center_lat"  double precision null,
  add column "pitch_center_lon"  double precision null,
  add column "pitch_heading_deg" double precision null,
  add column "pitch_length_m"    double precision null,
  add column "pitch_width_m"     double precision null;
alter table "public"."sessions"
  add constraint "sessions_pitch_heading_deg_check"
  check ("pitch_heading_deg" is null or ("pitch_heading_deg" >= 0 and "pitch_heading_deg" < 360));
