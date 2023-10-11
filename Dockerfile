ARG IMG_TAG=latest

# Compile the gaiad binary
FROM golang:1.20-alpine AS gaiad-builder
WORKDIR /src/app/
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ENV PACKAGES curl make git libc-dev bash gcc linux-headers eudev-dev python3
RUN apk add --no-cache $PACKAGES
RUN make build

# Add to a distroless container
FROM cgr.dev/chainguard/static:$IMG_TAG
ARG IMG_TAG
COPY --from=gaiad-builder /src/app/build/bzed /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER nonroot

ENTRYPOINT ["bzed", "start"]