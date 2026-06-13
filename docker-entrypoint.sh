#!/bin/sh
# Entrypoint for the Postal container.
#
# The "serve" role applies database migrations before starting the API so the
# schema is always current on deploy. The "worker" role skips migrations (it
# starts after serve via depends_on) and any other subcommand passes through.
set -e

if [ "$1" = "serve" ]; then
  if [ -z "$POSTAL_DATABASE_URL" ]; then
    echo "entrypoint: POSTAL_DATABASE_URL is required for the serve role" >&2
    exit 1
  fi
  echo "entrypoint: applying database migrations..."
  goose -dir /app/db/migrations postgres "$POSTAL_DATABASE_URL" up
  echo "entrypoint: migrations up to date."
fi

exec /app/postal "$@"
