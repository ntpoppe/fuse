FROM golang:1.24-alpine AS build

ENV GOTOOLCHAIN=auto

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /fuse ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates sqlite netcat-openbsd

WORKDIR /app

COPY demo/seed/sqlite/shop.sql /tmp/shop.sql
RUN mkdir -p /data && sqlite3 /data/shop.db < /tmp/shop.sql

COPY --from=build /fuse /usr/local/bin/fuse
COPY demo/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
