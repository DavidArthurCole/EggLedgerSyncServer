FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o server .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /app/server .

EXPOSE 8080

ENTRYPOINT ["./server"]
