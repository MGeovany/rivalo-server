-- V3-A: structured post-match result on sessions. All optional, set via
-- PATCH /v1/sessions/{id} like other context fields (does not touch metrics).
alter table "public"."sessions"
  add column "outcome"     text     null,
  add column "score"       text     null,
  add column "competition" text     null,
  add column "goals"       smallint null,
  add column "assists"     smallint null,
  add column "notes"       text     null;

alter table "public"."sessions"
  add constraint "sessions_outcome_check"
    check ("outcome" is null or "outcome" in ('win', 'draw', 'loss')),
  add constraint "sessions_competition_check"
    check ("competition" is null or "competition" in ('friendly', 'league', 'tournament')),
  add constraint "sessions_goals_check"
    check ("goals" is null or "goals" >= 0),
  add constraint "sessions_assists_check"
    check ("assists" is null or "assists" >= 0);
