# Этап сборки
FROM golang:1.25-alpine3.22 AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY . .
RUN go build -o LIS_app ./cmd

# Финальный образ
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/LIS_app .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY sql/schema ./sql/schema

ENV TZ=UTC
ENV DB_MIGRATION_PATH=/app/sql/schema

CMD goose -dir "$DB_MIGRATION_PATH" postgres "$DATABASE_URL" up && ./LIS_app
