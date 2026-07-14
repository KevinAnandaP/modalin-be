# Modalin Backend (modalin-be)

Backend API untuk proyek Modalin menggunakan **Golang**, **Fiber v2**, **GORM**, dan **PostgreSQL** dengan struktur proyek berbasis fitur di dalam `internal/` (Standard Go Project Layout).

## Prasyarat
- Go 1.20+
- PostgreSQL

## Struktur Proyek Utama
```text
modalin-be/
├── cmd/
│   └── api/
│       └── main.go       # Entrypoint aplikasi
├── configs/
│   ├── .env              # Variabel lingkungan (lokal)
│   └── .env.example      # Contoh variabel lingkungan
├── internal/
│   ├── health/           # Contoh fitur "health"
│   │   ├── handler/      # Controller/HTTP handler
│   │   ├── service/      # Business logic (folder preserved)
│   │   └── repository/   # Database access (folder preserved)
│   └── router/
│       └── router.go     # Setup routing API
├── pkg/
│   ├── config/           # Helper loader konfigurasi
│   ├── database/         # Helper koneksi database GORM
│   └── utils/            # Utilitas validator dll
├── .dockerignore         # Docker ignore
├── .gitignore            # Git ignore
├── Dockerfile            # Konfigurasi container image backend
├── docker-compose.yml    # Orkestrasi backend & PostgreSQL
├── go.mod
└── go.sum
```

## Cara Menjalankan

1. **Clone repositori** (jika dari repo remote):
   ```bash
   git clone <repository-url>
   cd modalin-be
   ```

2. **Salin & sesuaikan variabel lingkungan**:
   ```bash
   cp configs/.env.example configs/.env
   ```
   *Buka `configs/.env` dan sesuaikan kredensial PostgreSQL Anda.*

3. **Jalankan Aplikasi**:
   ```bash
   go run cmd/api/main.go
   ```

## Endpoint Utama
- **Root API**: `GET http://localhost:8080/`
- **Health Check**: `GET http://localhost:8080/health` atau `GET http://localhost:8080/api/v1/health`
- **Ping Route**: `GET http://localhost:8080/api/v1/ping`

## Aturan Kerja & Standardisasi Route

1. **Prefix Wajib**: Semua endpoint bisnis baru wajib didaftarkan di bawah grup rute `/api/v1` pada file `internal/router/router.go`.
   - *Benar*: `GET /api/v1/users`, `POST /api/v1/investments`
   - *Salah*: `GET /users`, `POST /investments`
2. **Pengecualian**: Hanya endpoint utilitas sistem global (seperti root homepage `/` dan load-balancer health check `/health`) yang diperbolehkan berada di luar prefix `/api/v1`.

---

## Menjalankan dengan Docker Compose

1. **Jalankan Container**:
   ```bash
   docker compose up -d --build
   ```
   *Perintah ini akan membangun image backend Go, mengunduh database PostgreSQL, dan menghubungkan keduanya secara otomatis.*

2. **Matikan Container**:
   ```bash
   docker compose down
   ```
   *Data database Anda akan tetap tersimpan secara aman di volume lokal `postgres_data`.*
