# =========================
# Build stage
# =========================
FROM golang:1.24.0-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Source
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o seed ./cmd/seed-csv
RUN CGO_ENABLED=0 GOOS=linux go build -o seed-cfit ./cmd/seed-cfit
RUN CGO_ENABLED=0 GOOS=linux go build -o clear-data ./cmd/clear-data


# =========================
# Runtime stage
# =========================
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl netcat-openbsd tzdata

# Copy binaries from builder
COPY --from=builder /app/api .
COPY --from=builder /app/migrate .
COPY --from=builder /app/seed .
COPY --from=builder /app/seed-cfit .
COPY --from=builder /app/clear-data .

# Copy entrypoint script
COPY entrypoint.sh .
RUN sed -i 's/\r$//' entrypoint.sh && chmod +x entrypoint.sh
# RUN chmod +x entrypoint.sh

# Copy necessary resources (fonts, seed data, etc)
COPY fonts ./fonts
# Latar slide PPT; tanpa file ini generate materi gagal ("failed to open background image").
# Di development folder public di-bind-mount, di production harus ada di dalam image.
COPY public/ppt-background.png ./public/ppt-background.png
RUN mkdir -p ./public/materi ./storage
COPY cmd/seed-csv/data ./cmd/seed-csv/data

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

# Use entrypoint script instead of direct command
ENTRYPOINT ["./entrypoint.sh"]
