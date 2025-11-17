# ---- Build stage ----
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install build deps
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy and build
COPY . .
RUN go build -o forum main.go

# ---- Runtime stage ----
FROM alpine:latest
WORKDIR /app

# Install runtime deps for SQLite
RUN apk add --no-cache sqlite-libs

# Copy only the built binary from builder
COPY --from=builder /app/forum .

# Expose port and run
EXPOSE 8080
CMD ["./forum"]
