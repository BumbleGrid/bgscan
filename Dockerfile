# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=secret,id=github_token \
    export GOPRIVATE="${GOPRIVATE:-github.com/BumbleGrid/*}" && \
    export GONOSUMDB="${GONOSUMDB:-github.com/BumbleGrid/*}" && \
    export GONOPROXY="${GONOPROXY:-github.com/BumbleGrid/*}" && \
    if [ -f /run/secrets/github_token ]; then \
      git config --global url."https://x-access-token:$(cat /run/secrets/github_token)@github.com/".insteadOf "https://github.com/"; \
    fi && \
    go mod download

ARG VERSION=dev
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/BumbleGrid/bgscan/config.DefaultExtractorVersion=${VERSION}" \
    -o /bgscan .

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /workspace

COPY --from=builder /bgscan /usr/local/bin/bgscan

USER nobody:nobody

ENTRYPOINT ["bgscan"]
