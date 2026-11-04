#!/usr/bin/env bash
# Headlessly install a fresh Matomo (brought up via docker-compose.e2e.yml) and
# seed a superuser API token for the e2e tests. Validated against matomo:5.3.0.
#
# Drives the web installer with a cookie jar (the DB step is pre-filled from the
# MATOMO_DATABASE_* env vars in the compose file), then inserts an app-specific
# token directly into the DB. Matomo stores tokens as sha512(token . salt), and
# we read the salt Matomo generated in config.ini.php, so the seeded token is
# accepted by the Reporting API with no UI token dance.
#
# Emits MATOMO_URL / MATOMO_TOKEN / MATOMO_ADMIN_PASS to stdout, and appends them
# to $GITHUB_OUTPUT and $GITHUB_ENV when running in GitHub Actions.
set -euo pipefail

URL="${MATOMO_URL:-http://localhost:8080}"
ADMIN_LOGIN="${MATOMO_ADMIN_LOGIN:-admin}"
ADMIN_PASS="${MATOMO_ADMIN_PASS:-Sup3rS3cret!}"
COMPOSE="${COMPOSE:-docker compose -f docker-compose.e2e.yml}"
JAR="$(mktemp)"
B="$URL/index.php"

curljar() { curl -fsS -c "$JAR" -b "$JAR" -L "$@"; }

echo "waiting for Matomo at $URL ..."
for i in $(seq 1 60); do
  [ "$(curl -s -o /dev/null -w '%{http_code}' "$URL/" || true)" = "200" ] && break
  sleep 3
done

echo "installer: database setup"
curljar "$B?action=databaseSetup&module=Installation" \
  --data-urlencode "type=InnoDB" \
  --data-urlencode "host=${MATOMO_DB_HOST:-db}" \
  --data-urlencode "username=${MATOMO_DB_USER:-matomo}" \
  --data-urlencode "password=${MATOMO_DB_PASS:-matomo}" \
  --data-urlencode "dbname=${MATOMO_DB_NAME:-matomo}" \
  --data-urlencode "tables_prefix=matomo_" \
  --data-urlencode "adapter=PDO\\MYSQL" \
  --data-urlencode "schema=Mariadb" \
  --data-urlencode "submit=Next" -o /dev/null

echo "installer: superuser"
curljar "$B?action=setupSuperUser&module=Installation" \
  --data-urlencode "login=$ADMIN_LOGIN" \
  --data-urlencode "password=$ADMIN_PASS" \
  --data-urlencode "password_bis=$ADMIN_PASS" \
  --data-urlencode "email=admin@example.com" \
  --data-urlencode "subscribe_newsletter_piwikorg=0" \
  --data-urlencode "subscribe_newsletter_professionalservices=0" \
  --data-urlencode "submit=Next" -o /dev/null

echo "installer: first website (seed, unused by tests)"
curljar "$B?action=firstWebsiteSetup&module=Installation" \
  --data-urlencode "siteName=e2e-seed" \
  --data-urlencode "url=http://seed.example.com" \
  --data-urlencode "timezone=UTC" \
  --data-urlencode "ecommerce=0" \
  --data-urlencode "submit=Next" -o /dev/null

curljar "$B?action=finished&module=Installation" -o /dev/null || true

echo "seeding superuser API token"
SALT="$($COMPOSE exec -T matomo sh -c "grep -E '^salt' /var/www/html/config/config.ini.php" | sed -E 's/.*= *"([^"]+)".*/\1/')"
[ -n "$SALT" ] || { echo "could not read salt from config.ini.php" >&2; exit 1; }
TOKEN="$(openssl rand -hex 16)"
$COMPOSE exec -T db mariadb -u"${MATOMO_DB_USER:-matomo}" -p"${MATOMO_DB_PASS:-matomo}" "${MATOMO_DB_NAME:-matomo}" -e \
  "INSERT INTO matomo_user_token_auth (login, description, password, date_created, system_token, hash_algo, secure_only) VALUES ('$ADMIN_LOGIN','e2e', SHA2(CONCAT('$TOKEN','$SALT'),512), UTC_TIMESTAMP(), 0, 'sha512', 0);"

echo "verifying token"
RESP="$(curl -fsS "$B?module=API&method=API.getMatomoVersion&format=JSON" --data-urlencode "token_auth=$TOKEN")"
case "$RESP" in
  *'"value"'*) echo "token OK: $RESP" ;;
  *) echo "token verification failed: $RESP" >&2; exit 1 ;;
esac

echo "MATOMO_URL=$URL"
echo "MATOMO_TOKEN=$TOKEN"
echo "MATOMO_ADMIN_PASS=$ADMIN_PASS"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  { echo "url=$URL"; echo "token=$TOKEN"; echo "admin_pass=$ADMIN_PASS"; } >>"$GITHUB_OUTPUT"
fi
if [ -n "${GITHUB_ENV:-}" ]; then
  { echo "MATOMO_URL=$URL"; echo "MATOMO_TOKEN=$TOKEN"; echo "MATOMO_ADMIN_PASS=$ADMIN_PASS"; } >>"$GITHUB_ENV"
fi
