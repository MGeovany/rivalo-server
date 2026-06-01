-- Profiles table: one row per authenticated user, keyed by the Supabase auth user id.

create table if not exists public.profiles (
    id                  uuid primary key references auth.users (id) on delete cascade,
    display_name        text        not null default '',
    preferred_position  text,
    height_cm           integer,
    weight_kg           double precision,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now()
);

-- Row Level Security: a user may only read or modify their own profile.
-- The Go backend connects as the database owner (RLS is bypassed there and
-- ownership is enforced in queries); these policies protect any direct access
-- made with a user's token (e.g. the Supabase client).
alter table public.profiles enable row level security;

drop policy if exists "profiles_select_own" on public.profiles;
create policy "profiles_select_own" on public.profiles
    for select using (auth.uid() = id);

drop policy if exists "profiles_insert_own" on public.profiles;
create policy "profiles_insert_own" on public.profiles
    for insert with check (auth.uid() = id);

drop policy if exists "profiles_update_own" on public.profiles;
create policy "profiles_update_own" on public.profiles
    for update using (auth.uid() = id) with check (auth.uid() = id);
