#!/usr/bin/env bash
set -euo pipefail
umask 077
package_dir="$(cd -- "$(dirname -- "$0")" && pwd)"
target="${1:?Usage: bash update.sh /absolute/path/to/existing-install [compose-project]}"
project="${2:-gradient-monitor}"
target="$(cd -- "$target" && pwd)"
if [[ ! -f "$target/.env" || ! -f "$target/compose.yaml" ]]; then
  echo 'Expected existing .env and compose.yaml in the installation directory.' >&2
  exit 1
fi
if [[ "$target" == "$package_dir" ]]; then
  echo 'Extract the update into a separate directory, not over the installation.' >&2
  exit 1
fi
cd -- "$package_dir"
sha256sum -c images.tar.sha256
compose() { docker compose --project-directory "$target" --env-file "$target/.env" -p "$project" -f "$target/compose.yaml" "$@"; }
new_compose() { docker compose --project-directory "$target" --env-file "$target/.env" -p "$project" -f "$package_dir/compose.yaml" "$@"; }
new_compose config --quiet
db_container="$(compose ps -q db)"
if [[ -z "$db_container" ]]; then
  echo 'Existing database is not running. Check the installation path and Compose project name.' >&2
  exit 1
fi
actual_project="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$db_container")"
if [[ "$actual_project" != "$project" ]]; then echo 'Compose project mismatch.' >&2; exit 1; fi
backup_dir="$target/backups/applications-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p -- "$backup_dir"
compose exec -T db pg_dump -U uamonitor -d uamonitor -Fc > "$backup_dir/database.dump"
test -s "$backup_dir/database.dump"
compose exec -T db pg_restore --list < "$backup_dir/database.dump" > /dev/null
cp -- "$target/compose.yaml" "$backup_dir/compose.yaml"
echo "Database and old Compose backup: $backup_dir"
docker load -i "$package_dir/images.tar"
# Explicit migration run also handles already-exited migrate containers.
new_compose run --rm --no-deps --pull never migrate
cp -- "$package_dir/compose.yaml" "$target/compose.yaml"
compose up -d --no-build --pull never --wait
# Nginx must resolve the new backend/frontend container addresses.
compose restart proxy
compose exec -T proxy nginx -t
compose ps
echo 'Update complete. Verify /add-server, Discord delivery and the private status link.'
echo "Rollback: cp '$backup_dir/compose.yaml' '$target/compose.yaml', then run the old Compose with up -d --pull never --wait and restart proxy."
