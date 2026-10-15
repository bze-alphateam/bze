# syntax=docker/dockerfile:1
#
# bzed image. Two users:
# - the interchaintest e2e suite (interchaintest/): built locally with
#   scripts/e2e/build-image.sh (or `make e2e-image`), which passes VERSION and
#   COMMIT from git so `bzed version` reports the tag;
# - nodes run in a container: .github/workflows/docker-image.yml builds every
#   release tag and pushes it to ghcr.io/bze-alphateam/bzed. One image serves
#   every network: the chain picks network-specific behaviour from the chain
#   id at run time, and no Go file depends on a build tag (the Makefile's
#   NETWORK=testnet adds a `testnet` tag nothing reads).

ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-bookworm AS builder

ARG VERSION=unknown
ARG COMMIT=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=readonly -trimpath \
      -ldflags "-X github.com/cosmos/cosmos-sdk/version.Name=bze \
                -X github.com/cosmos/cosmos-sdk/version.AppName=bzed \
                -X github.com/cosmos/cosmos-sdk/version.Version=${VERSION} \
                -X github.com/cosmos/cosmos-sdk/version.Commit=${COMMIT} \
                -w -s" \
      -o /out/bzed ./cmd/bzed

FROM debian:bookworm-slim

# curl + jq: node start scripts read the trust height/hash and the genesis
# from an RPC. ~/.bze exists, owned by bze, so a volume mounted there starts
# out writable by the node.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl jq \
 && rm -rf /var/lib/apt/lists/* \
 && groupadd --gid 1025 bze \
 && useradd --uid 1025 --gid 1025 --create-home --home-dir /home/bze bze \
 && install -d -o bze -g bze /home/bze/.bze

COPY --from=builder /out/bzed /usr/local/bin/bzed

# Last, so a new version does not invalidate the apt layer.
ARG VERSION=unknown
LABEL org.opencontainers.image.version="${VERSION}"

USER 1025:1025
WORKDIR /home/bze

# No ENTRYPOINT: interchaintest supplies the full command line.
CMD ["bzed", "version"]
