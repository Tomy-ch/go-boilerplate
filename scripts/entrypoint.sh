#!/bin/sh
set -e

echo "Running database migration..."
/app/server migrate-up

if [ $? -ne 0 ]; then
  echo "database migration failed"
  exit 1
fi

echo "Starting server..."
/app/server serve
