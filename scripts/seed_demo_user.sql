-- Demo seed (SQL only). Prefer: make seed  (creates auth user + rich path/samples).
--
--   psql "$DATABASE_URL" -f scripts/seed_demo_user.sql

\set ON_ERROR_STOP on

DO $$
DECLARE
  uid uuid;
  demo_email constant text := 'appreview@rivalo.app';
  halftime constant int := 2730;
BEGIN
  SELECT id INTO uid FROM auth.users WHERE email = demo_email;
  IF uid IS NULL THEN
    RAISE EXCEPTION 'No auth user for %. Run: make seed', demo_email;
  END IF;

  INSERT INTO public.profiles (id, display_name, preferred_position, height_cm, weight_kg, birth_year)
  VALUES (uid, 'Alex Demo', 'Midfielder', 175, 72, 1995)
  ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    preferred_position = EXCLUDED.preferred_position,
    height_cm = EXCLUDED.height_cm,
    weight_kg = EXCLUDED.weight_kg,
    birth_year = EXCLUDED.birth_year,
    updated_at = now();

  DELETE FROM public.session_path WHERE session_id IN (
    'a1000001-0000-4000-8000-000000000001',
    'a1000001-0000-4000-8000-000000000002',
    'a1000001-0000-4000-8000-000000000003',
    'a1000001-0000-4000-8000-000000000004',
    'a1000001-0000-4000-8000-000000000005'
  );
  DELETE FROM public.session_samples WHERE session_id IN (
    'a1000001-0000-4000-8000-000000000001',
    'a1000001-0000-4000-8000-000000000002',
    'a1000001-0000-4000-8000-000000000003',
    'a1000001-0000-4000-8000-000000000004',
    'a1000001-0000-4000-8000-000000000005'
  );
  DELETE FROM public.sessions WHERE id IN (
    'a1000001-0000-4000-8000-000000000001',
    'a1000001-0000-4000-8000-000000000002',
    'a1000001-0000-4000-8000-000000000003',
    'a1000001-0000-4000-8000-000000000004',
    'a1000001-0000-4000-8000-000000000005'
  );

  INSERT INTO public.sessions (
    id, user_id, started_at, ended_at, duration_s, distance_m,
    hr_avg, hr_max, speed_max_kmh, sprints, intensity, calories_kcal, source,
    mode, match_type, halftime_offset_s, match_rating, position, surface, match_tag, feeling
  ) VALUES
    ('a1000001-0000-4000-8000-000000000001', uid, now() - interval '35 days', now() - interval '35 days' + interval '82 minutes', 4920, 8120, 138, 162, 24.2, 7, 68.0, 620, 'watch', 'quick', '7-a-side', NULL, NULL, 'Midfielder', 'Artificial turf', 'league', 4),
    ('a1000001-0000-4000-8000-000000000002', uid, now() - interval '28 days', now() - interval '28 days' + interval '90 minutes', 5400, 9050, 145, 171, 25.1, 11, 74.0, 710, 'watch', 'quick', '7-a-side', NULL, NULL, 'Midfielder', 'Artificial turf', 'league', 4),
    ('a1000001-0000-4000-8000-000000000003', uid, now() - interval '21 days', now() - interval '21 days' + interval '85 minutes', 5100, 7680, 141, 165, 23.4, 6, 70.0, 590, 'watch', 'quick', '9-a-side', NULL, NULL, 'Midfielder', 'Artificial turf', 'league', 4),
    ('a1000001-0000-4000-8000-000000000004', uid, now() - interval '14 days', now() - interval '14 days' + interval '88 minutes', 5280, 8340, 148, 175, 25.8, 10, 76.0, 655, 'watch', 'structured', '11-a-side', halftime, 72.0, 'Midfielder', 'Artificial turf', 'league', 4),
    ('a1000001-0000-4000-8000-000000000005', uid, now() - interval '7 days',  now() - interval '7 days'  + interval '91 minutes', 5460, 9420, 152, 178, 26.3, 12, 81.0, 740, 'watch', 'structured', '11-a-side', halftime, 78.5, 'Midfielder', 'Artificial turf', 'league', 4);

  -- Dense HR/speed samples every 10s (with half for structured matches).
  INSERT INTO public.session_samples (session_id, t_offset_s, hr, speed_kmh, half)
  SELECT s.id, t,
    LEAST(s.hr_max, s.hr_avg - 18 + (t / 60) * 2 + (t % 45) / 3),
    CASE WHEN (t / 90) % 2 = 0 AND t % 90 < 12 THEN 22.0 + sin(t * 0.3) * 2 ELSE 7.5 + sin(t * 0.004) * 4 END,
    CASE WHEN s.halftime_offset_s IS NOT NULL AND t >= s.halftime_offset_s THEN 2 WHEN s.halftime_offset_s IS NOT NULL THEN 1 ELSE NULL END
  FROM public.sessions s
  CROSS JOIN generate_series(0, s.duration_s, 10) AS t
  WHERE s.id LIKE 'a1000001-%';

  -- GPS path every 5s: pitch X/Y encoded into lat/lon (attack direction → +longitude).
  INSERT INTO public.session_path (session_id, t_offset_s, latitude, longitude)
  SELECT s.id, t,
    14.0723
      + (
        CASE
          WHEN (t::float / s.duration_s) < 0.12 THEN 0.48
          WHEN (t::float / s.duration_s) < 0.28 THEN 0.55 + sin(t * 0.02) * 0.06
          WHEN (t::float / s.duration_s) < 0.45 THEN 0.42
          WHEN (t::float / s.duration_s) < 0.62 THEN 0.38
          WHEN (t::float / s.duration_s) < 0.78 THEN 0.58
          ELSE 0.70
        END
        + CASE WHEN (t::float / s.duration_s) BETWEEN 0.5 AND 0.85 THEN cos(t * 0.028) * 0.07 ELSE 0 END
        + cos(t * 0.09) * 0.04
      ) * 0.00072,
    -87.1921
      + (
        CASE
          WHEN (t::float / s.duration_s) < 0.12 THEN 0.28
          WHEN (t::float / s.duration_s) < 0.28 THEN 0.42
          WHEN (t::float / s.duration_s) < 0.45 THEN 0.58
          WHEN (t::float / s.duration_s) < 0.62 THEN 0.70
          WHEN (t::float / s.duration_s) < 0.78 THEN 0.78
          ELSE 0.82
        END
        + CASE WHEN (t::float / s.duration_s) BETWEEN 0.5 AND 0.85 THEN sin(t * 0.035) * 0.06 + 0.04 ELSE 0 END
        + sin(t * 0.11) * 0.03
      ) * 0.00011
  FROM public.sessions s
  CROSS JOIN generate_series(0, s.duration_s, 5) AS t
  WHERE s.id LIKE 'a1000001-%';

  RAISE NOTICE 'Seeded profile + 5 sessions with path/samples for % (%)', demo_email, uid;
END $$;
