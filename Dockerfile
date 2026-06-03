# ════════════════════════════════════════════════════════════════
# Stage 1 — Builder
# Menggunakan golang:alpine untuk compile. Hasilnya HANYA binary.
# ════════════════════════════════════════════════════════════════
FROM golang:1.25-alpine AS builder

# Install dependencies build (git untuk go modules via VCS)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go.mod & go.sum dulu → layer cache lebih efisien.
# Kalau kode berubah tapi dependency tidak, layer ini di-cache.
COPY go.mod go.sum* ./
RUN go mod download && go mod verify

# Copy seluruh source code
COPY . .

# Build binary dengan optimisasi:
#   CGO_ENABLED=0  → binary static (tidak butuh libc di runtime)
#   -ldflags       → strip debug info & DWARF → ukuran binary lebih kecil
#   -trimpath      → hapus path lokal dari binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o /build/app \
    ./cmd/api

# ════════════════════════════════════════════════════════════════
# Stage 2 — Runner (Final Image)
# scratch/alpine: hanya berisi binary + CA certs. Tidak ada shell,
# tidak ada package manager → attack surface minimal.
# ════════════════════════════════════════════════════════════════
FROM alpine:3.20 AS runner

# Tambah ca-certificates untuk HTTPS keluar & tzdata untuk timezone
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser  -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy hanya binary dari stage builder
COPY --from=builder /build/app .

# Jalankan sebagai non-root user (security best practice)
USER appuser

EXPOSE 8080

# Healthcheck — Docker/orchestrator bisa deteksi app tidak sehat
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./app"]
