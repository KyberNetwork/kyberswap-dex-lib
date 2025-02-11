# Build
FROM golang:1.23 AS build
ARG GH_USER
WORKDIR /build

RUN --mount=target=.,rw \
    --mount=type=secret,id=gh_pat \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GH_PAT=$(cat /run/secrets/gh_pat) && \
    git config --global url."https://${GH_USER}:${GH_PAT}@github.com/".insteadOf 'https://github.com/' && \
    GOPRIVATE=github.com/KyberNetwork GOOS=linux CGO_LDFLAGS_ALLOW=.* go build -ldflags '-s -w' -tags nethttpomithttp2,go_json -o /out/ ./cmd/app

# Minimal image
FROM alpine:3
WORKDIR /app

RUN apk add --no-cache ca-certificates gcompat libgcc tzdata
ENV CC=gcc
COPY internal/pkg/config/files internal/pkg/config/files
COPY cmd/liquidityscore cmd/liquidityscore
RUN apk add --no-cache py3-scipy && \
    chmod +x cmd/liquidityscore/main.py
COPY --from=build /out/app ./server
CMD ["./server"]
