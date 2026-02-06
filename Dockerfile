# Step 1: Build stage
FROM golang:1.25.6-alpine AS builder

# Pasang git karena beberapa modul Go memerlukannya
RUN apk add --no-cache git

WORKDIR /app

# Gunakan Proxy Go agar download lebih stabil
ENV GOPROXY=https://proxy.golang.org,direct

# Copy file modul dulu (untuk memanfaatkan cache layer Docker)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Baru copy seluruh source code
COPY . .

# Build aplikasi dengan flag static agar bisa jalan di alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# Step 2: Final image stage
FROM alpine:latest
# Tambahkan ca-certificates supaya aplikasi bisa akses HTTPS (jika perlu)
RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/main .

COPY .env .

COPY --from=builder /app/docs ./docs

EXPOSE 3000

CMD ["./main"]