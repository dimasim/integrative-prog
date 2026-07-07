# Go RESTful API Boilerplate
**Chapter 4 — Building RESTful Web Services with PHP 7** (diimplementasikan dengan Go)

## Tech Stack
- **Runtime**: Go 1.22
- **Framework**: Gin
- **Database**: PostgreSQL (eksternal di VirtualBox VM)
- **Auth**: JWT Bearer Token + Static API Key
- **Infra**: Docker Multi-stage Build (Alpine)

---

## Struktur Proyek
```
go-restapi/
├── cmd/
│   └── api/
│       └── main.go              ← Entry point, wiring DI, server startup
├── internal/
│   ├── config/
│   │   └── config.go            ← Load .env, validasi env vars
│   ├── database/
│   │   └── postgres.go          ← Koneksi PostgreSQL + connection pool
│   ├── domain/
│   │   └── user.go              ← Entity, DTO, sentinel errors
│   ├── handler/
│   │   └── user_handler.go      ← HTTP handler, bind & validate, error mapping
│   ├── middleware/
│   │   ├── auth.go              ← JWT auth, API Key auth, RequireRole
│   │   └── logger.go            ← Zerolog request logger, panic recovery
│   ├── repository/
│   │   └── user_repository.go   ← SQL queries (prepared statements ONLY)
│   └── service/
│       └── user_service.go      ← Business logic, bcrypt, JWT generation
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── scripts/
│   └── migrate.sh               ← Helper script untuk golang-migrate CLI
├── .env                         ← Kredensial (JANGAN di-commit!)
├── .env.example                 ← Template .env yang aman di-commit
├── .gitignore
├── docker-compose.yml
├── Dockerfile                   ← Multi-stage build (builder + runner)
├── go.mod
└── Makefile                     ← Shortcut commands
```

---

## Quick Start

### 1. Setup Environment
```bash
cp .env.example .env
# Edit .env sesuaikan DB_HOST, DB_USER, DB_PASSWORD, JWT_SECRET, API_KEY
```

### 2. Jalankan Migrasi Database
```bash
wget https://go.dev/dl/go1.22.2.linux-amd64.tar.gz

sudo tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz

nano ~/.bashrc

# tambah export PATH=$PATH:/usr/local/go/bin
# install make saya lupa caranya

# Install golang-migrate CLI (sekali saja)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Buat database dulu di PostgreSQL VM
psql -h <VM_IP> -U admin -c "CREATE DATABASE restapi_db;"

# Jalankan migrasi
migrate -path ./migrations -database "postgres://admin:admin123@127.0.0.1:5432/restapi_db?sslmode=disable" up
```

### 3. Development Lokal
```bash
go mod tidy
make run
```

### 4. Docker (Produksi)
```bash
# Build image (multi-stage, hasilnya ~20MB)
make docker-build

# Jalankan via Docker Compose
make compose-up
docker network connect integrative-prog_default @containerpostgres

```

---

## API Endpoints

| Method | Path | Auth | Deskripsi |
|--------|------|------|-----------|
| GET | `/health` | — | Health check |
| POST | `/api/v1/auth/register` | — | Register user baru |
| POST | `/api/v1/auth/login` | — | Login, dapat JWT token |
| GET | `/api/v1/users` | JWT + admin | List semua user |
| GET | `/api/v1/users/:id` | JWT | Get user by ID |
| PATCH | `/api/v1/users/:id` | JWT + admin | Update user |
| DELETE | `/api/v1/users/:id` | JWT + admin | Soft-delete user |
| GET | `/api/internal/users` | API Key | Service-to-service |

---

## Contoh Request

### Register
```bash
curl -X POST http://localhost:8099/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","password":"secret123","role":"user"}'
```

### Login
```bash
curl -X POST http://localhost:8099/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john@example.com","password":"secret123"}'
```

### Get User (dengan JWT)
```bash
curl http://localhost:8099/api/v1/users/1 \
  -H "Authorization: Bearer <token_dari_login>"
```

### API Key (service-to-service)
```bash
curl http://localhost:8099/api/internal/users \
  -H "X-API-Key: your-static-api-key"
```

---

## Security Checklist (Chapter 4)
- [x] **SQL Injection** — semua query pakai `$1, $2` parameterized statements
- [x] **Password** — bcrypt hash, tidak pernah disimpan plain text
- [x] **JWT** — validasi signing method (cegah algorithm confusion attack)
- [x] **API Key** — header `X-API-Key` untuk service-to-service
- [x] **Input Validation** — struct tags `validate:"required,email,min=8"`
- [x] **Error Masking** — sentinel errors dipetakan ke HTTP status, detail DB tidak bocor
- [x] **Non-root Docker** — container berjalan sebagai user `appuser` (UID 1001)
- [x] **Soft Delete** — data tidak benar-benar dihapus, `deleted_at` diisi
- [x] **Graceful Shutdown** — server tunggu request selesai sebelum mati
- [x] **Structured Logging** — zerolog dengan field yang konsisten
# integrative-prog
