#!/bin/sh
set -eu

BASE_DOMAIN="${OSMS_BASE_DOMAIN:-}"
HTTPS_PORT="${OSMS_CADDY_HTTPS_PORT:-443}"

https_public() {
  host="$1"
  if [ -z "$HTTPS_PORT" ] || [ "$HTTPS_PORT" = "443" ]; then
    echo "https://${host}"
  else
    echo "https://${host}:${HTTPS_PORT}"
  fi
}

if [ -n "$BASE_DOMAIN" ]; then
  ORIGIN="$(https_public "$BASE_DOMAIN")"
  PORTAL_URL="${VITE_PORTAL_URL:-${ORIGIN}}"
else
  PORTAL_URL="${VITE_PORTAL_URL:-}"
  if [ -z "$PORTAL_URL" ] && [ -n "${PUBLIC_HOST:-}" ]; then
    PORTAL_URL="http://${PUBLIC_HOST}:5174"
  fi
  PORTAL_URL="${PORTAL_URL:-http://localhost:5174}"
fi

cat > /usr/share/nginx/html/runtime-config.js <<EOF
window.__RUNTIME_CONFIG__ = {
  portalUrl: "${PORTAL_URL}"
};
EOF
