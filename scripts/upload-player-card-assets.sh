#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT/.env}"
ASSETS_DIR="${ASSETS_DIR:-$ROOT/../rivalo-ios/Sources/Resources/PlayerCardAssets}"
BUCKET="player-card-assets"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE (need SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY)" >&2
  exit 1
fi

SUPABASE_URL="$(grep '^SUPABASE_URL=' "$ENV_FILE" | cut -d= -f2- | tr -d '"')"
SERVICE_KEY="$(grep '^SUPABASE_SERVICE_ROLE_KEY=' "$ENV_FILE" | cut -d= -f2- | tr -d '"')"

if [[ -z "$SUPABASE_URL" || -z "$SERVICE_KEY" ]]; then
  echo "SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY must be set in $ENV_FILE" >&2
  exit 1
fi

if [[ ! -d "$ASSETS_DIR" ]]; then
  echo "Assets directory not found: $ASSETS_DIR" >&2
  exit 1
fi

upload_file() {
  local object_path="$1"
  local file_path="$2"
  local content_type="$3"

  local url="${SUPABASE_URL%/}/storage/v1/object/${BUCKET}/${object_path}"
  local status
  status="$(curl -sS -o /tmp/rivalo-upload-body.txt -w '%{http_code}' \
    -X POST "$url" \
    -H "Authorization: Bearer ${SERVICE_KEY}" \
    -H "apikey: ${SERVICE_KEY}" \
    -H "Content-Type: ${content_type}" \
    -H "x-upsert: true" \
    --data-binary @"${file_path}")"

  if [[ "$status" != "200" ]]; then
    echo "Upload failed ($status): ${object_path}" >&2
    cat /tmp/rivalo-upload-body.txt >&2
    return 1
  fi
  echo "Uploaded ${object_path}"
}

tiers=(bronze silver gold platinum emerald diamond holographic)
layers=(background frame fx-overlay photo-mask-soft)

for tier in "${tiers[@]}"; do
  for layer in "${layers[@]}"; do
    file="${ASSETS_DIR}/${tier}/${layer}.png"
    if [[ ! -f "$file" ]]; then
      echo "Missing file: $file" >&2
      exit 1
    fi
    upload_file "${tier}/${layer}.png" "$file" "image/png"
  done
done

for meta in layout-guide.json README.md; do
  if [[ -f "${ASSETS_DIR}/${meta}" ]]; then
    upload_file "$meta" "${ASSETS_DIR}/${meta}" "application/json"
  fi
done

for tier in "${tiers[@]}"; do
  manifest="${ASSETS_DIR}/${tier}/manifest.json"
  if [[ -f "$manifest" ]]; then
    upload_file "${tier}/manifest.json" "$manifest" "application/json"
  fi
done

echo "Done. Public base URL:"
echo "${SUPABASE_URL%/}/storage/v1/object/public/${BUCKET}/"
