-- Public Supabase Storage bucket for layered player card templates (PNG per tier/layer).
-- Upload with: make upload-player-card-assets (from rivalo-server)

insert into storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
values (
  'player-card-assets',
  'player-card-assets',
  true,
  5242880,
  array['image/png', 'application/json']
)
on conflict (id) do update set
  public = excluded.public,
  file_size_limit = excluded.file_size_limit,
  allowed_mime_types = excluded.allowed_mime_types;

create policy "player_card_assets_public_read"
on storage.objects
for select
to public
using (bucket_id = 'player-card-assets');
