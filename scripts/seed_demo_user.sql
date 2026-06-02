-- Demo seed (SQL only). Prefer: make seed  (creates auth user + data).
--
-- Use this file only if the auth user already exists. Requires psql and DATABASE_URL.
--
--   psql "$DATABASE_URL" -f scripts/seed_demo_user.sql

\set ON_ERROR_STOP on

DO $$
DECLARE
  uid uuid;
  demo_email constant text := 'marlongeo1999+mid@gmail.com';
BEGIN
  SELECT id INTO uid FROM auth.users WHERE email = demo_email;
  IF uid IS NULL THEN
    RAISE EXCEPTION 'No auth user for %. Run: make seed', demo_email;
  END IF;

  INSERT INTO public.profiles (id, display_name, preferred_position, height_cm, weight_kg)
  VALUES (uid, 'Geovany', 'Midfielder', 170, 70)
  ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    preferred_position = EXCLUDED.preferred_position,
    height_cm = EXCLUDED.height_cm,
    weight_kg = EXCLUDED.weight_kg,
    updated_at = now();

  DELETE FROM public.session_samples
  WHERE session_id IN (
    'a1000001-0000-4000-8000-000000000001',
    'a1000001-0000-4000-8000-000000000002',
    'a1000001-0000-4000-8000-000000000003',
    'a1000001-0000-4000-8000-000000000004',
    'a1000001-0000-4000-8000-000000000005'
  );
  DELETE FROM public.sessions
  WHERE id IN (
    'a1000001-0000-4000-8000-000000000001',
    'a1000001-0000-4000-8000-000000000002',
    'a1000001-0000-4000-8000-000000000003',
    'a1000001-0000-4000-8000-000000000004',
    'a1000001-0000-4000-8000-000000000005'
  );

  INSERT INTO public.sessions (
    id, user_id, started_at, ended_at, duration_s, distance_m,
    hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source
  ) VALUES
    ('a1000001-0000-4000-8000-000000000001', uid, now() - interval '35 days', now() - interval '35 days' + interval '82 minutes', 4920, 8120, 138, 162, 24.2, 7, 68.0, 620, 'watch'),
    ('a1000001-0000-4000-8000-000000000002', uid, now() - interval '28 days', now() - interval '28 days' + interval '90 minutes', 5400, 9050, 145, 171, 25.1, 11, 74.0, 710, 'watch'),
    ('a1000001-0000-4000-8000-000000000003', uid, now() - interval '21 days', now() - interval '21 days' + interval '85 minutes', 5100, 7680, 141, 165, 23.4, 6, 70.0, 590, 'watch'),
    ('a1000001-0000-4000-8000-000000000004', uid, now() - interval '14 days', now() - interval '14 days' + interval '88 minutes', 5280, 8340, 148, 175, 25.8, 10, 76.0, 655, 'watch'),
    ('a1000001-0000-4000-8000-000000000005', uid, now() - interval '7 days',  now() - interval '7 days'  + interval '91 minutes', 5460, 9420, 152, 178, 26.3, 12, 81.0, 740, 'watch');

  INSERT INTO public.session_samples (session_id, t_offset_s, hr, speed_kmh)
  SELECT 'a1000001-0000-4000-8000-000000000001', t, 125 + (t / 300) * 8 + ((t % 300) / 50), 8.0 + (t::float / 4920) * 10
  FROM generate_series(0, 4920, 300) AS t;

  INSERT INTO public.session_samples (session_id, t_offset_s, hr, speed_kmh)
  SELECT 'a1000001-0000-4000-8000-000000000002', t, 130 + (t / 300) * 9 + ((t % 300) / 45), 8.5 + (t::float / 5400) * 11
  FROM generate_series(0, 5400, 300) AS t;

  INSERT INTO public.session_samples (session_id, t_offset_s, hr, speed_kmh)
  SELECT 'a1000001-0000-4000-8000-000000000003', t, 128 + (t / 300) * 7 + ((t % 300) / 55), 7.8 + (t::float / 5100) * 10
  FROM generate_series(0, 5100, 300) AS t;

  INSERT INTO public.session_samples (session_id, t_offset_s, hr, speed_kmh)
  SELECT 'a1000001-0000-4000-8000-000000000004', t, 132 + (t / 300) * 10 + ((t % 300) / 48), 9.0 + (t::float / 5280) * 11
  FROM generate_series(0, 5280, 300) AS t;

  INSERT INTO public.session_samples (session_id, t_offset_s, hr, speed_kmh)
  SELECT 'a1000001-0000-4000-8000-000000000005', t, 135 + (t / 300) * 11 + ((t % 300) / 42), 9.2 + (t::float / 5460) * 12
  FROM generate_series(0, 5460, 300) AS t;

  RAISE NOTICE 'Seeded profile + 5 sessions for % (%)', demo_email, uid;
END $$;
