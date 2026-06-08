-- Add full date of birth to profiles. birth_year is kept (derived from
-- birth_date on write) so the age-based HRmax calculation keeps working.
alter table "public"."profiles"
  add column "birth_date" date null;

alter table "public"."profiles"
  add constraint "profiles_birth_date_check"
  check ("birth_date" is null or ("birth_date" between '1900-01-01' and '2100-12-31'));
