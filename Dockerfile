# Vendor stage
FROM golang:1.19 as dep
ARG GH_PAT
WORKDIR /build
COPY go.mod go.sum ./
ENV GOPRIVATE=github.com/KyberNetwork
RUN git config --global url."https://${GH_PAT}:x-oauth-basic@github.com/".insteadOf https://github.com/
RUN GO111MODULE=on go mod download
COPY . .
RUN go mod vendor
# copy missing vendored github.com/KyberNetwork/aevm sources 
RUN cp -r $(go env GOPATH)/pkg/mod/github.com/\!kyber\!network/aevm@$(go list -m github.com/KyberNetwork/aevm | cut -d " " -f 2)/c \
    vendor/github.com/KyberNetwork/aevm
RUN chmod -R +w vendor/github.com/KyberNetwork/aevm

## Lint stage
#FROM golangci/golangci-lint:v1.33.0 as lint
#WORKDIR /build
#COPY --from=dep /build .
#RUN golangci-lint run --verbose --timeout 5m0s

# Build binary stage
FROM golang:1.19-alpine as build
WORKDIR /build
RUN apk update
RUN apk add build-base
COPY --from=dep /build .
RUN CGO_ENABLED=1 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o server -tags nethttpomithttp2 ./cmd/app

# Minimal image
FROM alpine:latest
WORKDIR /app
COPY internal/pkg/config internal/pkg/config
COPY internal/pkg/abis internal/pkg/abis
COPY --from=build /build/server server
RUN apk update
RUN apk upgrade
RUN apk add ca-certificates
RUN apk --no-cache add tzdata
RUN /app/server --help
CMD ["./server"]
