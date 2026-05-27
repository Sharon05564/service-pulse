# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Download dependencies first so Docker can cache this layer separately.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and compile a statically linked binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o servicepulse ./cmd/api

# ── Stage 2: run ──────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# ca-certificates is needed so the health-checker can make HTTPS requests.
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/servicepulse .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./servicepulse"]
