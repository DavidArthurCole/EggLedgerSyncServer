FROM cgr.dev/chainguard/go:latest-dev AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o server .

FROM cgr.dev/chainguard/static:nonroot

WORKDIR /app

COPY --from=build /app/server .

EXPOSE 8080

ENTRYPOINT ["./server"]
