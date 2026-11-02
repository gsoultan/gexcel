#!/usr/bin/env bash
# Verifies that every hand-written Go file starts with the copyright header.
set -euo pipefail
header='// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>'
status=0
while IFS= read -r -d '' f; do
  case "$f" in
    */testdata/*) continue ;;
  esac
  if [ "$(head -n 1 "$f")" != "$header" ] || [ -n "$(sed -n 2p "$f")" ]; then
    echo "missing or malformed copyright header: $f"
    status=1
  fi
done < <(find . -name '*.go' -not -path './.git/*' -print0)
exit $status
