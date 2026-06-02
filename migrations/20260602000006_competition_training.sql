-- Allow "training" as a competition value (matches match_tag training).
alter table "public"."sessions"
  drop constraint if exists "sessions_competition_check";

alter table "public"."sessions"
  add constraint "sessions_competition_check"
    check ("competition" is null or "competition" in ('friendly', 'league', 'tournament', 'training'));
