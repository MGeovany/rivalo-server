-- Court extras: indoor flag and free-text notes.
alter table "public"."pitches"
  add column "indoor" boolean null,
  add column "notes"  text    null;
