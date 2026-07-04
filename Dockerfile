FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o /bin/migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/migrate /app/migrate

USER app

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8080/live || exit 1

CMD ["/app/api"]
