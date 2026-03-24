# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    linux-headers

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build with static linking for wasmvm
COPY . .

# Build with CGO for wasmvm support, statically linked against musl
RUN CGO_ENABLED=1 LEDGER_ENABLED=false \
    go build -mod=readonly \
    -ldflags "-X github.com/cosmos/cosmos-sdk/version.Name=bze \
              -X github.com/cosmos/cosmos-sdk/version.AppName=bzed \
              -w -s -linkmode=external -extldflags '-static'" \
    -tags "netgo,muslc" \
    -trimpath \
    -o /src/build/bzed ./cmd/bzed

# Runtime stage
FROM alpine:3

RUN apk add --no-cache \
    ca-certificates \
    curl \
    jq

# Create heighliner user (interchaintest standard)
RUN addgroup -g 1025 -S heighliner && \
    adduser -u 1025 -S heighliner -G heighliner

# Copy statically linked binary
COPY --from=builder /src/build/bzed /usr/local/bin/bzed

# Create data directory for interchaintest
RUN mkdir -p /var/cosmos-chain && chown -R 1025:1025 /var/cosmos-chain

USER 1025:1025
WORKDIR /home/heighliner

ENTRYPOINT ["bzed"]
