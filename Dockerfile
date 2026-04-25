FROM golang:1.24-alpine AS build

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG BUILD_VERSION
ARG BUILD_DATE

# Two-step build: first binary for SHA256, second with SHA256 embedded.
RUN VERSION=${BUILD_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)} && \
    DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)} && \
    PKG="github.com/DavidArthurCole/EggLedgerSyncServer/handlers" && \
    FLAGS="-s -w -X ${PKG}.BuildVersion=${VERSION} -X ${PKG}.BuildDate=${DATE}" && \
    CGO_ENABLED=0 go build -trimpath -ldflags "${FLAGS}" -o server_unsigned . && \
    SHA256=$(sha256sum server_unsigned | awk '{print $1}') && \
    CGO_ENABLED=0 go build -trimpath \
        -ldflags "${FLAGS} -X ${PKG}.BuildSHA256=${SHA256}" \
        -o server .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /app/server .

EXPOSE 8080

ENTRYPOINT ["./server"]
