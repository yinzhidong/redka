#!/bin/sh
set -e

bun run dev & ./bin/redka -h 0.0.0.0 -p 6379 redka.db

exec nginx -g 'daemon off;'
