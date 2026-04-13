#!/usr/bin/env bash

set -euo pipefail

if [ $# -ne 1 ]; then
  printf 'usage: %s <version>\n' "$0" >&2
  exit 1
fi

version=${1#v}

perl -0pi -e 's/^  version: ".*" # The application version$/  version: "'"$version"'" # The application version/m' build/config.yml
perl -0pi -e 's/^version: .*/version: '"$version"'/m' build/linux/nfpm/nfpm.yaml
