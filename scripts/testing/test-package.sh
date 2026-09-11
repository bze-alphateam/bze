#!/usr/bin/env bash
# Optional args: pass a grep -E pattern to select packages. The default targets, under x/, the
# keeper/types/ante packages plus each module's `module` package (genesis import/export tests) and
# its `migrations/vN` packages (store migration tests), and the app/upgrades packages (upgrade
# handlers). Everything the release path depends on runs in CI; simulation and testutil do not.
if [ "$#" -gt 0 ]; then
  PATTERN="$*"
else
  PATTERN='/(keeper|types|ante|module|migrations/v[0-9]+|app/upgrades|app/upgrades/v[0-9]+)$'
fi

# Get packages matching the pattern
# shellcheck disable=SC2207
PACKAGES=($(go list ./x/... ./app/upgrades/... | grep -E "$PATTERN"))

if [ ${#PACKAGES[@]} -eq 0 ]; then
  echo "No keeper or types packages found"
  exit 1
fi

echo "Running tests for:"
printf '%s\n' "${PACKAGES[@]}"
echo ""

# Run tests with verbose output and race detection
go test -v -race -timeout 10m "${PACKAGES[@]}"
