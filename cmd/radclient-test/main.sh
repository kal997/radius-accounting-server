#!/bin/bash
set -euo pipefail

: "${RADIUS_SERVER:?RADIUS_SERVER not set}"
: "${RADIUS_SHARED_SECRET:?RADIUS_SHARED_SECRET not set}"

SERVER="${RADIUS_SERVER}:1813"

echo "Running radclient tests against $SERVER ..."

for req in /requests/*.txt; do
  echo "Sending $req ..."
  radclient -x "$SERVER" acct "$RADIUS_SHARED_SECRET" < "$req"
done

echo "All requests sent."
