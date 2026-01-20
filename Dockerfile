# Stage 1: Builder
FROM golang:1.24-alpine AS builder

# Install build tools
RUN apk add --no-cache git nodejs npm

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum package.json package-lock.json ./

# Download deps
RUN go mod download
RUN npm install --legacy-peer-deps

# Copy source code
COPY . .

# Generate assets
RUN templ generate
RUN npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Stage 2: Runner
FROM alpine:latest

WORKDIR /app

# Copy binary and static assets from builder
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]
