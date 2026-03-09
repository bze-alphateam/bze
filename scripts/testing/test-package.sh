#!/usr/bin/env bash
# Optional args: pass a grep -E pattern to select packages. Default targets keeper/types/ante under x/.
if [ "$#" -gt 0 ]; then
  PATTERN="$*"
else
  PATTERN='/(keeper|types|ante)$'
fi

# Get packages in x/ matching the pattern
# shellcheck disable=SC2207
PACKAGES=($(go list ./x/... | grep -E "$PATTERN"))

if [ ${#PACKAGES[@]} -eq 0 ]; then
  echo "No keeper or types packages found"
  exit 1
fi

echo "Running tests for:"
printf '%s\n' "${PACKAGES[@]}"
echo ""

# Run tests with verbose output and race detection
go test -v -race -timeout 10m "${PACKAGES[@]}"
