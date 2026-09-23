#!/usr/bin/env bash
# Build a production-shaped schema in an empty database for DB-backed tests.
#
# migrate -dir up alone is not enough: the runner's default exclude list skips
# 007 (which creates kungal_user_state), 005 drops columns later migrations
# restore, and the baseline galgame_contributor shape does not match 069.
# The working order below was verified on 2026-09-18.
#
# Reads TEST_DATABASE_DSN (URL form only). Never prints it.

set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

if [[ -z "${TEST_DATABASE_DSN:-}" ]]; then
	echo "bootstrap: TEST_DATABASE_DSN is not set" >&2
	exit 1
fi

case "${TEST_DATABASE_DSN}" in
postgres://* | postgresql://*) ;;
*)
	echo "bootstrap: TEST_DATABASE_DSN must be URL form (postgres://user@host:port/db?sslmode=disable)" >&2
	exit 1
	;;
esac

export KUN_DATABASE_URL="${TEST_DATABASE_DSN}"
export OAUTH_SERVER_URL="${OAUTH_SERVER_URL:-http://127.0.0.1:9}"
export OAUTH_CLIENT_ID="${OAUTH_CLIENT_ID:-bootstrap}"
export OAUTH_CLIENT_SECRET="${OAUTH_CLIENT_SECRET:-bootstrap}"
export OAUTH_REDIRECT_URI="${OAUTH_REDIRECT_URI:-http://127.0.0.1:9/callback}"
export KUN_NEXTMOE_API_KEY="${KUN_NEXTMOE_API_KEY:-bootstrap}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.1}"

others="$(psql "${TEST_DATABASE_DSN}" -Atqc "SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> '_migrations' ORDER BY 1")"
if [[ -n "${others}" ]]; then
	echo "bootstrap: refusing to run; public schema is not empty (other than _migrations)" >&2
	exit 1
fi

expect_migrate_fail() {
	local needle="$1"
	shift
	local out status
	set +e
	out="$(go run ./cmd/migrate "$@" 2>&1)"
	status=$?
	set -e
	if [[ "${status}" -eq 0 ]]; then
		echo "bootstrap: migrate ${*} succeeded; expected it to fail at ${needle}" >&2
		exit 1
	fi
	if [[ "${out}" != *"${needle}"* ]]; then
		echo "bootstrap: migrate ${*} failed, but not at ${needle}:" >&2
		printf '%s\n' "${out//"${TEST_DATABASE_DSN}"/<TEST_DATABASE_DSN>}" | tail -n 20 >&2
		exit 1
	fi
}

run_migrate() {
	go run ./cmd/migrate "$@"
}

# 1. First pass dies at 053: kungal_user_state is created by 007, which the
#    runner excludes by default.
expect_migrate_fail 053_add_notification_preferences -dir up

# 2. Create kungal_user_state.
run_migrate -only 007

# 3. Second pass dies at 069: baseline created an older galgame_contributor.
expect_migrate_fail 069_galgame_contributor -dir up

# 4. Drop the stale table so 069 can recreate it.
psql "${TEST_DATABASE_DSN}" -v ON_ERROR_STOP=1 -c "DROP TABLE galgame_contributor CASCADE;"

# The runner's own default exclude list, read rather than copied: when a number
# leaves that list, this follows. Step 5 extends it and the freshness check at
# the end skips it, because a deploy-then-drop migration is deliberately applied
# nowhere until its own moment and would fail that check forever.
excluded="$(sed -n 's/.*flag\.String("exclude", "\([^"]*\)".*/\1/p' cmd/migrate/main.go | head -1)"

# 5. Remainder of the default up set, including 069 and 096, but not 141: step 7
#    re-runs 018 against the post-005 shape, and 018 reads
#    galgame_resource.galgame_id, which 141 renames to work_id. With 141 in this
#    pass the bootstrap died at "column r.galgame_id does not exist (SQLSTATE
#    42703)".
#    145 names work_id in its trigger, so it waits for 141 too: in this pass it
#    died at "column \"work_id\" of relation \"galgame_resource\" does not exist
#    (SQLSTATE 42703)".
run_migrate -dir up -exclude "${excluded},141,145"

# 6. Excluded-by-default migrations, one at a time, after 007 exists.
run_migrate -only 005
run_migrate -only 006
run_migrate -only 012
run_migrate -only 015

# 7. 005 drops columns/tables that later migrations restored. Forget those
#    later files so they can run again against the post-005 shape.
psql "${TEST_DATABASE_DSN}" -v ON_ERROR_STOP=1 -c "DELETE FROM _migrations WHERE name ~ '^(018|022|023|069|079|092)_';"
run_migrate -only 018
run_migrate -only 022
run_migrate -only 023
run_migrate -only 069
run_migrate -only 079
run_migrate -only 092

# 8. 141 last, once nothing re-runs against the old column names, then what
#    depends on its work_id.
run_migrate -only 141
run_migrate -only 145

is_excluded() {
	local prefix="${1%%_*}"
	case ",${excluded}," in
	*",${prefix},"*) return 0 ;;
	*) return 1 ;;
	esac
}

shopt -s nullglob
newest=""
for f in migrations/*.up.sql; do
	name="$(basename "${f}" .up.sql)"
	is_excluded "${name}" && continue
	newest="${name}"
done
shopt -u nullglob
if [[ -z "${newest}" ]]; then
	echo "bootstrap: no up migrations found" >&2
	exit 1
fi

applied="$(psql "${TEST_DATABASE_DSN}" -Atqc "SELECT 1 FROM _migrations WHERE name = '${newest}'")"
if [[ "${applied}" != "1" ]]; then
	echo "bootstrap: newest migration ${newest} is not recorded in _migrations" >&2
	exit 1
fi

reg="$(psql "${TEST_DATABASE_DSN}" -Atqc "SELECT to_regclass('public.galgame_contributor')")"
if [[ -z "${reg}" || "${reg}" == "NULL" ]]; then
	echo "bootstrap: galgame_contributor is missing" >&2
	exit 1
fi

vndb="$(psql "${TEST_DATABASE_DSN}" -Atqc "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'galgame' AND column_name = 'vndb_id'")"
if [[ "${vndb}" != "0" ]]; then
	echo "bootstrap: galgame.vndb_id must not exist" >&2
	exit 1
fi
