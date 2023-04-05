# Vendor stage
FROM golang:1.19 as dep
WORKDIR /build
COPY go.mod go.sum ./
RUN GO111MODULE=on go mod download
COPY . .
RUN go mod vendor

## Lint stage
#FROM golangci/golangci-lint:v1.33.0 as lint
#WORKDIR /build
#COPY --from=dep /build .
#RUN golangci-lint run --verbose --timeout 5m0s

# Build binary stage
FROM golang:1.19 as build
WORKDIR /build
COPY --from=dep /build .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o server -tags nethttpomithttp2 ./cmd/app

# Minimal image
FROM alpine:latest
WORKDIR /app
COPY internal/pkg/config internal/pkg/config
COPY internal/pkg/abis internal/pkg/abis
COPY internal/pkg/data internal/pkg/data
COPY --from=build /build/server server
RUN apk update
RUN apk upgrade
RUN apk add ca-certificates
RUN apk --no-cache add tzdata
CMD ["./server"]
