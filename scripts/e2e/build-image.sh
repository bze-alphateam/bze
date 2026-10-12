#!/usr/bin/env bash
# Builds the bzed Docker images used by the interchaintest e2e suite.
#
#   build-image.sh local              build the working tree as bze:local
#   build-image.sh <git-ref> [tag]    build <git-ref> as bze:<tag> (default tag: the ref)
#
# A ref is exported with `git archive`, and the current Dockerfile and
# .dockerignore are copied over it, so refs that predate them build too.
set -euo pipefail

usage() {
  echo "usage: $0 local | <git-ref> [tag]" >&2
  exit 1
}

[ "$#" -ge 1 ] || usage

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

REF="$1"

if [ "$REF" = "local" ]; then
  TAG="${2:-local}"
  VERSION="$(git describe --tags --dirty 2>/dev/null | sed 's/^v//' || echo local)"
  COMMIT="$(git rev-parse HEAD)"
  CONTEXT="$ROOT"
else
  # Docker tags cannot contain '/', so a branch like origin/main becomes origin-main.
  TAG="${2:-${REF//\//-}}"
  COMMIT="$(git rev-parse --verify "${REF}^{commit}")"
  VERSION="$(git describe --tags "$COMMIT" 2>/dev/null | sed 's/^v//' || echo "$COMMIT")"
  CONTEXT="$(mktemp -d)"
  trap 'rm -rf "$CONTEXT"' EXIT
  git archive "$COMMIT" | tar -x -C "$CONTEXT"
  cp "$ROOT/Dockerfile" "$ROOT/.dockerignore" "$CONTEXT/"
fi

echo "Building bze:${TAG} (version ${VERSION}, commit ${COMMIT})"
DOCKER_BUILDKIT=1 docker build \
  --build-arg "VERSION=${VERSION}" \
  --build-arg "COMMIT=${COMMIT}" \
  -t "bze:${TAG}" \
  -f "$CONTEXT/Dockerfile" \
  "$CONTEXT"
