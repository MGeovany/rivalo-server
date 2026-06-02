-- Add post-match opponent ("who you played against") to sessions (V2 context).
-- Free text, optional; set via PATCH /v1/sessions/{id} like other context fields.
alter table "public"."sessions"
  add column "opponent" text null;
