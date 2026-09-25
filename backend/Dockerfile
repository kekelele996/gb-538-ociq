FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app
COPY --from=builder /out/server /usr/local/bin/noisetrace-server
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/noisetrace-server"]
