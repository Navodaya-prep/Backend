# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server .

# ── Stage 2: Run ──────────────────────────────────────────────────────────────
FROM alpine:3.21

WORKDIR /app

# ca-certificates needed for outbound HTTPS (MongoDB Atlas, SendGrid, etc.)
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server .

# Cloud Run injects PORT at runtime; default 8080 matches main.go fallback
EXPOSE 8080

CMD ["./server"]
